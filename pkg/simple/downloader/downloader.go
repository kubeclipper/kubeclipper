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

package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
)

const (
	ManifestFilename = "manifest.json"
	ImageFilename    = "images.tar.gz"
	ConfigFilename   = "configs.tar.gz"
	BaseDstDir       = "/tmp/kc-downloader"
	baseManifestDir  = "/opt/kc/manifest"
	ChartFilename    = "charts.tgz"
)

var options *Options

// SetOptions set downloader options
func SetOptions(op *Options) {
	options = op
}

type Downloader struct {
	// k8s component repo
	baseURI string
	// the default directory for storing resources, e.g. /tmp/kc-downloader/.k8s/v1.23.3/amd64
	dstDir string
	// the default directory to the top-level manifest file, e.g. /opt/kc/manifest/k8s/v1.23.3/amd64
	manifestDir string
	// the default directory to the config manifest file, e.g. /opt/kc/manifest/k8s/v1.23.3/amd64/config
	cManifestDir string
	dryRun       bool
	// enable remote download
	// online bool
	// inherits the component context
	ctx context.Context
}

func NewInstance(ctx context.Context, name, version, arch string, online, dryRun bool) (*Downloader, error) {
	if options == nil {
		return nil, fmt.Errorf("the required downloader configuration is missing, you need to call SetOptions before calling NewInstance")
	}
	var baseURI, dstDir, manifestDir, cManifestDir string
	if online {
		baseURI = CloudStaticServer
	} else {
		baseURI = options.Address
	}
	if !dryRun {
		dstDir = filepath.Join(BaseDstDir, "."+name, version)
		manifestDir = filepath.Join(baseManifestDir, name, version, arch)
		cManifestDir = filepath.Join(baseManifestDir, name, version, arch, "config")
		// create required directories
		if err := createDirs(dstDir, manifestDir, cManifestDir); err != nil {
			return nil, err
		}
	}
	return &Downloader{
		ctx:          ctx,
		baseURI:      fmt.Sprintf("%s/%s/%s/%s", baseURI, name, version, arch),
		dryRun:       dryRun,
		dstDir:       dstDir,
		manifestDir:  manifestDir,
		cManifestDir: cManifestDir,
	}, nil
}

// DownloadConfigs download config file
func (dl *Downloader) DownloadConfigs() (string, error) {
	return filepath.Join(dl.dstDir, ConfigFilename), dl.Download(ConfigFilename)
}

// RemoveConfigs remove config file
func (dl *Downloader) RemoveConfigs() error {
	manifest, err := dl.getManifestElements(dl.cManifestDir)
	if err != nil {
		return err
	}
	// clean the list of files in the manifest.json
	for _, f := range manifest {
		p := filepath.Join(f.Path, f.Name)
		if p == "" || p == "/" {
			continue
		}
		if err = os.RemoveAll(p); err == nil {
			logger.Debugf("remove %s successfully", p)
			continue
		}
	}
	// remove configs.tar.gz
	return os.RemoveAll(filepath.Join(dl.dstDir, ConfigFilename))
}

// DownloadImages download image file
func (dl *Downloader) DownloadImages() (string, error) {
	return filepath.Join(dl.dstDir, ImageFilename), dl.Download(ImageFilename)
}

// RemoveImages remove image file
func (dl *Downloader) RemoveImages() error {
	return os.RemoveAll(filepath.Join(dl.dstDir, ImageFilename))
}

// DownloadCharts download chart file
func (dl *Downloader) DownloadCharts() (string, error) {
	return filepath.Join(dl.dstDir, ChartFilename), dl.Download(ChartFilename)
}

// RemoveCharts remove chart file
func (dl *Downloader) RemoveCharts() error {
	return os.RemoveAll(filepath.Join(dl.dstDir, ChartFilename))
}

func (dl *Downloader) GetChartDownloadPath() string {
	return filepath.Join(dl.dstDir, ChartFilename)
}

