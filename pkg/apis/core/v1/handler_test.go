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

package v1

import "testing"

func Test_validateExternalPort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{name: "valid port", port: "443", wantErr: false},
		{name: "valid max port", port: "65535", wantErr: false},
		{name: "valid min port", port: "1", wantErr: false},
		{name: "zero port", port: "0", wantErr: true},
		{name: "negative port", port: "-1", wantErr: true},
		{name: "port too large", port: "70000", wantErr: true},
		{name: "not a number", port: "abc", wantErr: true},
		{name: "empty string", port: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExternalPort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExternalPort(%q) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}
