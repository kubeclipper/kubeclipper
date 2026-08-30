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
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

const defaultTempDir = "/tmp"

const (
	stagingDirPerm = 0700
	stagedFilePerm = 0600
)

// SendPackageV2 scp file to remote host
// Deprecated
func SendPackageV2(sshConfig *sshutils.SSH, location string, hosts []string, dstDir string, before, after *string) error {
	return SendPackageV2WithTempDir(sshConfig, location, hosts, dstDir, before, after, defaultTempDir)
}

// SendPackageV2WithTempDir scp a package to remote hosts using tempDir for local staging.
//
//nolint:funlen,gocyclo // preserve the existing transfer state machine while adding configurable staging.
func SendPackageV2WithTempDir(
	sshConfig *sshutils.SSH,
	location string,
	hosts []string,
	dstDir string,
	before, after *string,
	tempDir string,
) error {
	var md5 string
	location, md5, err := downloadFile(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "downloadFile")
	}
	stagedLocation, cleanup, err := stagePackage(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "stagePackage")
	}
	defer cleanup()
	location = stagedLocation
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
						errCh <- errors.WithMessagef(err, "[%s] remove old file %s", host, fullPath)
						return
					}
					if err = ret.Error(); err != nil {
						logger.Errorf("[%s]remove old file(%s) err %s", host, fullPath, err.Error())
						errCh <- errors.WithMessagef(err, "[%s] remove old file %s", host, fullPath)
						return
					}
					ok, err := sshConfig.CopyForMD5V2WithTempDir(host, location, fullPath, md5, tempDir)
					if err != nil {
						logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, location, err.Error())
						errCh <- errors.WithMessagef(err, "[%s] copy %s", host, fullPath)
						return
					}
					if !ok {
						logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
						errCh <- fmt.Errorf("[%s] copy %s md5 validate failed", host, fullPath)
						return
					}
					logger.V(2).Infof("[%s]copy file(%s) md5 validate success", host, location)
				}
			} else {
				ok, err := sshConfig.CopyForMD5V2WithTempDir(host, location, fullPath, md5, tempDir)
				if err != nil {
					logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, fullPath, err.Error())
					errCh <- errors.WithMessagef(err, "[%s] copy %s", host, fullPath)
					return
				}
				if !ok {
					logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
					errCh <- fmt.Errorf("[%s] copy %s md5 validate failed", host, fullPath)
					return
				}
				logger.V(2).Infof("[%s]copy file(%s) md5 validate success", host, fullPath)
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
		// A failure sent just before the last transfer completed may race with
		// the stop signal; drain it instead of reporting success.
		select {
		case err = <-errCh:
			if err != nil {
				return err
			}
		default:
		}
		return nil
	}
}

