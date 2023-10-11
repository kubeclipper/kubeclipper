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

package sshutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSHCmd(t *testing.T) {
	defer func() {
		Cmd("rm", "/tmp/d.txt")
		Cmd("rm", "/tmp/dd.txt")
	}()
	ret, err := SSHCmd(nil, "", "bash -c \"echo 123\"")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "123", ret.StdoutToString(""))

	_, err = SSHCmd(nil, "", WrapEcho("12345", "/tmp/d.txt"))
	if err != nil {
		t.Fatal(err)
	}
	ret, err = RunCmdAsSSH("cat /tmp/d.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "12345", ret.StdoutToString(""))

	_, err = SSHCmdWithSudo(nil, "", WrapEcho("54321", "/tmp/dd.txt"))
	if err != nil {
		t.Fatal(err)
	}
	ret, err = RunCmdAsSSH("cat /tmp/dd.txt")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "54321", ret.StdoutToString(""))
}

func TestSSHTable(t *testing.T) {
	ret, err := SSHCmdWithSudo(nil, "", "ls .")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret.Table())
}
