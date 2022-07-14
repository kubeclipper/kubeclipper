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

package utils

import (
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const _host = "172.20.150.220"

var (
	_defaultTimeout = time.Duration(1) * time.Minute
	_ssh            = &sshutils.SSH{
		User:              "test2",
		Password:          "root",
		ConnectionTimeout: &_defaultTimeout,
	}
)

func TestSendPackageV2(t *testing.T) {
	// local := "/Users/lixueduan/17x/tmp/Docker.dmg"
	// TODO: test failed
	//local := "/Users/lixueduan/17x/tmp/nats-streaming-server-v0.24.3-darwin-arm64.tar.gz"
	//remote := "/root/tmp"
	//err := SendPackageV2(_ssh, local, []string{_host}, remote, nil, nil)
	//if err != nil {
	//	t.Fatal(err)
	//}
}