func downloadFile(location, tempDir string) (filePATH, md5 string, err error) {
	if _, ok := httputil.IsURL(location); ok {
		sourceDir := filepath.Join(tempDir, "kc-source")
		absPATH := filepath.Join(sourceDir, path.Base(location))
		exist, err := sshutils.IsFileExist(absPATH)
		if err != nil {
			return "", "", err
		}
		if !exist {
			// generator download cmd
			dwnCmd := downloadCmd(location)
			// os exec download command
			if downloadErr := sshutils.Cmd("/bin/sh", "-c", "mkdir -p "+sourceDir+" && cd "+sourceDir+" && "+dwnCmd); downloadErr != nil {
				return "", "", downloadErr
			}
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

func SendPackage(sshConfig *sshutils.SSH, location string, hosts []string, dstDir string, before, after *string) error {
	return SendPackageWithTempDir(sshConfig, location, hosts, dstDir, before, after, defaultTempDir)
}

// SendPackageWithTempDir scp a package to remote hosts using tempDir for local staging.
//
//nolint:funlen,gocyclo // preserve the existing transfer state machine while adding configurable staging.
func SendPackageWithTempDir(
	sshConfig *sshutils.SSH,
	location string,
	hosts []string,
	dstDir string,
	before, after *string,
	tempDir string,
) error {
	var md5 string
	location, md5, err := downloadFile(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "downloadFile")
	}
	// Keep the local source separate from the remote extraction directory. A
	// target host may be the machine running kcctl and remove tempDir/kc while
	// other hosts are still copying the package.
	stagedLocation, cleanup, err := stagePackage(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "stagePackage")
	}
	defer cleanup()
	location = stagedLocation
	pkg := path.Base(location)
	// scp to ~/kc/kc-bj-cd.tar.gz
	fullPath := fmt.Sprintf("%s/%s", dstDir, pkg)
	mkDstDir := fmt.Sprintf("mkdir -p %s || true", dstDir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	var errCh = make(chan error, len(hosts))
	defer close(errCh)

	wg.Add(len(hosts))
	p := mpb.NewWithContext(ctx, mpb.WithWidth(60), mpb.WithWaitGroup(&wg))

	for _, host := range hosts {
		task := fmt.Sprintf("%s:", host)
		//queue := make([]*mpb.Bar, 4)

		hookbar := p.Add(0,
			mpb.NopStyle().Build(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.Name("run before hook..."),
			),
		)

		checkfilebar := p.Add(0,
			mpb.NopStyle().Build(),
			mpb.BarQueueAfter(hookbar, false),
			mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.Name("check file exist..."),
			),
		)

		checkmd5bar := p.Add(0,
			mpb.NopStyle().Build(),
			mpb.BarQueueAfter(checkfilebar, false),
			mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.Name("check file md5..."),
			),
		)

		downloadbar := p.Add(0,
			mpb.BarStyle().Rbound("|").Build(),
			mpb.BarQueueAfter(checkmd5bar, false),
			//mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.Name("downloading", decor.WCSyncSpaceR),
				decor.Percentage(decor.WCSyncSpace),
				decor.Name(" | "),
				decor.CountersKibiByte("% .2f / % .2f"),
				//decor.OnComplete(decor.Name("\x1b[31mdownloading\x1b[0m", decor.WCSyncSpaceR), "done!"),
				//decor.OnComplete(decor.EwmaETA(decor.ET_STYLE_MMSS, 0, decor.WCSyncWidth), ""),
			),
			mpb.AppendDecorators(
				// replace ETA decorator with "done" message, OnComplete event
				//decor.OnComplete(
				//	// ETA decorator with ewma age of 60
				//	decor.EwmaETA(decor.ET_STYLE_GO, 60, decor.WCSyncWidth), "done",
				//),
				//decor.EwmaETA(decor.ET_STYLE_GO, 90),
				//decor.EwmaETA(decor.ET_STYLE_GO, 60),
				//decor.Name(" ] "),
				decor.AverageSpeed(decor.UnitKiB, "% .2f", decor.WCSyncSpace),
				//decor.EwmaSpeed(decor.UnitKiB, "% .2f", 60),
			),
		)

		checkmd5afterdownloadbar := p.Add(0,
			mpb.NopStyle().Build(),
			mpb.BarQueueAfter(downloadbar, false),
			mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.Name("check file md5..."),
			),
		)

		afterhookbar := p.Add(0,
			mpb.NopStyle().Build(),
			mpb.BarQueueAfter(checkmd5afterdownloadbar, false),
			mpb.BarFillerClearOnComplete(),
			mpb.PrependDecorators(
				decor.Name(task, decor.WC{W: len(task) + 1, C: decor.DidentRight}),
				decor.OnComplete(decor.Name("run after hook...", decor.WCSyncSpaceR), "done!"),
			),
		)

		go func(host string) {
			defer wg.Done()

			_, err := sshutils.SSHCmd(sshConfig, host, mkDstDir)
			if err != nil {
				errCh <- errors.WithMessage(err, "mkdir dst")
				return
			}
			if before != nil {
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
			hookbar.SetTotal(-1, true)

			exist, err := sshConfig.IsFileExistV2(host, fullPath)
			if err != nil {
				errCh <- errors.WithMessage(err, "IsFileExistV2")
				return
			}
			checkfilebar.SetTotal(-1, true)

			if exist {
				removeMD5, err := sshConfig.MD5FromRemote(host, fullPath)
				if err != nil {
					errCh <- errors.WithMessage(err, "check remote file MD5")
					return
				}
				if md5 == removeMD5 {
					checkmd5bar.SetTotal(-1, true)
					downloadbar.SetTotal(-1, true)
					checkmd5afterdownloadbar.SetTotal(-1, true)
				} else {
					rm := fmt.Sprintf("rm -rf %s", fullPath)
					ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, rm)
					if err != nil {
						errCh <- errors.WithMessage(err, "remove remote old files")
						return
					}
					if err = ret.Error(); err != nil {
						errCh <- errors.WithMessage(err, "remove remote old files")
						return
					}
					checkmd5bar.SetTotal(-1, true)
					if copyErr := sshConfig.CopySudoWithBarWithTempDir(downloadbar, host, location, fullPath, tempDir); copyErr != nil {
						errCh <- errors.WithMessage(copyErr, "download")
						return
					}
					if downloadbar.IsRunning() {
						downloadbar.SetTotal(-1, true)
					}
					remoteMD5, err := sshConfig.MD5FromRemote(host, fullPath)
					if err != nil {
						errCh <- errors.WithMessage(err, "check remote file MD5")
						return
					}
					if md5 == remoteMD5 {
						checkmd5afterdownloadbar.SetTotal(-1, true)
					} else {
						errCh <- errors.WithMessage(err, "check remote file MD5")
						return
					}
				}
			} else {
				checkmd5bar.SetTotal(-1, true)
				if err := sshConfig.CopySudoWithBarWithTempDir(downloadbar, host, location, fullPath, tempDir); err != nil {
					errCh <- errors.WithMessage(err, "download")
					return
				}
				if downloadbar.IsRunning() {
					downloadbar.SetTotal(-1, true)
				}
				remoteMD5, err := sshConfig.MD5FromRemote(host, fullPath)
				if err != nil {
					errCh <- errors.WithMessage(err, "check remote file MD5")
					return
				}
				if md5 == remoteMD5 {
					checkmd5afterdownloadbar.SetTotal(-1, true)
				} else {
					errCh <- errors.WithMessage(err, "check remote file MD5")
					return
				}
			}

			if after != nil {
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
			afterhookbar.SetTotal(-1, true)
		}(host)
	}
	var stopCh = make(chan struct{})
	go func() {
		defer close(stopCh)
		p.Wait()
	}()

	select {
	case err = <-errCh:
		return err
	case <-stopCh:
		return nil
	}
}

