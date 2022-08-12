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
	"net"
	"os"
	"os/exec"
	"path"
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

func md5sumLocal(localPath string) string {
	cmd := fmt.Sprintf("md5sum %s | cut -d\" \" -f1", localPath)
	c := exec.Command("sh", "-c", cmd)
	out, err := c.CombinedOutput()
	if err != nil {
		logger.Errorf("exec cmd failed due to %s", err.Error())
	}
	md5 := string(out)
	md5 = strings.ReplaceAll(md5, "\n", "")
	md5 = strings.ReplaceAll(md5, "\r", "")

	return md5
}

func (ss *SSH) CopyForMD5(host, localFilePath, remoteFilePath, md5 string) bool {
	if md5 == "" {
		md5 = md5sumLocal(localFilePath)
	}
	logger.V(2).Infof("source file md5 value is %s", md5)
	_ = ss.Copy(host, localFilePath, remoteFilePath)
	remoteMD5, _ := ss.MD5FromRemote(host, remoteFilePath)
	logger.V(2).Infof("host: %s , remote md5: %s", host, remoteMD5)
	remoteMD5 = strings.TrimSpace(remoteMD5)
	md5 = strings.TrimSpace(md5)
	if remoteMD5 == md5 {
		logger.Info("md5 validate true")
		return true
	}
	logger.Error("md5 validate false")
	return false
}

// CopyForMD5V2 copy and check md5
func (ss *SSH) CopyForMD5V2(host, localFilePath, remoteFilePath, localMD5 string) (bool, error) {
	var err error
	if localMD5 == "" {
		localMD5, err = MD5FromLocal(localFilePath)
		if err != nil {
			return false, err
		}
	}
	logger.V(3).Infof("source file(%s) md5 value is %s", localFilePath, localMD5)
	err = ss.CopySudo(host, localFilePath, remoteFilePath)
	if err != nil {
		return false, err
	}
	remoteMD5, err := ss.MD5FromRemote(host, remoteFilePath)
	if err != nil {
		return false, err
	}
	logger.V(3).Infof("[%s]remote file(%s) md5 value is %s", host, remoteFilePath, remoteMD5)
	if strings.TrimSpace(localMD5) == strings.TrimSpace(remoteMD5) {
		logger.V(4).Infof("md5 validate true localMd5:%s remoteMd5:%s", localMD5, remoteMD5)
		return true, nil
	}
	logger.Errorf("md5 validate false localMd5:%s remoteMd5:%s", localMD5, remoteMD5)
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
	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		logger.Fatalf("[%s]sftp conn failed: %s", host, err)
	}
	defer sftpClient.Close()
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		logger.Fatalf("[%s]open local file %s failed: %s", host, localFilePath, err)
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		logger.Fatalf("[%s]open remote file %s failed: %s", host, remoteFilePath, err)

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
	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		logger.Fatalf("[%s]sftp conn failed: %s", host, err)
	}
	defer sftpClient.Close()
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		logger.Fatalf("[%s]open remote file %s failed: %s", host, remoteFilePath, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFilePath)
	if err != nil {
		logger.Fatalf("[%s]create local file %s failed: %s", host, localFilePath, err)
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

// CopyRemoteFileToLocal is scp remote file to local
func (ss *SSH) CopyRemoteFileToLocal(host, localFilePath, remoteFilePath string) {
	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		logger.Fatalf("[%s]sftp conn failed: %s", host, err)
	}
	defer sftpClient.Close()
	// open remote source file
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		logger.Fatalf("[%s]sftp open remote file %s failed: %s", host, remoteFilePath, err)
	}
	defer srcFile.Close()

	// open local Destination file
	dstFile, err := os.Create(localFilePath)
	if err != nil {
		logger.Fatalf("[%s]sftp open local file %s failed: %s", host, localFilePath, err)

	}
	defer dstFile.Close()
	// copy to local file
	_, _ = srcFile.WriteTo(dstFile)
}

// CopyLocalToRemote is copy file or dir to remotePath, add md5 validate
func (ss *SSH) CopyLocalToRemote(host, localPath, remotePath string) {
	sftpClient, err := ss.sftpConnect(host)
	if err != nil {
		logger.Fatalf("[%s]sftp conn failed: %s", host, err)
	}
	defer sftpClient.Close()
	sshClient, err := ss.connect(host)
	if err != nil {
		logger.Fatalf("[%s]ssh conn failed: %s", host, err)
	}
	defer sshClient.Close()
	s, _ := os.Stat(localPath)
	if s.IsDir() {
		ss.copyLocalDirToRemote(host, sshClient, sftpClient, localPath, remotePath)
	} else {
		baseRemoteFilePath := filepath.Dir(remotePath)
		mkDstDir := fmt.Sprintf("mkdir -p %s || true", baseRemoteFilePath)
		_ = ss.CmdAsync(host, mkDstDir)
		ss.copyLocalFileToRemote(host, sshClient, sftpClient, localPath, remotePath)
	}
}

// ssh session is a problem, 复用 ssh 链接
func (ss *SSH) copyLocalDirToRemote(host string, sshClient *ssh.Client, sftpClient *sftp.Client, localPath, remotePath string) {
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		logger.Fatalf("readDir err : %s", err)
	}
	_ = sftpClient.Mkdir(remotePath)
	for _, file := range localFiles {
		lfp := path.Join(localPath, file.Name())
		rfp := path.Join(remotePath, file.Name())
		if file.IsDir() {
			_ = sftpClient.Mkdir(rfp)
			ss.copyLocalDirToRemote(host, sshClient, sftpClient, lfp, rfp)
		} else {
			ss.copyLocalFileToRemote(host, sshClient, sftpClient, lfp, rfp)
		}
	}
}

// solve the session
func (ss *SSH) copyLocalFileToRemote(host string, sshClient *ssh.Client, sftpClient *sftp.Client, localPath, remotePath string) {
	srcFile, err := os.Open(localPath)
	if err != nil {
		logger.Fatalf("open file [%s] err : %s", localPath, err)
	}
	defer srcFile.Close()
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		logger.Error("err:", err)
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
		logger.V(2).Infof("[%s]transfer local [%s] to Dst [%s] total size is: %.2f%s ;speed is %d%s", host, localPath, remotePath, totalLength, totalUnit, speed, unit)
	}
	if !ss.isCopyMd5Success(sshClient, localPath, remotePath) {
		//	logger.Debug("[ssh][%s] copy local file: %s to remote file: %s validate md5sum success", host, localPath, remotePath)
		// } else {
		logger.Errorf("[%s] copy local file: %s to remote file: %s validate md5sum failed", host, localPath, remotePath)
	}
}

func (ss *SSH) isCopyMd5Success(sshClient *ssh.Client, localFile, remoteFile string) bool {
	cmd := fmt.Sprintf("md5sum %s | cut -d\" \" -f1", remoteFile)
	localMd5 := md5sumLocal(localFile)
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return false
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := sshSession.RequestPty("xterm", 80, 40, modes); err != nil {
		return false
	}

	if sshSession.Stdout != nil {
		sshSession.Stdout = nil
		sshSession.Stderr = nil
	}
	b, err := sshSession.CombinedOutput(cmd)
	if err != nil {
		logger.Fatalf("Error exec command failed: %s", err)
	}
	var remoteMd5 string
	if b != nil {
		remoteMd5 = string(b)
		remoteMd5 = strings.ReplaceAll(remoteMd5, "\r\n", "")
	}
	return localMd5 == remoteMd5
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
