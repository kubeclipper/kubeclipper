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

package backupstore

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func init() {
	RegisterProvider(&FilesystemStore{})
}

var _ BackupStore = (*FilesystemStore)(nil)

const (
	defaultRootDir = "/opt/kc/backups"
)

type FilesystemStore struct {
	RootDir string `json:"rootDir,omitempty" yaml:"rootDir,omitempty"` // root directory for storing backup files
}

func (fs *FilesystemStore) Type() string {
	return FSStorage
}

func (fs *FilesystemStore) Create() (BackupStore, error) {
	if fs.RootDir == "" {
		fs.RootDir = defaultRootDir
	}

	return fs, nil
}

func (fs *FilesystemStore) Save(ctx context.Context, r io.Reader, fileName string) (err error) {
	defer logProbe(ctx, fmt.Sprintf("save backup to %s", filepath.Join(fs.RootDir, fileName)), err)
	w, err := os.Create(filepath.Join(fs.RootDir, fileName))
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return w.Sync()
}

func (fs *FilesystemStore) Delete(ctx context.Context, fileName string) (err error) {
	defer logProbe(ctx, fmt.Sprintf("delete backup from %s", filepath.Join(fs.RootDir, fileName)), err)
	err = os.Remove(filepath.Join(fs.RootDir, fileName))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// The target file is already deleted.
		return nil
	}
	return
}

func (fs *FilesystemStore) Download(ctx context.Context, fileName string, w io.Writer) (err error) {
	defer logProbe(ctx, fmt.Sprintf("download backup from %s", filepath.Join(fs.RootDir, fileName)), err)
	f, err := os.OpenFile(filepath.Join(fs.RootDir, fileName), os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}

	write := bufio.NewWriter(w)

	_, err = bufio.NewReader(f).WriteTo(write)
	if err != nil {
		return err
	}
	return write.Flush()
}
