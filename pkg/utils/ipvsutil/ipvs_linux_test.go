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

package ipvsutil

import (
	"testing"
)

func TestClear(t *testing.T) {
	t.Skip("can't run in github action")

	type args struct {
		dryRun bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "base",
			args:    args{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Clear(tt.args.dryRun); err != nil {
				t.Errorf("Clear() error = %v", err)
			}
		})
	}
}

func TestCreateIPVS(t *testing.T) {
	t.Skip("can't run in github action")

	type args struct {
		vs     *VirtualServer
		dryRun bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "base",
			args: args{
				vs: &VirtualServer{
					Address: "169.254.169.100",
					Port:    6443,
					RealServers: []RealServer{
						{Address: "172.18.94.111", Port: 6433},
						{Address: "172.18.94.200", Port: 6433},
						{Address: "172.18.94.114", Port: 6433},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateIPVS(tt.args.vs, tt.args.dryRun); err != nil {
				t.Errorf("CreateIPVS() error = %v", err)
			}
		})
	}
}

func TestDeleteIPVS(t *testing.T) {
	t.Skip("can't run in github action")

	type args struct {
		vs     *VirtualServer
		dryRun bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "base",
			args: args{
				vs: &VirtualServer{
					Address: "169.254.169.100",
					Port:    6443,
					RealServers: []RealServer{
						{Address: "172.18.94.111", Port: 6433},
						{Address: "172.18.94.200", Port: 6433},
						{Address: "172.18.94.114", Port: 6433},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DeleteIPVS(tt.args.vs, tt.args.dryRun); err != nil {
				t.Errorf("DeleteIPVS() error = %v", err)
			}
		})
	}
}
