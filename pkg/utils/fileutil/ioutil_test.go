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
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteFileWithDataFunc(t *testing.T) {
	content := "hello\nworld\n"
	tests := []struct {
		filename string
		flag     int
		mode     os.FileMode
		dataFunc func(w io.Writer) error
	}{
		{
			filename: "./test.txt",
			flag:     os.O_WRONLY | os.O_CREATE | os.O_TRUNC,
			mode:     0644,
			dataFunc: func(w io.Writer) error {
				_, err := w.Write([]byte(content))
				return err
			},
		},
	}
	for _, tt := range tests {
		err := WriteFileWithDataFunc(tt.filename, tt.flag, tt.mode, tt.dataFunc, false)
		assert.NoError(t, err)
		data, err := os.ReadFile(tt.filename)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		os.Remove(tt.filename)
	}
}
