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
	"testing"
)

func TestDiskInfo(t *testing.T) {
	type args struct {
		byteSize ByteSize
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				byteSize: 2048,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DiskInfo(tt.args.byteSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiskInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGetSysInfo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "get system info test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetSysInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSysInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestHostInfo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "get host info",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HostInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("HostInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestMemoryInfo(t *testing.T) {
	type args struct {
		byteSize ByteSize
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get memory info test",
			args: args{
				byteSize: 2048,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MemoryInfo(tt.args.byteSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