// Keep staging isolated from the remote extraction directory.
func stagePackage(location, tempDir string) (stagedPath string, cleanup func(), err error) {
	sourceDir := filepath.Join(tempDir, "kc-source")
	if mkdirErr := os.MkdirAll(sourceDir, stagingDirPerm); mkdirErr != nil {
		return "", func() {}, mkdirErr
	}
	dir, err := os.MkdirTemp(sourceDir, "stage-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }
	staged := filepath.Join(dir, path.Base(location))

	if linkErr := os.Link(location, staged); linkErr == nil {
		return staged, cleanup, nil
	}

	src, err := os.Open(location)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	defer src.Close()
	dst, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, stagedFilePerm)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if _, err = io.Copy(dst, src); err != nil {
		_ = dst.Close()
		cleanup()
		return "", func() {}, err
	}
	if err = dst.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return staged, cleanup, nil
}

func SendPackageLocal(location string, dstDir string, after *string) error {
	return SendPackageLocalWithTempDir(location, dstDir, after, defaultTempDir)
}

// SendPackageLocalWithTempDir copies a package locally using tempDir for local staging.
func SendPackageLocalWithTempDir(location, dstDir string, after *string, tempDir string) error {
	location, _, err := downloadFile(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "downloadFile")
	}
	stagedLocation, cleanup, err := stagePackage(location, tempDir)
	if err != nil {
		return errors.Wrap(err, "stagePackage")
	}
	defer cleanup()
	location = stagedLocation

	mkDstDir := fmt.Sprintf("mkdir -p %s || true", dstDir)
	if err = sshutils.Cmd("/bin/sh", "-c", mkDstDir); err != nil {
		return err
	}

	fullPath := fmt.Sprintf("%s/%s", dstDir, path.Base(location))
	// remove if exist
	rm := fmt.Sprintf("rm -rf %s || true", fullPath)
	if err = sshutils.Cmd("/bin/sh", "-c", rm); err != nil {
		return err
	}
	// move to dst
	cp := fmt.Sprintf("cp %s %s", location, fullPath)
	if err = sshutils.Cmd("/bin/sh", "-c", cp); err != nil {
		return err
	}

	if after != nil {
		ret, err := sshutils.RunCmdAsSSH(*after)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return errors.WithMessage(err, "run after hook")
		}
	}

	return nil
}
