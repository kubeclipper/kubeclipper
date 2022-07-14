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

package token

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/apiserver/pkg/authentication/user"
)

func TestTokenIssueToAndVerify(t *testing.T) {
	s := &jwtTokenIssuer{
		name:             "kubeclipper",
		secret:           []byte("D9ykGOmuE3yXe35Wh3mRniGT"),
		maximumClockSkew: 0,
	}
	u := &user.DefaultInfo{
		Name: "admin",
	}
	token, err := s.IssueTo(u, "bearer", 10*time.Hour)
	if err != nil {
		t.Errorf("IssueTo() error = %v", err)
		return
	}
	got, _, err2 := s.Verify(token)
	if err2 != nil {
		t.Errorf("Verify() error = %v", err2)
		return
	}
	if !reflect.DeepEqual(u, got) {
		t.Errorf("Verify() got = %v, want %v", got, u)
	}
}
