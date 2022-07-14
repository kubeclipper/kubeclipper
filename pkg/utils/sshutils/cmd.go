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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

func IsFileExist(filepath string) (bool, error) {
	// ls -l $dir | grep $name | wc -l
	fileName := path.Base(filepath)
	fileDirName := path.Dir(filepath)
	fileCommand := fmt.Sprintf("ls -l %s | grep %s | wc -l", fileDirName, fileName)
	ret, err := CmdToString("/bin/sh", "-c", fileCommand)
	if err != nil {
		return false, err
	}
	if err = ret.Error(); err != nil {
		return false, err
	}
	data := strings.ReplaceAll(strings.TrimSpace(ret.Stdout), "\r", "")
	data = strings.ReplaceAll(data, "\n", "")
	return data != "0", nil
}

func CmdToString(name string, arg ...string) (Result, error) {
	var ret Result
	logger.Infof("exec cmd is %s %v: ", name, arg)
	cmd := exec.Command(name, arg[:]...)
	ret.PrintCmd = cmd.String()
	cmd.Stdin = os.Stdin
	var bout, berr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &bout, &berr
	err := cmd.Run()
	if err != nil {
		return ret, err
	}
	ret.Stdout = bout.String()
	ret.Stderr = berr.String()
	return ret, nil
}

func Cmd(name string, arg ...string) {
	logger.Infof("exec cmd is %s %v: ", name, arg)
	cmd := exec.Command(name, arg[:]...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logger.Fatal("os call error.", err)
	}
}
