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

package fileutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func WriteFile(path string, flag int, perm os.FileMode, data []byte) error {
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func WriteFileWithDataFunc(path string, flag int, perm os.FileMode, dataFunc func(w io.Writer) error, dryRun bool) error {
	if dryRun {
		return nil
	}
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	return dataFunc(f)
}

func WriteFileWithContext(ctx context.Context, path string, flag int, perm os.FileMode, fn func(w io.Writer) error, dryRun bool) error {
	defer func() {
		ln := fmt.Sprintf("[%s] + rendering %s\n\n", time.Now().Format(time.RFC3339), path)
		if check, err := cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(ln)); err != nil {
			// detect context content and distinguish errors
			if check {
				logger.Error("get operation step log file failed: "+err.Error(),
					zap.String("operation", component.GetOperationID(ctx)),
					zap.String("step", component.GetStepID(ctx)),
					zap.String("cmd", fmt.Sprintf("render %s", path)),
				)
			} else {
				// commands do not need to be logged
				logger.Debug("this command does not need to be logged", zap.String("cmd", fmt.Sprintf("render %s", path)))
			}
		}
	}()
	return WriteFileWithDataFunc(path, flag, perm, fn, dryRun)
}

func Peek(filepath string, offset int64, length int) (data []byte, err error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return
	}
	if length < 0 {
		// If length < 0, read until end of the file.
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(f); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// read length of data
	data = make([]byte, length)
	if _, err = f.Read(data); err != nil {
		return nil, err
	}
	return
}
