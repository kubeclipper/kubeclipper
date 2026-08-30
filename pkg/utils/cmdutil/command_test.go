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
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ec := NewExecCmd(context.Background(), "echo", "hello world!")
	err := ec.Run()
	assert.NoError(t, err)

	errTests := []struct {
		expectedErr error
		*ExecCmd
	}{
		{
			expectedErr: errors.New("exec: \"echo_\": executable file not found in $PATH"),
			ExecCmd:     NewExecCmd(context.Background(), "echo_", "hello, world!"),
		},
	}
	for _, tt := range errTests {
		err := tt.ExecCmd.Run()
		assert.EqualError(t, err, tt.expectedErr.Error(), tt)
	}
}

/* func TestRunCmdWithContext(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer cancel()
	type args struct {
		ctx     context.Context
		command string
		args    []string
	}
	tests := []struct {
		name string
		args args
		//want    bytes.Buffer
		//want1   bytes.Buffer
		wantErr bool
	}{
		{
			name: "run shell command",
			args: args{
				ctx:     context.TODO(),
				command: "ls",
				args:    []string{"/tmp"},
			},
			wantErr: false,
		},
		{
			name: "run shell command with bash",
			args: args{
				ctx:     context.TODO(),
				command: "/bin/bash",
				args:    []string{"-c", "kubectl get po"},
			},
			wantErr: true,
		},
		{
			name: "run shell command with pipeline",
			args: args{
				ctx:     context.TODO(),
				command: "/bin/bash",
				args:    []string{"-c", "kubectl get po || true"},
			},
			wantErr: false,
		},
		{
			name: "run shell command with timeout err",
			args: args{
				ctx:     timeoutCtx,
				command: "/bin/bash",
				args:    []string{"-c", "kubectl get po"},
			},
			wantErr: true,
		},
		{
			name: "run shell scripts",
			args: args{
				ctx:     context.TODO(),
				command: "/bin/bash",
				args: []string{"-c", `cat > k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward=1
EOF
cat >> k8s.conf << EOF
net.ipv6.conf.all.forwarding=1
fs.file-max = 100000
vm.max_map_count=262144
EOF
cat >> limit.conf << EOF
#IncreaseMaximumNumberOfFileDescriptors
* soft nproc 65535
* hard nproc 65535
* soft nofile 65535
* hard nofile 65535
#IncreaseMaximumNumberOfFileDescriptors
EOF
rm k8s.conf
rm limit.conf
`},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RunCmdWithContext(tt.args.ctx, tt.args.command, tt.args.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCmdWithContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
*/

func TestTailBufferKeepsMostRecentTail(t *testing.T) {
	tb := newTailBuffer(16)
	if _, err := tb.Write([]byte("0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	if _, err := tb.Write([]byte("XYZ")); err != nil {
		t.Fatal(err)
	}
	if got := tb.String(); got != "3456789abcdefXYZ" {
		t.Fatalf("tail = %q, want the most recent 16 bytes", got)
	}
}

func TestTailBufferReportsFullWriteWhenInputExceedsLimit(t *testing.T) {
	tb := newTailBuffer(16)
	input := strings.Repeat("x", 32)
	n, err := tb.Write([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if n != len(input) {
		t.Fatalf("Write returned %d, want %d", n, len(input))
	}
	if got := tb.String(); got != input[len(input)-16:] {
		t.Fatalf("tail = %q, want the most recent 16 bytes", got)
	}
}

// A grandchild that inherits the command's stdout pipe and outlives it used to
// block Cmd.Wait forever (single worker wedge). WaitDelay must make Run return
// once the command itself exited.
func TestRunReturnsWhenGrandchildHoldsPipe(t *testing.T) {
	ec := NewExecCmd(context.Background(), "/bin/sh", "-c", "echo started; sleep 30 & exit 0")
	// Shorten the grace so the test does not wait for the 30s default.
	ec.WaitDelay = 500 * time.Millisecond
	start := time.Now()
	err := ec.Run()
	elapsed := time.Since(start)
	if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("Run blocked for %v; WaitDelay did not give up on the pipe", elapsed)
	}
	if !strings.Contains(ec.StdOut(), "started") {
		t.Fatalf("stdout = %q, want the child's output", ec.StdOut())
	}
}
