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
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

const KB = 1024
const MB = 1024 * 1024

// CopyForMD5V2 copy and check md5
func (ss *SSH) CopyForMD5V2(host, localFilePath, remoteFilePath, localMD5 string) (bool, error) {
	var err error
	if localMD5 == "" {
		localMD5, err = MD5FromLocal(localFilePath)
		if err != nil {
			return false, err
		}
	}
	err = ss.CopySudo(host, localFilePath, remoteFilePath)
	if err != nil {
		return false, err
	}
	remoteMD5, err := ss.MD5FromRemote(host, remoteFilePath)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(localMD5) == strings.TrimSpace(remoteMD5) {
		return true, nil
	}
	return false, nil
}

func (ss *SSH) CopySudo(host, localFilePath, remoteFilePath string) error {
	if ss.User == "root" { // root user,need not transit
		return ss.Copy(host, localFilePath, remoteFilePath)
	}
	// 	if not root,first scp to /tmp,then sudo mv to target
	middle := filepath.Join("/tmp", remoteFilePath)
	err := ss.Copy(host, localFilePath, middle)
	if err != nil {
		return errors.Wrap(err, "copy")
	}
	// TODO maybe need chown
	ret, err := SSHCmdWithSudo(ss, host, fmt.Sprintf("mkdir -pv %s && mv -f %s %s", filepath.Dir(remoteFilePath), middle, remoteFilePath))
	if err != nil {
		return errors.Wrap(err, "mv")
	}
	return errors.Wrap(ret.Error(), "mv")
}

// Copy is
func (ss *SSH) Copy(host, localFilePath, remoteFilePath string) error {
	// do mkdir to ensure remote dir always exists
	ret, err := SSHCmd(ss, host, fmt.Sprintf("mkdir -pv %s", filepath.Dir(remoteFilePath)))
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	// if need run as exec,change to use scp cmd.
	if SSHToCmd(ss, host) {
		ret, err = CmdToString("scp", localFilePath, remoteFilePath)
		if err != nil {
			return err
		}
		return ret.Error()
	}

	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	buf := make([]byte, 100*MB) // 100mb
	total := 0
	unit := ""
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		length, _ := dstFile.Write(buf[0:n])
		isKb := length/MB < 1
		speed := 0
		if isKb {
			total += length
			unit = "KB"
			speed = length / KB
		} else {
			total += length
			unit = "MB"
			speed = length / MB
		}
		totalLength, totalUnit := toSizeFromInt(total)
		logger.Infof("[%s]transfer total size is: %.2f%s ;speed is %d%s", host, totalLength, totalUnit, speed, unit)
	}
	return nil
}

func (ss *SSH) DownloadSudo(host, localFilePath, remoteFilePath string) error {
	if ss.User == "root" { // root user,need not transit
		return ss.download(host, localFilePath, remoteFilePath)
	}
	// 	if not root,first scp to /tmp,then sudo mv to target
	middle := filepath.Join("/tmp", localFilePath)
	err := ss.download(host, middle, remoteFilePath)
	if err != nil {
		return errors.Wrap(err, "download")
	}

	ret, err := SSHCmdWithSudo(ss, host, fmt.Sprintf("mkdir -pv %s && mv -f %s %s", filepath.Dir(localFilePath), middle, localFilePath))
	if err != nil {
		return errors.Wrap(err, "mv")
	}
	return errors.Wrap(ret.Error(), "mv")
}

func (ss *SSH) download(host, localFilePath, remoteFilePath string) error {
	ret, err := CmdToString("mkdir", "-pv", filepath.Dir(localFilePath))
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}

	// if need run as exec,change to use scp cmd.
	if SSHToCmd(ss, host) {
		ret, err = CmdToString("scp", remoteFilePath, localFilePath)
		if err != nil {
			return err
		}
		return ret.Error()
	}

	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	buf := make([]byte, 100*MB) // 100mb
	total := 0
	unit := ""
	for {
		n, _ := srcFile.Read(buf)
		if n == 0 {
			break
		}
		length, _ := dstFile.Write(buf[0:n])
		isKb := length/MB < 1
		speed := 0
		if isKb {
			total += length
			unit = "KB"
			speed = length / KB
		} else {
			total += length
			unit = "MB"
			speed = length / MB
		}
		totalLength, totalUnit := toSizeFromInt(total)
		logger.Infof("[%s]transfer total size is: %.2f%s ;speed is %d%s", host, totalLength, totalUnit, speed, unit)
	}
	return nil
}

// SftpConnect  is
func (ss *SSH) sftpConnect(host string) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = ss.sshAuthMethod(ss.Password, ss.PkFile, ss.PkPassword)

	clientConfig = &ssh.ClientConfig{
		User:    ss.User,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Config: ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
		},
	}

	// connet to ssh
	addr = ss.addrReformat(host)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

func toSizeFromInt(length int) (float64, string) {
	isMb := length/MB > 1
	value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(length)/MB), 64)
	if isMb {
		return value, "MB"
	}
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", float64(length)/KB), 64)
	return value, "KB"

}
