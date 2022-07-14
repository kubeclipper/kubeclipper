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

package strutil

import (
	"testing"
)

func TestBase64Encode(t *testing.T) {
	src := "aRMIwp3NJC3G324a"
	expected := "YVJNSXdwM05KQzNHMzI0YQ=="
	if encoded := Base64Encode(src); encoded != expected {
		t.Errorf("base64 encode result: %s, expected: %s", encoded, expected)
	}
}

func TestTrimDuplicates(t *testing.T) {
	src := []string{"foo", "bar", "bar", "baz", "baz", "baz"}
	expected := []string{"foo", "bar", "baz"}
	dst := TrimDuplicates(src)
	if len(dst) != len(expected) {
		t.Error("trim duplicated string failed")
		return
	}
	for i, str := range expected {
		if dst[i] != str {
			t.Error("trim duplicated string failed")
			return
		}
	}
}
