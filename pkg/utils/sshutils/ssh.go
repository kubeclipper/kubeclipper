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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	User              string         `json:"user" yaml:"user,omitempty"`
	Password          string         `json:"password" yaml:"password,omitempty"`
	Port              int            `json:"port" yaml:"port,omitempty"`
	PkFile            string         `json:"pkFile" yaml:"pkFile,omitempty"`
	PkPassword        string         `json:"pkPassword" yaml:"pkPassword,omitempty"`
	ConnectionTimeout *time.Duration `json:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`
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
	pkData, err := ss.readFile(pkFile)
	if err != nil {
		return nil, err
	}
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

func (ss *SSH) readFile(name string) ([]byte, error) {
	content, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return content, nil
}
