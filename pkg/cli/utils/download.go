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
	"fmt"
	"path"
	"sync"

	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

// SendPackageV2 scp file to remote host
func SendPackageV2(sshConfig *sshutils.SSH, location string, hosts []string, dstDir string, before, after *string) error {
	var md5 string
	// download pkg to /tmp/kc/
	location, md5, err := downloadFile(location)
	if err != nil {
		return errors.Wrap(err, "downloadFile")
	}
	pkg := path.Base(location)
	// scp to ~/kc/kc-bj-cd.tar.gz
	fullPath := fmt.Sprintf("%s/%s", dstDir, pkg)
	mkDstDir := fmt.Sprintf("mkdir -p %s || true", dstDir)
	var wg sync.WaitGroup
	var errCh = make(chan error, len(hosts))
	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			_, _ = sshutils.SSHCmd(sshConfig, host, mkDstDir)
			logger.V(2).Infof("[%s]please wait for mkDstDir", host)
			if before != nil {
				logger.V(2).Infof("[%s]please wait for before hook", host)
				ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, *before)
				if err != nil {
					errCh <- errors.WithMessage(err, "run before hook")
					return
				}
				if err = ret.Error(); err != nil {
					errCh <- errors.WithMessage(err, "run before hook ret.Error()")
					return
				}
			}
			exists, err := sshConfig.IsFileExistV2(host, fullPath)
			if err != nil {
				errCh <- errors.WithMessage(err, "IsFileExistV2")
				return
			}
			if exists {
				validate, err := sshConfig.ValidateMd5sumLocalWithRemote(host, location, fullPath)
				if err != nil {
					errCh <- errors.WithMessage(err, "ValidateMd5sumLocalWithRemote")
					return
				}

				if validate {
					logger.Infof("[%s]SendPackage:  %s file is exist and ValidateMd5 success", host, fullPath)
				} else {
					// del then copy
					rm := fmt.Sprintf("rm -rf %s", fullPath)
					ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, rm)
					if err != nil {
						logger.Errorf("[%s]remove old file(%s) err %s", host, fullPath, err.Error())
						return
					}
					if err = ret.Error(); err != nil {
						logger.Errorf("[%s]remove old file(%s) err %s", host, fullPath, err.Error())
						return
					}
					ok, err := sshConfig.CopyForMD5V2(host, location, fullPath, md5)
					if err != nil {
						logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, location, err.Error())
						return
					}
					if ok {
						logger.V(2).Infof("[%s]copy file(%s) md5 validate success", host, location)
					} else {
						logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
					}
				}
			} else {
				ok, err := sshConfig.CopyForMD5V2(host, location, fullPath, md5)
				if err != nil {
					logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, fullPath, err.Error())
					return
				}
				if ok {
					logger.V(2).Infof("[%s]copy file(%s) md5 validate success", host, location)
				} else {
					logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
				}
			}

			if after != nil {
				logger.V(2).Infof("[%s]please wait for after hook", host)
				ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, *after)
				if err != nil {
					errCh <- errors.WithMessage(err, "run after hook")
					return
				}
				if err = ret.Error(); err != nil {
					errCh <- errors.WithMessage(err, "run after hook ret.Error()")
					return
				}
			}
		}(host)
	}
	var stopCh = make(chan struct{}, 1)
	go func() {
		defer func() {
			close(stopCh)
			close(errCh)
		}()
		wg.Wait()
		stopCh <- struct{}{}
	}()

	select {
	case err = <-errCh:
		return err
	case <-stopCh:
		return nil
	}
}

func downloadFile(location string) (filePATH, md5 string, err error) {
	if _, ok := httputil.IsURL(location); ok {
		absPATH := "/tmp/kc/" + path.Base(location)
		exist, err := sshutils.IsFileExist(absPATH)
		if err != nil {
			return "", "", err
		}
		if !exist {
			// generator download cmd
			dwnCmd := downloadCmd(location)
			// os exec download command
			sshutils.Cmd("/bin/sh", "-c", "mkdir -p /tmp/kc && cd /tmp/kc && "+dwnCmd)
		}
		location = absPATH
	}
	// file md5
	md5, err = sshutils.MD5FromLocal(location)
	return location, md5, errors.Wrap(err, "MD5FromLocal")
}

// downloadCmd build cmd from url.
func downloadCmd(url string) string {
	// only http
	u, isHTTP := httputil.IsURL(url)
	var c = ""
	if isHTTP {
		param := ""
		if u.Scheme == "https" {
			param = "--no-check-certificate"
		}
		c = fmt.Sprintf(" wget -c %s %s", param, url)
	}
	return c
}
