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
	"fmt"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
)

func IsFloatIP(sshConfig *SSH, ip string) (bool, error) {
	// 1.extra all ip
	result, err := SSHCmd(sshConfig, ip, `ifconfig|grep inet|awk {'print $2'}`)
	if err != nil {
		return false, err
	}
	if result.ExitCode != 0 {
		return false, fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
	}
	ips := strings.Split(result.Stdout, "\n")
	ips = sliceutil.RemoveString(ips, func(item string) bool {
		return item == ""
	})
	// 2.check
	return !sliceutil.HasString(ips, ip), nil
}

func GetRemoteHostName(sshConfig *SSH, hostIP string) (string, error) {
	result, err := SSHCmd(sshConfig, hostIP, "hostname")
	if err != nil {
		return "", err
	}
	return strings.ToLower(result.StdoutToString("")), nil
}
