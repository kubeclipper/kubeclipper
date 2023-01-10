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
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"

	"github.com/golang/mock/gomock"

	mock_cluster "github.com/kubeclipper/kubeclipper/pkg/models/cluster/mock"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	nfsprovisioner "github.com/kubeclipper/kubeclipper/pkg/component/nfs"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var (
	c1 = &v1.Cluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo",
		},

		Masters: v1.WorkerNodeList{
			{
				ID:     "1e3ea00f-1403-46e5-a486-70e4cb29d541",
				Labels: map[string]string{common.LabelNodeRole: "master"},
			},
			{
				ID:     "43ed594a-a76f-4370-a14d-551e7b6153de",
				Labels: map[string]string{common.LabelNodeRole: "master"},
			},
			{
				ID:     "c7a91d86-cd53-4c3f-85b0-fbc657778067",
				Labels: map[string]string{common.LabelNodeRole: "master"},
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
		KubernetesVersion: "v1.18.6",
		// ControlPlaneEndpoint: "172.18.94.114:6443",
		CertSANs:      nil,
		LocalRegistry: "172.18.94.144:5000",
		ContainerRuntime: v1.ContainerRuntime{
			Type:             v1.CRIDocker,
			Version:          "19.03.12",
			DataRootDir:      "/var/lib/docker",
			InsecureRegistry: []string{"172.18.94.144:5000"},
		},
		Networking: v1.Networking{
			IPFamily:      v1.IPFamilyIPv4,
			Services:      v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterServiceSubnet}},
			Pods:          v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterPodSubnet}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},

		KubeProxy: v1.KubeProxy{},
		Etcd: v1.Etcd{
			DataDir: "/var/lib/etcd",
		},
		CNI: v1.CNI{
			LocalRegistry: "172.18.94.144:5000",
			Type:          "calico",
			Version:       "v3.11.2",
			Calico: &v1.Calico{
				IPv4AutoDetection: "first-found",
				IPv6AutoDetection: "first-found",
				Mode:              "Overlay-Vxlan-All",
				IPManger:          true,
				MTU:               1440,
			},
		},
	}
	extraMeta = &component.ExtraMetadata{
		Masters: []component.Node{
			{
				ID:       "1e3ea00f-1403-46e5-a486-70e4cb29d541",
				IPv4:     "192.168.1.1",
				NodeIPv4: "192.168.2.1",
				Region:   "default",
			},
			{
				ID:       "43ed594a-a76f-4370-a14d-551e7b6153de",
				IPv4:     "192.168.1.2",
				NodeIPv4: "192.168.2.2",
				Region:   "default",
			},
			{
				ID:       "c7a91d86-cd53-4c3f-85b0-fbc657778067",
				IPv4:     "192.168.1.3",
				NodeIPv4: "192.168.2.3",
				Region:   "default",
			},
		},
		Workers: []component.Node{
			{
				ID:       "4cf1ad74-704c-4290-a523-e524e930245d",
				IPv4:     "192.168.1.4",
				NodeIPv4: "192.168.2.4",
				Region:   "default",
			},
			{
				ID:       "ae4ba282-27f9-4a93-8fe9-63f786781d48",
				IPv4:     "192.168.1.5",
				NodeIPv4: "192.168.2.5",
				Region:   "default",
			},
		},
	}
	bp = &v1.Backup{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "backup01",
		},
		ClusterNodes: map[string]string{},
		Status: v1.BackupStatus{
			FileName: "backupTests",
		},
	}
)

