/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package cmdutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// commandOutputBufferSize caps the in-memory tail kept per output stream. The
// full stream is persisted to the task log file by the executor; the in-memory
// copy only backs error messages and output parsing, so an unbounded buffer
// would only grow the agent's RSS until a verbose command OOMs it.
const commandOutputBufferSize = 1 << 20 // 1 MiB

// tailBuffer keeps at most max bytes: the most recent tail of the stream.
// Writes beyond the cap drop the oldest bytes in place, so the buffer never
// exceeds max while staying allocation-free in the steady state.
type tailBuffer struct {
	buf   []byte
	limit int
}

func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{buf: make([]byte, 0, limit), limit: limit}
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if t.limit > 0 && len(p) > t.limit {
		p = p[len(p)-t.limit:]
	}
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.limit; over > 0 {
		copy(t.buf, t.buf[over:])
		t.buf = t.buf[:t.limit]
	}
	return n, nil
}

func (t *tailBuffer) Bytes() []byte  { return t.buf }
func (t *tailBuffer) String() string { return string(t.buf) }

type ExecCmd struct {
	stdOutBuf *tailBuffer
	stdErrBuf *tailBuffer
	startTime time.Time
	*exec.Cmd
}

func NewExecCmd(ctx context.Context, command string, args ...string) *ExecCmd {
	ec := &ExecCmd{
		stdOutBuf: newTailBuffer(commandOutputBufferSize),
		stdErrBuf: newTailBuffer(commandOutputBufferSize),
		Cmd:       exec.CommandContext(ctx, command, args...),
		startTime: time.Now(),
	}
	ec.Stdout, ec.Stderr = ec.stdOutBuf, ec.stdErrBuf
	return ec
}

// SetStdoutMultiWriter with multi stdout writer
func (ec *ExecCmd) SetStdoutMultiWriter(writers ...io.Writer) *ExecCmd {
	ec.Cmd.Stdout = io.MultiWriter(append(writers, ec.stdOutBuf)...)
	return ec
}

// SetStderrMultiWriter with multi stderr writer
func (ec *ExecCmd) SetStderrMultiWriter(writers ...io.Writer) *ExecCmd {
	ec.Cmd.Stderr = io.MultiWriter(append(writers, ec.stdErrBuf)...)
	return ec
}

func (ec *ExecCmd) StdOut() string { return ec.stdOutBuf.String() }
func (ec *ExecCmd) StdErr() string { return ec.stdErrBuf.String() }

// String returns a human-readable description of command
func (ec *ExecCmd) CommandString() string { return ec.Cmd.String() }

func (ec *ExecCmd) Run() error {
	if ec.Process != nil {
		return errors.New("exec: already started")
	}
	// Stdout/Stderr are os.Pipes (the buffers plus the log file writer), so
	// Cmd.Wait blocks until every pipe writer closes — including grandchildren
	// that inherited the fds and outlive the command, which would wedge the
	// single task worker forever. WaitDelay gives up on the pipe drains after
	// the command itself has exited; the process-group TERM/KILL path in
	// configureProcessGroup still owns killing a command that hangs.
	if ec.WaitDelay == 0 {
		ec.WaitDelay = commandTerminationGrace
	}
	ec.Env = os.Environ()
	return ec.Cmd.Run()
}

// Marshal merges stdout & stderr and returns bytes slice
func (ec *ExecCmd) Marshal() ([]byte, error) {
	buf := &bytes.Buffer{}

	// format: [2006-01-02T15:04:05Z07:00]: ${command line}
	// e.g., [2006-01-02T15:04:05Z07:00]: systemctl start docker
	buf.WriteString(fmt.Sprintf("[%s]: %s\n", ec.startTime.Format(time.RFC3339), ec.Cmd.String()))

	// copy stdout contents to buffer
	if _, err := buf.Write(ec.stdOutBuf.Bytes()); err != nil {
		return buf.Bytes(), err
	}

	// copy stderr contents to buffer
	if _, err := buf.Write(ec.stdErrBuf.Bytes()); err != nil {
		return buf.Bytes(), err
	}

	return feedLines(buf.Bytes()), nil
}

func feedLines(data []byte) []byte {
	l := len(data)
	// line feed ('\n') hex is 0x0A
	if l >= 2 && data[l-1] == 0x0A && data[l-2] == 0x0A {
		return data
	}
	return feedLines(append(data, 0x0A))
}
