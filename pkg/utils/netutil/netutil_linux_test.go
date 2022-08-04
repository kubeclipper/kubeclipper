//go:build linux
// +build linux

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

package netutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultIP(t *testing.T) {
	// ip -4 route get 114.114.114.114
	cmd := exec.Command("bash", "-c", "ip -4 route get 114.114.114.114")
	stdOutBuf := &bytes.Buffer{}
	cmd.Stdout = stdOutBuf
	if err := cmd.Run(); err != nil {
		assert.FailNowf(t, "failed to exec ip r command", err.Error())
	}
	list := strings.Split(stdOutBuf.String(), " ")
	ip, err := GetDefaultIP(true, "")
	if err != nil {
		assert.FailNowf(t, "failed to get default IP", err.Error())
	}
	assert.Equal(t, list[6], ip.To4().String())
}

func TestGetDefaultGateway(t *testing.T) {
	// ip -4 route get 114.114.114.114
	cmd := exec.Command("bash", "-c", "ip -4 route get 114.114.114.114")
	stdOutBuf := &bytes.Buffer{}
	cmd.Stdout = stdOutBuf
	if err := cmd.Run(); err != nil {
		assert.FailNowf(t, "failed to exec ip r command", err.Error())
	}
	list := strings.Split(stdOutBuf.String(), " ")
	gw, err := GetDefaultGateway(true)
	if err != nil {
		assert.FailNowf(t, "failed to get default gateway", err.Error())
	}
	assert.Equal(t, list[2], gw.To4().String())
}
