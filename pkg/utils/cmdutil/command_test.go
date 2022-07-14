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
	"testing"

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
