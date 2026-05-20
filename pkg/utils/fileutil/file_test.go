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

func TestCopyFile(t *testing.T) {
	expected := `test content`
	srcFile := "test_src.txt"
	dstFile := "test_dst.txt"

	if err := WriteFileWithDataFunc(srcFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		func(w io.Writer) error {
			_, err := w.Write([]byte(expected))
			return err
		}, false); err != nil {
		assert.FailNowf(t, "failed to write file", err.Error())
	}

	if err := CopyFile(srcFile, dstFile, 0644); err != nil {
		assert.FailNowf(t, "failed to copy file", err.Error())
	}

	actual, err := os.ReadFile(dstFile)
	if err != nil {
		assert.FailNowf(t, "failed to read copied file", err.Error())
	}
	assert.Equal(t, expected, string(actual))
	os.Remove(srcFile)
	os.Remove(dstFile)
}
