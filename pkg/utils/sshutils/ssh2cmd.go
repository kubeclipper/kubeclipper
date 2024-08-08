/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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
	"os/exec"
)

// SSHToCmd if caller don't provide enough config for run ssh cmd，change to run cmd by os.exec on localhost.
// for aio deploy now.
func SSHToCmd(sshConfig *SSH, host string) bool {

	ret := host == "" ||
		sshConfig == nil ||
		sshConfig != nil && sshConfig.User == "" || (sshConfig.Password == "" && sshConfig.PkFile == "" && sshConfig.PrivateKey == "")
	return ret
}

func ExtraExitCode(err error) (bool, int) {
	if exiterr, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0
		return true, exiterr.ExitCode()
	}
	return false, -1
}