// DownloadCustomImages download custom image file
func (dl *Downloader) DownloadCustomImages(imageList ...string) (files []string, err error) {
	for _, image := range imageList {
		if err = dl.Download(image); err != nil {
			return
		}
		files = append(files, filepath.Join(dl.dstDir, image))
	}
	return
}

// RemoveCustomImages remove custom image file
func (dl *Downloader) RemoveCustomImages(imageList ...string) (err error) {
	for _, v := range imageList {
		// only the last error is recorded, can ignore it
		err = os.RemoveAll(v)
	}
	return
}

// DownloadAll download config and image file
func (dl *Downloader) DownloadAll() (cPath, iPath string, err error) {
	cPath, err = dl.DownloadAndUnpackConfigs()
	if err != nil {
		return
	}
	iPath, err = dl.DownloadImages()
	if err != nil {
		return
	}
	return
}

// RemoveAll remove config and image file
func (dl *Downloader) RemoveAll() error {
	if err := dl.RemoveConfigs(); err != nil {
		return err
	}
	return dl.RemoveImages()
}

// DownloadAndUnpackConfigs download and unpack config file
func (dl *Downloader) DownloadAndUnpackConfigs() (string, error) {
	fullPath := filepath.Join(dl.dstDir, ConfigFilename)
	if dl.dryRun {
		logger.Debug("install config dry run", zap.String("srcFile", dl.baseURI), zap.String("dstFile", fullPath))
		return fullPath, nil
	}
	// download config file
	if err := dl.Download(ConfigFilename); err != nil {
		return "", err
	}
	// tar zxvf /dst-dir/configs.tar.gz -C /
	_, err := cmdutil.RunCmdWithContext(dl.ctx, dl.dryRun, "bash", "-c", fmt.Sprintf("tar -zxvf %s -C /", filepath.Join(dl.dstDir, ConfigFilename)))
	if err != nil {
		return fullPath, err
	}
	mElements, err := dl.getManifestElements(dl.cManifestDir)
	if err != nil {
		return fullPath, err
	}
	var files []string
	for _, element := range mElements {
		files = append(files, filepath.Join(element.Path, element.Name))
	}
	logger.Debugf("List of files %v to validate, manifest file locate at %s", files, dl.cManifestDir)

	if err = dl.validateMd5Digest(mElements, files); err != nil {
		return fullPath, err
	}
	return fullPath, nil
}

// Download file list
func (dl *Downloader) Download(fileList ...string) (err error) {
	if dl.dryRun {
		logger.Debug("dry run download", zap.String("srcDir", dl.baseURI), zap.String("dstDir", dl.dstDir), zap.Strings("fileList", fileList))
		return
	}
	// top-level manifest file
	mElements, err := dl.getManifestElements(dl.manifestDir)
	if err != nil {
		logger.Errorf("get manifest file failed: %v", err)
		return
	}
	var files []string
	for _, filename := range fileList {
		absolutePath := filepath.Join(dl.dstDir, filename)
		files = append(files, absolutePath)
		// download resource file
		if err = dl.DownloadFile(dl.dstDir, filename); err != nil {
			logger.Errorf("download %s resource file failed: %s", absolutePath)
			return
		}
	}
	if err = dl.validateMd5Digest(mElements, files); err != nil {
		logger.Errorf("check %v digest failed: %v", files, err)
		return
	}
	logger.Debugf("download %v package successfully", files)
	return
}

// validateMd5Digest validates md5 digest of file list
// files param: the value must be an absolute path
func (dl *Downloader) validateMd5Digest(manifest []ManifestElement, files []string) (err error) {
	if len(files) <= 0 {
		logger.Debugf("no list of files to be verified is provided")
		return nil
	}
	// obtaining md5 digest by running the command, md5sum -t /tmp/file1 /tmp/file2
	ec, err := cmdutil.RunCmdWithContext(dl.ctx, dl.dryRun, "bash", "-c", fmt.Sprintf("md5sum -t %s", strings.Join(files, " ")))
	if err != nil {
		return
	}
	// parse the digests line by line
	digests := strings.Split(strings.Trim(ec.StdOut(), "\n"), "\n")
	for _, digest := range digests {
		digestArr := strings.Split(strings.TrimSpace(digest), " ")
		if len(digestArr) < 2 {
			return fmt.Errorf("failed to parse digest information: %s", digest)
		}
		for _, v := range manifest {
			if v.Name == digestArr[1] && v.Digest != digestArr[0] {
				return fmt.Errorf("file %s check md5sum failed", digestArr[1])
			}
		}
	}
	return
}

