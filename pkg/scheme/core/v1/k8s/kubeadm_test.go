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

package k8s

import (
	"bytes"
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func Test_kubectlTerminal_renderTo(t *testing.T) {
	c := &KubectlTerminal{
		// ImageRegistryAddr: "192.168.234.130:5000",
		ImageRegistryAddr: "",
	}
	w := &bytes.Buffer{}
	err := c.renderTo(w)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(w.String())
}

func TestKubeadmConfig_renderTo(t *testing.T) {
	type fields struct {
		ClusterConfigAPIVersion string
		ContainerRuntime        string
		Etcd                    v1.Etcd
		Networking              v1.Networking
		KubeProxy               v1.KubeProxy
		Kubelet                 v1.Kubelet
		ClusterName             string
		KubernetesVersion       string
		ControlPlaneEndpoint    string
		CertSANs                []string
		LocalRegistry           string
	}
	tests := []struct {
		name   string
		fields fields
		wantW  string
	}{
		{
			name: "base",
			fields: fields{
				ClusterConfigAPIVersion: "v1beta3",
				ContainerRuntime:        "containerd",
				Etcd:                    v1.Etcd{DataDir: "/var/lib/etcd"},
				Networking: v1.Networking{
					IPFamily:      v1.IPFamilyIPv4,
					Services:      v1.NetworkRanges{CIDRBlocks: []string{"10.96.0.0/16"}},
					Pods:          v1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/24"}},
					DNSDomain:     "cluster.local",
					ProxyMode:     "ipvs",
					WorkerNodeVip: "8.8.8.8",
				},
				KubeProxy:            v1.KubeProxy{},
				Kubelet:              v1.Kubelet{RootDir: "/var/lib/kubelet"},
				ClusterName:          "test-cluster",
				KubernetesVersion:    "v1.23.6",
				ControlPlaneEndpoint: "apiserver.cluster.local:6443",
				CertSANs:             []string{"127.0.0.1"},
				LocalRegistry:        "127.0.0.1:5000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepper := &KubeadmConfig{
				ClusterConfigAPIVersion: tt.fields.ClusterConfigAPIVersion,
				ContainerRuntime:        tt.fields.ContainerRuntime,
				Etcd:                    tt.fields.Etcd,
				Networking:              tt.fields.Networking,
				KubeProxy:               tt.fields.KubeProxy,
				Kubelet:                 tt.fields.Kubelet,
				ClusterName:             tt.fields.ClusterName,
				KubernetesVersion:       tt.fields.KubernetesVersion,
				ControlPlaneEndpoint:    tt.fields.ControlPlaneEndpoint,
				CertSANs:                tt.fields.CertSANs,
				LocalRegistry:           tt.fields.LocalRegistry,
			}
			w := &bytes.Buffer{}
			err := stepper.renderTo(w)
			if err != nil {
				t.Errorf("renderTo() error = %v", err)
				return
			}
			t.Log(w.String())
		})
	}
}

func TestCNI_renderCalicoTo(t *testing.T) {
	tests := []struct {
		name    string
		stepper CNIInfo
		wantW   string
		wantErr bool
	}{
		{
			name: "base",
			stepper: CNIInfo{
				CNI: v1.CNI{
					LocalRegistry: "172.0.0.1:5000",
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
				DualStack:   false,
				PodIPv4CIDR: "172.25.0.0/24",
				PodIPv6CIDR: "aaa:bbb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := tt.stepper.renderCalicoTo(w)
			if err != nil {
				t.Errorf("renderCalicoTo() error = %v", err)
				return
			}
			t.Log(w.String())
		})
	}
}
