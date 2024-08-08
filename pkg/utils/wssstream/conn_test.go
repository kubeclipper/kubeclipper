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

package wssstream

import (
	"net/http"
	"testing"
)

func TestIsWebSocketRequest(t *testing.T) {
	goodRequest, _ := http.NewRequest("GET", "http://example.com", nil)
	goodRequest.Header.Set("Upgrade", "websocket")
	goodRequest.Header.Set("Connection", "upgrade")
	badRequest, _ := http.NewRequest("GET", "http://example.com", nil)
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "websocket request",
			args: args{
				req: goodRequest,
			},
			want: true,
		},
		{
			name: "not a websocket request",
			args: args{
				req: badRequest,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWebSocketRequest(tt.args.req); got != tt.want {
				t.Errorf("IsWebSocketRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
