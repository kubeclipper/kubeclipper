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

package nodestatus

import (
	"reflect"
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sysutil"
)

func Test_attachedVolumes(t *testing.T) {
	type args struct {
		d sysutil.Disk
	}
	tests := []struct {
		name string
		args args
		want []v1.AttachedVolume
	}{
		{
			name: "base",
			args: args{
				d: sysutil.Disk{
					DiskDevices: []sysutil.DiskDevice{
						{
							Device:     "device1",
							Mountpoint: "/root/point1",
						},
						{
							Device:     "device2",
							Mountpoint: "/root/point2",
						},
						{
							Device:     "device3",
							Mountpoint: "/root/point3",
						},
					},
				},
			},
			want: []v1.AttachedVolume{
				{
					Name:       "device1",
					DevicePath: "/root/point1",
				},
				{
					Name:       "device2",
					DevicePath: "/root/point2",
				},
				{
					Name:       "device3",
					DevicePath: "/root/point3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := attachedVolumes(tt.args.d); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("attachedVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toNodeAddress(t *testing.T) {
	type args struct {
		n []sysutil.Net
	}
	tests := []struct {
		name string
		args args
		want []v1.NodeAddress
	}{
		{
			name: "base",
			args: args{
				n: []sysutil.Net{
					{
						Addrs: []sysutil.InterfaceAddr{
							{
								Family: "family1",
								Addr:   "10.0.0.1",
							},
							{
								Family: "family1",
								Addr:   "10.0.0.2",
							},
						},
					},
				},
			},
			want: []v1.NodeAddress{
				{
					Type:    "family1",
					Address: "10.0.0.1",
				},
				{
					Type:    "family1",
					Address: "10.0.0.2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toNodeAddress(tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toNodeAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
