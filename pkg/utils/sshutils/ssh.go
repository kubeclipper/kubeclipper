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
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	User              string         `json:"user" yaml:"user,omitempty"`
	Password          string         `json:"password" yaml:"password,omitempty"`
	Port              int            `json:"port" yaml:"port,omitempty"`
	PkFile            string         `json:"pkFile" yaml:"pkFile,omitempty"`
	PrivateKey        string         `json:"privateKey" yaml:"privateKey,omitempty"`
	PkPassword        string         `json:"pkPassword" yaml:"pkPassword,omitempty"`
	ConnectionTimeout *time.Duration `json:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`
}

func NewSSH() *SSH {
	return &SSH{
		User: "root",
		Port: 22,
	}
}

func (ss *SSH) NewClient(host string) (*ssh.Client, error) {
	return ss.connect(host)
}

func (ss *SSH) connect(host string) (*ssh.Client, error) {
	var (
		pkData []byte
		err    error
	)
	if ss.PkFile != "" {
		pkData, err = os.ReadFile(ss.PkFile)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("privatekey %s not exists", ss.PkFile)
			}
			return nil, errors.WithMessage(err, "read private key")
		}
		ss.PrivateKey = string(pkData)
	}
	auth, err := ss.sshAuthMethod(ss.Password, ss.PrivateKey, ss.PkPassword)
	if err != nil {
		if strings.Contains(err.Error(), "ssh: no key found") {
			return nil, fmt.Errorf("privatekey %s invalid", ss.PkFile)
		}
		if err == x509.IncorrectPasswordError {
			return nil, fmt.Errorf("privatekey %s password incorrect", ss.PkFile)
		}
		return nil, errors.WithMessage(err, "get auth method")
	}
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
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "ssh: handshake failed: ssh: unable to authenticate") {
			err = fmt.Errorf("ssh to %s@%s failed,please check user、password or privatekey", ss.User, host)
		}
		return nil, err
	}
	return client, nil
}

func (ss *SSH) addrReformat(host string) string {
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, ss.Port)
	}
	return host
}

func (ss *SSH) sshAuthMethod(passwd, pkData, pkPasswd string) ([]ssh.AuthMethod, error) {
	auth := make([]ssh.AuthMethod, 0)
	if pkData != "" {
		am, err := ss.sshPrivateKeyMethod([]byte(pkData), pkPasswd)
		if err != nil {
			return nil, err
		}
		auth = append(auth, am)
	}
	if passwd != "" {
		auth = append(auth, ss.sshPasswordMethod(passwd))
	}
	return auth, nil
}

func (ss *SSH) sshPrivateKeyMethod(pkData []byte, pkPassword string) (am ssh.AuthMethod, err error) {
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
