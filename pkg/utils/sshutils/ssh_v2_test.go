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

// const _host = "172.20.151.80"
//
// //const _host = "172.20.150.220"
//
// var (
//	_defaultTimeout = time.Duration(1) * time.Minute
//	sshConfig       = &SSH{
//		User:              "root",
//		Password:          "Thinkbig1",
//		ConnectionTimeout: &_defaultTimeout,
//	}
//	_testSSH = &SSH{
//		User:              "test2",
//		Password:          "root",
//		ConnectionTimeout: &_defaultTimeout,
//	}
// )
//
// func TestSSHCmd(t *testing.T) {
//	result, err := SSHCmd(sshConfig, _host, "ls")
//	if err != nil {
//		t.Fatal(err)
//	}
//	t.Log(result)
// }
//
// func TestCmdPatch(t *testing.T) {
//	walk := func(result Result, err error) error {
//		t.Log(result)
//		t.Log("err: ", err)
//		if err != nil {
//			return err
//		}
//		if result.ExitCode != 0 {
//			return fmt.Errorf("stderr:%s", result.Stderr)
//		}
//		return nil
//	}
//	hosts := []string{
//		//"172.20.149.53",
//		//"172.20.150.220",
//		"172.20.150.200",
//		"172.20.151.80",
//	}
//	// err := CmdBatch(sshConfig, hosts, "ls /tmp2", walk)
//	// err := CmdBatch(sshConfig, hosts, "ls ~", walk)
//	// err := CmdBatch(sshConfig, hosts, "id -u", walk)
//	err := CmdBatchWithSudo(sshConfig, hosts, "id -u", walk)
//	// err := CmdBatch(sshConfig, hosts, "sudo -u root ls /root", walk)
//	// err := CmdBatch(sshConfig, hosts, "ls /tmp2 2>/dev/null", walk)
//	if err != nil {
//		t.Fatal(err)
//	}
//	t.Log("cmd run success")
// }

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
