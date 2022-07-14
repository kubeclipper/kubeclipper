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
	"encoding/base64"
)

func Base64Encode(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func StringDefaultIfEmpty(dft, src string) string {
	if src == "" {
		return dft
	}
	return src
}

func TrimDuplicates(src []string) []string {
	if src == nil {
		return nil
	}
	m := map[string]struct{}{}
	i := 0
	for _, str := range src {
		if _, ok := m[str]; !ok {
			m[str] = struct{}{}
			src[i] = str
			i++
		}
	}
	return src[:i]
}