// getManifestElements get the manifest file and parse it
func (dl *Downloader) getManifestElements(prefix string) (manifest []ManifestElement, err error) {
	filePath := filepath.Join(prefix, ManifestFilename)
	// download top-level manifest file
	if dl.manifestDir == prefix {
		if err = dl.DownloadFile(prefix, ManifestFilename); err != nil {
			logger.Debugf("download [%s] file failed: %v", filePath, err)
			return
		}
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Debugf("read [%s] file contents failed: %v", filePath, err)
		return
	}
	if err = json.Unmarshal(data, &manifest); err != nil {
		return
	}
	logger.Debugf("unmarshal [%s] file contents failed: %v", filePath, err)
	return
}

// DownloadFile download the file to the specified directory
func (dl *Downloader) DownloadFile(dstDir, filename string) (err error) {
	prefix := fmt.Sprintf("backsource.%d-%.3f.", os.Getpid(), float64(time.Now().UnixNano())/float64(time.Second))
	dstFile := path.Join(dstDir, filename)
	file, err := os.CreateTemp(filepath.Dir(dstFile), prefix)
	if err != nil {
		return fmt.Errorf("create temp file failed: %v", err)
	}
	defer file.Close()
	fullURL := fmt.Sprintf("%s/%s", dl.baseURI, filename)
	logger.Debug("start to download file", zap.String("download from", fullURL))
	resp, err := httpGet(fullURL, 0)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()
	defer func() {
		// assembly command
		cmd := fmt.Sprintf("download from %s", fullURL)
		// handle command execution errors
		dealErr := func(e error) (s string) {
			if e != nil {
				return fmt.Sprintf("\n an error occurred: %+v", err)
			}
			return
		}
		ln := fmt.Sprintf("[%s] + %s %s\n\n", time.Now().Format(time.RFC3339), cmd, dealErr(err))
		if check, sErr := cmdutil.CheckContextAndAppendStepLogFile(dl.ctx, []byte(ln)); sErr != nil {
			// detect context content and distinguish errors
			if check {
				logger.Error("get operation step log file failed: "+sErr.Error(),
					zap.String("operation", component.GetOperationID(dl.ctx)),
					zap.String("step", component.GetStepID(dl.ctx)),
					zap.String("cmd", cmd),
				)
			} else {
				// commands do not need to be logged
				logger.Debug("this command does not need to be logged", zap.String("cmd", cmd))
			}
		}
	}()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to download from source, response code: %d", resp.StatusCode)
	}
	buf := make([]byte, 512*1024)
	reader := fileutil.NewFileReader(resp.Body, false)
	if _, err = io.CopyBuffer(file, reader, buf); err != nil {
		return fmt.Errorf("copy buffer error: %d", err)
	}
	return fileutil.MoveFile(file.Name(), dstFile)
}

func httpGet(url string, timeout time.Duration) (resp *http.Response, err error) {
	var (
		cancel func()
		client = &http.Client{}
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	if timeout > 0 {
		timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		req = req.WithContext(timeoutCtx)
		cancel = cancelFunc
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if cancel == nil {
		return res, nil
	}
	res.Body = newWithFuncReadCloser(res.Body, cancel)
	return res, nil
}

type withFuncReadCloser struct {
	f func()
	io.ReadCloser
}

func (wrc *withFuncReadCloser) Close() error {
	if wrc.f != nil {
		wrc.f()
	}
	return wrc.ReadCloser.Close()
}

func newWithFuncReadCloser(rc io.ReadCloser, f func()) io.ReadCloser {
	return &withFuncReadCloser{
		f:          f,
		ReadCloser: rc,
	}
}

func createDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := fileutil.CreateDirIfNotExists(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
