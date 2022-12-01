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

package v1

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"

	"k8s.io/apimachinery/pkg/runtime"

	nfsprovisioner "github.com/kubeclipper/kubeclipper/pkg/component/nfs"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var (
	c2 = &v1.Cluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo",
		},

		Masters: v1.WorkerNodeList{
			{
				ID: "1e3ea00f-1403-46e5-a486-70e4cb29d541",
			},
			{
				ID: "43ed594a-a76f-4370-a14d-551e7b6153de",
			},
			{
				ID: "c7a91d86-cd53-4c3f-85b0-fbc657778067",
			},
		},
		Workers: v1.WorkerNodeList{
			{
				ID: "4cf1ad74-704c-4290-a523-e524e930245d",
			},
			{
				ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
			},
		},
		ContainerRuntime: v1.ContainerRuntime{
			Type:        "docker",
			Version:     "19.03.12",
			DataRootDir: "/var/lib/docker",
		},
		Addons:    addons,
		KubeProxy: v1.KubeProxy{},
		Etcd:      v1.Etcd{},
		CNI: v1.CNI{
			LocalRegistry: "172.20.150.138:5000",
			Type:          "calico",
			Version:       "v3.21.2",
			Calico: &v1.Calico{
				IPv4AutoDetection: "first-found",
				IPv6AutoDetection: "first-found",
				Mode:              "Overlay-Vxlan-All",
				IPManger:          true,
				MTU:               1440,
			},
		},
		Networking: v1.Networking{
			IPFamily:      v1.IPFamilyIPv4,
			Services:      v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterServiceSubnet}},
			Pods:          v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterPodSubnet}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},
	}
	addons = []v1.Addon{
		{
			Name:    "nfs-provisioner",
			Version: "v1",
			Config:  runtime.RawExtension{Raw: []byte(`{"scName": "nfs-provisioner-v1"}`)},
		},
	}
)

func init() {
	nfs := &nfsprovisioner.NFSProvisioner{}
	_ = component.Register(fmt.Sprintf(component.RegisterFormat, "nfs-provisioner", "v1"), nfs)

}

func Test_checkComponents(t *testing.T) {
	type args struct {
		cluster *v1.Cluster
		pc      *PatchComponents
	}
	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "test component whether exist when uninstall",
			arg: args{
				cluster: c2,
				pc: &PatchComponents{
					Uninstall: true,
					Addons: []v1.Addon{
						{
							Name:    "nfs-provisioner",
							Version: "v1",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test component whether exist when install",
			arg: args{
				cluster: c2,
				pc: &PatchComponents{
					Uninstall: false,
					Addons: []v1.Addon{
						{
							Name:    "nfs-provisioner",
							Version: "v1",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "test component whether exist when install",
			arg: args{
				cluster: c2,
				pc: &PatchComponents{
					Uninstall: false,
					Addons: []v1.Addon{
						{
							Name:    "testComponent",
							Version: "v1",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.arg.pc.checkComponents(test.arg.cluster); (err != nil) != test.wantErr {
				t.Errorf(" checkComponents() error: %v", err)
			}
		})
	}
}

func Test_addOrRemoveComponentFromCluster(t *testing.T) {
	type args struct {
		cluster *v1.Cluster
		pc      *PatchComponents
	}
	tests := []struct {
		name string
		arg  args
		want []v1.Addon
	}{
		{
			name: "test uninstall component",
			arg: args{
				cluster: c2,
				pc: &PatchComponents{
					Uninstall: true,
					Addons: []v1.Addon{
						{
							Name:    "nfs-provisioner",
							Version: "v1",
							Config: runtime.RawExtension{
								Raw: []byte(`{"scName": "nfs-provisioner-v1"}`),
							},
						},
					},
				},
			},
			want: []v1.Addon{},
		},
		{
			name: "test install component",
			arg: args{
				cluster: c2,
				pc: &PatchComponents{
					Uninstall: false,
					Addons: []v1.Addon{
						{
							Name:    "testComponent",
							Version: "v1",
							Config: runtime.RawExtension{
								Raw: []byte(`{"scName": "testComponent-v1"}`),
							},
						},
					},
				},
			},
			want: []v1.Addon{
				{
					Name:    "testComponent",
					Version: "v1",
					Config:  runtime.RawExtension{Raw: []byte(`{"scName": "testComponent-v1"}`)},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.arg.pc.addOrRemoveComponentFromCluster(test.arg.cluster)
			if err != nil {
				t.Errorf("addOrRemoveComponentFromCluster() error: %v", err)
			}
			if !reflect.DeepEqual(got.Addons, test.want) {
				t.Errorf(" addOrRemoveComponentFromCluster() error: got %v, want %v", got.Addons, test.want)
			}
		})
	}
}
