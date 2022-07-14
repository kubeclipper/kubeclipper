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

package oplog

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
)

const OperationLogSuffix = ".log"

var _ component.OperationLogFile = (*OperationLog)(nil)

type OperationLog struct {
	cfg    *Options
	suffix string
}

// NewOperationLog create the operation log directory on startup.
// currently log file suffix is .log and cannot be changed.
func NewOperationLog(opts *Options) (component.OperationLogFile, error) {
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, err
	}

	return &OperationLog{
		cfg:    opts,
		suffix: OperationLogSuffix,
	}, nil
}

// GetRootDir get operation log root dir
func (op *OperationLog) GetRootDir() string {
	return op.cfg.Dir
}

// CreateOperationDir create operation dir if not exists
func (op *OperationLog) CreateOperationDir(opID string) error {
	if opID == "" {
		return errors.New("opID is invalid")
	}
	return fileutil.CreateDirIfNotExists(filepath.Join(op.cfg.Dir, opID), 0755)
}

// GetOperationDir get operation dir
func (op *OperationLog) GetOperationDir(opID string) (path string, err error) {
	if opID == "" {
		err = errors.New("opID is invalid")
		return
	}
	path = filepath.Join(op.cfg.Dir, opID)
	return
}

// CreateStepLogFile create a file based on opID and stepID, open the file to return the file descriptor. You should close it.
func (op *OperationLog) CreateStepLogFile(opID, stepID string) (*os.File, error) {
	if opID == "" || stepID == "" {
		return nil, errors.New("opID or stepID is invalid")
	}
	return os.OpenFile(filepath.Join(op.cfg.Dir, opID, stepID+OperationLogSuffix), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
}

// GetStepLogFile get step log file path
func (op *OperationLog) GetStepLogFile(opID, stepID string) (path string, err error) {
	if opID == "" || stepID == "" {
		err = errors.New("opId or stepId is invalid")
		return
	}
	path = filepath.Join(op.cfg.Dir, opID, stepID+OperationLogSuffix)
	return
}

// GetStepLogContent get step log content
func (op *OperationLog) GetStepLogContent(opID, stepID string, offset int64, length int) (content []byte, deliverySize int64, logSize int64, err error) {
	if opID == "" || stepID == "" {
		err = errors.New("opId or stepId is invalid")
		return
	}
	realLen := op.cfg.SingleThreshold
	// length greater than zero and the threshold is not exceeded, length allowed
	if length > 0 && int64(length) <= realLen {
		realLen = int64(length)
	}
	filename := filepath.Join(op.cfg.Dir, opID, stepID+OperationLogSuffix)
	// get file stat
	stat, err := os.Stat(filename)
	if err != nil {
		return
	}
	logSize = stat.Size()
	// calculate the file remain size. if remain size jle 0, then set real length to 0.
	remainSize := logSize - offset
	if remainSize > 0 {
		if remainSize < realLen {
			realLen = remainSize
		}
	} else {
		realLen = 0
	}

	content, err = fileutil.Peek(filename, offset, int(realLen))
	if err != nil {
		return
	}
	deliverySize = int64(len(content))
	return
}

// CreateStepLogFileAndAppend create a file based on opID and stepID, and append data to step log file.
func (op *OperationLog) CreateStepLogFileAndAppend(opID, stepID string, data []byte) error {
	if err := op.CreateOperationDir(opID); err != nil {
		return err
	}
	f, err := op.CreateStepLogFile(opID, stepID)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func (op *OperationLog) TruncateStepLogFile(opID, stepID string) error {
	path, err := op.GetStepLogFile(opID, stepID)
	if err != nil {
		return err
	}
	return os.Truncate(path, 0)
}