func Test_parseOperationFromCluster(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	type args struct {
		c      *v1.Cluster
		meta   *component.ExtraMetadata
		action v1.StepAction
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test parse cluster install steps",
			args: args{
				c:      c1,
				meta:   extraMeta,
				action: v1.ActionInstall,
			},
			wantErr: false,
		},
		{
			name: "test parse cluster uninstall steps",
			args: args{
				c:      c1,
				meta:   extraMeta,
				action: v1.ActionUninstall,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h.parseOperationFromCluster(tt.args.meta, tt.args.c, tt.args.action)
			if (err != nil) != tt.wantErr && !IgnoreError(err) {
				t.Errorf("parseOperationFromCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_parseOperationFromComponent(t *testing.T) {
	type args struct {
		action     v1.StepAction
		meta       *component.ExtraMetadata
		cluster    *v1.Cluster
		components []v1.Addon
	}
	h := newHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	nfs := nfsprovisioner.NFSProvisioner{
		ManifestsDir:     "/tmp/.nfs",
		Namespace:        "kube-system",
		Replicas:         1,
		ServerAddr:       "172.20.151.105",
		SharedPath:       "/tmp/nfs/data",
		StorageClassName: "nfs-sc",
		IsDefault:        false,
		ReclaimPolicy:    "Delete",
		ArchiveOnDelete:  false,
		MountOptions:     nil,
	}
	nfsByte, _ := json.Marshal(nfs)
	com := []v1.Addon{
		{
			Name:    "nfs-provisioner",
			Version: "v1",
			Config: runtime.RawExtension{
				Raw:    nfsByte,
				Object: nil,
			},
		},
	}
	tests := []struct {
		name string
		arg  args
	}{
		{
			name: "test parse component install step",
			arg: args{
				action:     v1.ActionInstall,
				meta:       extraMeta,
				cluster:    c1,
				components: com,
			},
		},
		{
			name: "test parse component uninstall step",
			arg: args{
				action:     v1.ActionUninstall,
				meta:       extraMeta,
				cluster:    c1,
				components: com,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := h.parseOperationFromComponent(context.Background(), test.arg.meta, test.arg.components, test.arg.cluster, test.arg.action)
			if err != nil && !IgnoreError(err) {
				t.Errorf("  parseOperationFromComponent() error: %v", err)
			}
		})
	}
}

func setUpClusterMock(clusterMockOperator *mock_cluster.MockOperator) {
	clusterMockOperator.EXPECT().ListNodes(gomock.Any(), gomock.Any()).Return(
		&v1.NodeList{
			TypeMeta: metav1.TypeMeta{},
			ListMeta: metav1.ListMeta{},
			Items: []v1.Node{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Node",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "1e3ea00f-1403-46e5-a486-70e4cb29d541",
						Labels: map[string]string{common.LabelNodeRole: "master"},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Node",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "43ed594a-a76f-4370-a14d-551e7b6153de",
						Labels: map[string]string{common.LabelNodeRole: "master"},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Node",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:   "c7a91d86-cd53-4c3f-85b0-fbc657778067",
						Labels: map[string]string{common.LabelNodeRole: "master"},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Node",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "4cf1ad74-704c-4290-a523-e524e930245d",
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Node",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
					},
				},
			},
		},
		nil).AnyTimes()
	clusterMockOperator.EXPECT().GetBackupPoint(gomock.Any(), gomock.Any(), gomock.Eq("0")).Return(
		&v1.BackupPoint{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BackupPoint",
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "backup01",
			},
			StorageType: "FS",
			FsConfig: &v1.FsConfig{
				BackupRootDir: "/tmp/backup",
			},
		},
		nil).AnyTimes()
}

func Test_parseRecoverySteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clusterMockOperator := mock_cluster.NewMockOperator(ctrl)
	setUpClusterMock(clusterMockOperator)

	h := &handler{
		clusterOperator: clusterMockOperator,
	}
	type args struct {
		cluster *v1.Cluster
		backup  *v1.Backup
		restore string
		action  v1.StepAction
	}
	tests := []struct {
		name string
		arg  args
	}{
		{
			name: "test recovery backup step",
			arg: args{
				cluster: c1,
				backup:  bp,
				restore: "/tmp/backup",
				action:  v1.ActionInstall,
			},
		},
		{
			name: "test delete backup step",
			arg: args{
				cluster: c1,
				backup:  bp,
				restore: "/tmp/backup",
				action:  v1.ActionUninstall,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := h.parseRecoverySteps(test.arg.cluster, test.arg.backup, "/tmp/backup", test.arg.action)
			if err != nil && !IgnoreError(err) {
				t.Errorf(" parseRecoverySteps() error: %v", err)
			}
		})
	}
}

func Test_parseActBackupSteps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clusterMockOperator := mock_cluster.NewMockOperator(ctrl)
	clusterMockOperator.EXPECT().GetBackupPointEx(gomock.Any(), gomock.Any(), gomock.Eq("0")).Return(
		&v1.BackupPoint{
			TypeMeta: metav1.TypeMeta{
				Kind:       "BackupPoint",
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "backup01",
			},
			StorageType: "fs",
			FsConfig: &v1.FsConfig{
				BackupRootDir: "/tmp/backup",
			},
		},
		nil).AnyTimes()
	clusterMockOperator.EXPECT().GetNodeEx(gomock.Any(), gomock.Any(), gomock.Eq("0")).Return(
		&v1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "947627ea-160f-48e2-9a90-07a9739425aa",
			},
			ProxyIpv4CIDR: "10.0.0.0/32",
			Status: v1.NodeStatus{
				Ipv4DefaultIP:     "127.0.0.1",
				NodeIpv4DefaultIP: "127.0.0.1",
				NodeInfo: v1.NodeSystemInfo{
					Hostname: "test",
				},
			},
		},
		nil).AnyTimes()
	h := &handler{
		clusterOperator: clusterMockOperator,
	}

	type args struct {
		cluster *v1.Cluster
		backup  *v1.Backup
		action  v1.StepAction
	}
	tests := []struct {
		name string
		arg  args
	}{
		{
			name: "test create backup step",
			arg: args{
				cluster: c1,
				backup:  bp,
				action:  v1.ActionInstall,
			},
		},
		{
			name: "test delete backup step",
			arg: args{
				cluster: c1,
				backup:  bp,
				action:  v1.ActionUninstall,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := h.parseActBackupSteps(test.arg.cluster, test.arg.backup, test.arg.action)
			if err != nil && !IgnoreError(err) {
				t.Errorf(" parseActBackupSteps() error: %v", err)
			}
		})
	}
}

const (
	availableMasterError    = "no master node available"
	allAvailableMasterError = "there is an unavailable master node in the cluster"
)

func IgnoreError(err error) bool {
	return strings.Contains(err.Error(), availableMasterError) || strings.Contains(err.Error(), allAvailableMasterError)
}
