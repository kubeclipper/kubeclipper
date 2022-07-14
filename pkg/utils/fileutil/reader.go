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
	"crypto/md5"
	"hash"
	"io"
)

func NewFileReader(src io.Reader, calculateMd5 bool) *FileReader {
	var md5sum hash.Hash
	if calculateMd5 {
		md5sum = md5.New()
	}

	return &FileReader{
		Src:    src,
		md5sum: md5sum,
	}
}

type FileReader struct {
	Src    io.Reader
	md5sum hash.Hash
}

func (r *FileReader) Read(p []byte) (n int, err error) {
	n, e := r.Src.Read(p)
	if e != nil && e != io.EOF {
		return n, e
	}
	if n > 0 {
		if r.md5sum != nil {
			_, _ = r.md5sum.Write(p[:n])
		}
	}
	return n, e
}

func (r *FileReader) Md5() string {
	if r.md5sum != nil {
		return GetMd5Sum(r.md5sum, nil)
	}
	return ""
}
