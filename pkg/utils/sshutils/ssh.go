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
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

type SSH struct {
	User              string         `json:"user" yaml:"user,omitempty"`
	Password          string         `json:"password" yaml:"password,omitempty"`
	Port              int            `json:"port" yaml:"port,omitempty"`
	PkFile            string         `json:"pkFile" yaml:"pkFile,omitempty"`
	PkPassword        string         `json:"pkPassword" yaml:"pkPassword,omitempty"`
	ConnectionTimeout *time.Duration `json:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`
}

func (ss *SSH) Connect(host string) (*ssh.Session, error) {
	client, err := ss.connect(host)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// use tty(pty) will redirect stderr to stdout by default
	if err = session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, err
	}

	return session, nil
}

func (ss *SSH) NewClient(host string) (*ssh.Client, error) {
	return ss.connect(host)
}

func (ss *SSH) connect(host string) (*ssh.Client, error) {
	auth := ss.sshAuthMethod(ss.Password, ss.PkFile, ss.PkPassword)
	config := ssh.Config{
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com",
			"arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}
	DefaultTimeout := time.Duration(1) * time.Minute
	if ss.ConnectionTimeout == nil {
		ss.ConnectionTimeout = &DefaultTimeout
	}
	clientConfig := &ssh.ClientConfig{
		User:            ss.User,
		Auth:            auth,
		Timeout:         *ss.ConnectionTimeout,
		Config:          config,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := ss.addrReformat(host)
	return ssh.Dial("tcp", addr, clientConfig)
}

func (ss *SSH) addrReformat(host string) string {
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, ss.Port)
	}
	return host
}

func (ss *SSH) sshAuthMethod(passwd, pkFile, pkPasswd string) (auth []ssh.AuthMethod) {
	if fileExist(pkFile) {
		am, err := ss.sshPrivateKeyMethod(pkFile, pkPasswd)
		if err == nil {
			auth = append(auth, am)
		}
	}
	if passwd != "" {
		auth = append(auth, ss.sshPasswordMethod(passwd))
	}

	return auth
}

func (ss *SSH) sshPrivateKeyMethod(pkFile, pkPassword string) (am ssh.AuthMethod, err error) {
	pkData := ss.readFile(pkFile)
	var pk ssh.Signer
	if pkPassword == "" {
		pk, err = ssh.ParsePrivateKey(pkData)
		if err != nil {
			return nil, err
		}
	} else {
		bufPwd := []byte(pkPassword)
		pk, err = ssh.ParsePrivateKeyWithPassphrase(pkData, bufPwd)
		if err != nil {
			return nil, err
		}
	}
	return ssh.PublicKeys(pk), nil
}

func (ss *SSH) sshPasswordMethod(passwd string) ssh.AuthMethod {
	return ssh.Password(passwd)
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func (ss *SSH) readFile(name string) []byte {
	content, err := ioutil.ReadFile(name)
	if err != nil {
		logger.Errorf("read %s file err is : %s", name, err)
		os.Exit(1)
	}
	return content
}

func (ss *SSH) CmdToString(host, cmd, spilt string) string {
	data := ss.Cmd(host, cmd)

	if data != nil {
		str := string(data)
		str = strings.ReplaceAll(str, "\r\n", spilt)
		return str
	}
	return ""
}

func (ss *SSH) Cmd(host string, cmd string) []byte {
	logger.V(2).Infof("[%s] %s", host, cmd)
	session, err := ss.Connect(host)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[ssh][%s] Error create ssh session failed,%s", host, err)
		}
	}()
	if err != nil {
		panic(1)
	}
	defer session.Close()
	b, err := session.CombinedOutput(cmd)
	logger.V(2).Infof("[%s] command result is: %s", host, string(b))
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[%s] Error exec command failed: %s", host, err)
		}
	}()
	if err != nil {
		panic(1)
	}
	return b
}

func readPipe(host string, pipe io.Reader, isErr bool) {
	r := bufio.NewReader(pipe)
	for {
		line, _, err := r.ReadLine()
		if line == nil {
			return
		} else if err != nil {
			logger.Infof("[%s] %s", host, line)
			logger.Errorf("[%s] %s", host, err)
			return
		} else {
			if isErr {
				logger.Errorf("[%s] %s", host, line)
			} else {
				logger.Infof("[%s] %s", host, line)
			}
		}
	}
}

func (ss *SSH) CmdAsync(host string, cmd string) error {
	logger.V(2).Infof("[%s] %s", host, cmd)
	session, err := ss.Connect(host)
	if err != nil {
		logger.Errorf("[%s] Error create ssh session failed,%s", host, err)
		return err
	}
	defer session.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		logger.Errorf("[%s] Unable to request StdoutPipe(): %s", host, err)
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		logger.Errorf("[%s] Unable to request StderrPipe(): %s", host, err)
		return err
	}
	if err := session.Start(cmd); err != nil {
		logger.Errorf("[%s] Unable to execute command: %s", host, err)
		return err
	}
	doneout := make(chan bool, 1)
	doneerr := make(chan bool, 1)
	go func() {
		readPipe(host, stderr, true)
		doneerr <- true
	}()
	go func() {
		readPipe(host, stdout, false)
		doneout <- true
	}()
	<-doneerr
	<-doneout
	return session.Wait()
}

func (ss *SSH) CmdOutput(host string, cmd string) ([]byte, error) {
	logger.V(2).Infof("[%s] %s", host, cmd)
	session, err := ss.Connect(host)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	b, err := session.CombinedOutput(cmd)
	logger.V(2).Infof("[%s] command result is: %s", host, string(b))

	if err != nil {
		return b, err
	}
	return b, nil
}

func (ss *SSH) CmdExitCode(host string, cmd string) (int, error) {
	logger.V(2).Infof("[%s] %s", host, cmd)
	session, err := ss.Connect(host)
	if err != nil {
		return -1, fmt.Errorf("[ssh][%s] Error create ssh session failed,%s", host, err)
	}
	defer session.Close()

	if err = session.Run(cmd); err != nil {
		if exitError, ok := err.(*ssh.ExitError); ok {
			return exitError.ExitStatus(), nil
		}
	}
	return 0, err
}
