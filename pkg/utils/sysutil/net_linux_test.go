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

package sysutil

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_netInfo(t *testing.T) {
	tests := []struct {
		name    string
		want    []Net
		wantErr bool
	}{
		{
			name:    "get net info test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := netInfo()
			fmt.Printf("got: %#v", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("netInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("netInfo() got = %v, want %v", got, tt.want)
			// }
		})
	}
}

func TestCUPInfo1(t *testing.T) {
	tests := []struct {
		name    string
		want    CPU
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CUPInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("CUPInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CUPInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_netInfo1(t *testing.T) {
	tests := []struct {
		name    string
		want    []Net
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := netInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("netInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("netInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}
