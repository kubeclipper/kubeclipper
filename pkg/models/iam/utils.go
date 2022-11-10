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

package iam

import iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

type userList []iamv1.User

func (l userList) Len() int      { return len(l) }
func (l userList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l userList) Less(i, j int) bool {
	t1 := l[i].CreationTimestamp.Time
	t2 := l[j].CreationTimestamp.Time
	if t1.After(t2) {
		return true
	} else if t1.Before(t2) {
		return false
	} else {
		return l[i].Name > l[j].Name
	}
}

// DesensitizationUserPassword mark user's passwd empty.
func DesensitizationUserPassword(user *iamv1.User) {
	user.Spec.EncryptedPassword = ""
}
