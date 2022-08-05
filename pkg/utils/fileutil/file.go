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
	"bufio"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"os"
	"time"
)

const (
	// BufferSize defines the buffer size when reading and writing file.
	bufferSize = 8 * 1024 * 1024
)

// IsRegularFile reports whether the file is a regular file.
// If the given file is a symbol link, it will follow the link.
func IsRegularFile(name string) bool {
	f, e := os.Stat(name)
	if e != nil {
		return false
	}
	return f.Mode().IsRegular()
}

// Md5Sum generates md5 for a given file.
func Md5Sum(name string) string {
	if !IsRegularFile(name) {
		return ""
	}
	f, err := os.Open(name)
	if err != nil {
		return ""
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, bufferSize)
	h := md5.New()

	_, err = io.Copy(h, r)
	if err != nil {
		return ""
	}

	return GetMd5Sum(h, nil)
}

// GetMd5Sum gets md5 sum as a string and appends the current hash to b.
func GetMd5Sum(md5 hash.Hash, b []byte) string {
	return fmt.Sprintf("%x", md5.Sum(b))
}

// MoveFile moves the file src to dst.
func MoveFile(src string, dst string) error {
	if !IsRegularFile(src) {
		return fmt.Errorf("failed to move %s to %s: src is not a regular file", src, dst)
	}
	if PathExist(dst) && !IsDir(dst) {
		if err := DeleteFile(dst); err != nil {
			return fmt.Errorf("failed to move %s to %s when deleting dst file: %v", src, dst, err)
		}
	}
	return os.Rename(src, dst)
}

// DeleteFile deletes a file not a directory.
func DeleteFile(filePath string) error {
	if !PathExist(filePath) {
		return fmt.Errorf("failed to delete file %s: file not exist", filePath)
	}
	if IsDir(filePath) {
		return fmt.Errorf("failed to delete file %s: file path is a directory rather than a file", filePath)
	}
	return os.Remove(filePath)
}

// DeleteFiles deletes all the given files.
func DeleteFiles(filePaths ...string) {
	if len(filePaths) > 0 {
		for _, f := range filePaths {
			_ = DeleteFile(f)
		}
	}
}

// PathExist reports whether the path is exist.
// Any error get from os.Stat, it will return false.
func PathExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// IsDir reports whether the path is a directory.
func IsDir(name string) bool {
	f, e := os.Stat(name)
	if e != nil {
		return false
	}
	return f.IsDir()
}

func CreateDirIfNotExists(path string, perm os.FileMode) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModeDir|perm)
	}
	return nil
}

func WriteTxtToFile(filePath, txtToWrite string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(txtToWrite)
	if err != nil {
		return err
	}
	return nil
}

func Backup(filePath, dir string) (bakFile string, err error) {
	if err = CreateDirIfNotExists(dir, 0644); err != nil {
		return
	}
	src, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	// ${timestamp}.bak as suffix
	bakFile = filePath + fmt.Sprintf(".%s.bak", time.Now().Format("20060102150405"))
	dst, err := os.Create(bakFile)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return
}
