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

type ExecCmd struct {
	stdOutBuf *bytes.Buffer
	stdErrBuf *bytes.Buffer
	startTime time.Time
	*exec.Cmd
}

func NewExecCmd(ctx context.Context, command string, args ...string) *ExecCmd {
	ec := &ExecCmd{
		stdOutBuf: &bytes.Buffer{},
		stdErrBuf: &bytes.Buffer{},
		Cmd:       exec.CommandContext(ctx, command, args...),
		startTime: time.Now(),
	}
	ec.Cmd.Stdout, ec.Cmd.Stderr = ec.stdOutBuf, ec.stdErrBuf
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
	if ec.Cmd.Process != nil {
		return errors.New("exec: already started")
	}
	ec.Cmd.Env = os.Environ()
	if err := ec.Cmd.Run(); err != nil {
		// errs = append(errs, err)
		// command runs and exits with a non-zero exit status
		// put it into stderr
		_, _ = ec.stdErrBuf.WriteString(err.Error())
		return err
	}
	return nil
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
