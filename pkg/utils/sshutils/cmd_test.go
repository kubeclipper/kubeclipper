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
)

func TestCmdV2(t *testing.T) {
	ret, err := CmdToString("ls", "-al")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ret)
}

func TestIsFileExistV2(t *testing.T) {
	exists, err := IsFileExist("123")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(exists)
}

func TestWhoami(t *testing.T) {
	whoami := Whoami()
	t.Log(whoami)
}

func TestRunCmdAsSSH(t *testing.T) {
	ret, err := RunCmdAsSSH("whoami1")
	if err != nil {
		t.Error(err)
	}
	t.Log(ret.String())
}
