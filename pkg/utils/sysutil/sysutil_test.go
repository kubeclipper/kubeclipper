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
			name: "valid byte size",
			args: args{
				byteSize: 2048,
			},
			wantErr: false,
		},
		{
			name: "minimum byte size",
			args: args{
				byteSize: KB,
			},
			wantErr: false,
		},
		{
			name: "maximum byte size",
			args: args{
				byteSize: TB,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DiskInfo(tt.args.byteSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("DiskInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Verify that we have disk devices
				if len(result.DiskDevices) == 0 {
					t.Skip("No disk devices found, skipping validation tests")
				}
				
				// Verify accumulation logic: total should be sum of all partitions
				var expectedTotal, expectedUsed, expectedFree uint64
				for _, device := range result.DiskDevices {
					expectedTotal += device.Total / uint64(tt.args.byteSize)
					expectedUsed += device.Used / uint64(tt.args.byteSize)
					expectedFree += device.Free / uint64(tt.args.byteSize)
				}
				
				if result.Total != expectedTotal {
					t.Errorf("DiskInfo() Total = %v, expected %v (sum of all partitions)", result.Total, expectedTotal)
				}
				
				if result.Used != expectedUsed {
					t.Errorf("DiskInfo() Used = %v, expected %v (sum of all partitions)", result.Used, expectedUsed)
				}
				
				if result.Free != expectedFree {
					t.Errorf("DiskInfo() Free = %v, expected %v (sum of all partitions)", result.Free, expectedFree)
				}
				
				// Verify used percentage calculation
				if result.Total > 0 {
					expectedUsedPercent := float64(result.Used) / float64(result.Total) * 100
					if result.UsedPercent < expectedUsedPercent-0.01 || result.UsedPercent > expectedUsedPercent+0.01 {
						t.Errorf("DiskInfo() UsedPercent = %.2f, expected approximately %.2f", result.UsedPercent, expectedUsedPercent)
					}
				}
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
