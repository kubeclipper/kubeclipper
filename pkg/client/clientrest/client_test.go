/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package clientrest

import (
	"net/http"
	"testing"
)

func TestIsInformerRawQuery(t *testing.T) {
	request1, _ := http.NewRequest("GET", "http://localhost", nil)
	request2, _ := http.NewRequest("GET", "http://localhost", nil)
	request1.Header.Set(QueryTypeHeader, InformerQuery)
	request2.Header.Set("Connection", "keep-alive")
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "is informer raw query",
			args: args{
				request1,
			},
			want: true,
		},
		{
			name: "not informer raw query",
			args: args{
				request2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInformerRawQuery(tt.args.req); got != tt.want {
				t.Errorf("IsInformerRawQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
