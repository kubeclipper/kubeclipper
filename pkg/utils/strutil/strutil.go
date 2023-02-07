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
	"strconv"
	"strings"
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

// ParseGitDescribeInfo parse `git describe` command return information
// Determine if there are currently any new commits
// new commit info example: v1.1.0-11+b25c67df4a2e87, it must be a branch.
// no new commit info example: v1.1.0, it could be a branch or a tag.
func ParseGitDescribeInfo(v string) (string, bool) {
	var nc bool
	i := strings.Split(v, "-")
	if len(i) > 1 {
		nc = true
	}
	return i[0], nc
}

// StealKubernetesMajorVersionNumber get kubernetes major version number
// e.g.: v1.23.6 convert to 123, v125 convert to 125
func StealKubernetesMajorVersionNumber(version string) (int, error) {
	version = strings.ReplaceAll(version, "v", "")
	version = strings.ReplaceAll(version, ".", "")

	version = strings.Join(strings.Split(version, "")[0:3], "")
	return strconv.Atoi(version)
}
