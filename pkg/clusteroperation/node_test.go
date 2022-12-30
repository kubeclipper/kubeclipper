package clusteroperation

import (
	"strings"
	"testing"

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
			Services:      v1.NetworkRanges{CIDRBlocks: []string{"10.96.0.0/16"}},
			Pods:          v1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/24"}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},
	}
	master = v1.WorkerNodeList{
		{
			ID: "1e3ea00f-1403-46e5-a486-70e4cb29d541",
		},
		{
			ID: "43ed594a-a76f-4370-a14d-551e7b6153de",
		},
		{
			ID: "c7a91d86-cd53-4c3f-85b0-fbc657778067",
		},
	}
	worker = v1.WorkerNodeList{
		{
			ID: "4cf1ad74-704c-4290-a523-e524e930245d",
		},
		{
			ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
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
)

func Test_MakeCompare(t *testing.T) {
	type args struct {
		cluster   *v1.Cluster
		patchNode *PatchNodes
	}
	tests := []struct {
		name    string
		arg     args
		wantErr error
	}{
		{
			name: "addWorker",
			arg: args{
				cluster: c2,
				patchNode: &PatchNodes{
					Operation: "add",
					Nodes:     worker,
					Role:      "worker",
				},
			},
			wantErr: nil,
		},
		{
			name: "removeWorker",
			arg: args{
				cluster: c2,
				patchNode: &PatchNodes{
					Operation: "remove",
					Nodes:     worker,
					Role:      "worker",
				},
			},
			wantErr: nil,
		},
		{
			name: "addMaster",
			arg: args{
				cluster: c2,
				patchNode: &PatchNodes{
					Operation: "add",
					Nodes:     master,
					Role:      "master",
				},
			},
			wantErr: ErrInvalidNodesRole,
		},
		{
			name: "removeMaster",
			arg: args{
				cluster: nil,
				patchNode: &PatchNodes{
					Operation: "remove",
					Nodes:     master,
					Role:      "master",
				},
			},
			wantErr: ErrInvalidNodesRole,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.arg.patchNode.MakeCompare(test.arg.cluster); err != nil && err != test.wantErr {
				t.Errorf(" MakeCompare() err: %v ", err)
			}
		})
	}
}

func Test_MakeOperation(t *testing.T) {
	type args struct {
		cluster    *v1.Cluster
		meta       component.ExtraMetadata
		patchNodes *PatchNodes
	}
	tests := []struct {
		name    string
		arg     args
		wantErr error
	}{
		{
			name: "test add worker node operation",
			arg: args{
				cluster: c2,
				meta:    *extraMeta,
				patchNodes: &PatchNodes{
					Operation: "add",
					Nodes: []v1.WorkerNode{
						{
							ID: "6b8456e8-2489-4321-bbb0-f8d75c065384",
						},
					},
					Role: "worker",
				},
			},
			wantErr: nil,
		},
		{
			name: "test remove worker node operation",
			arg: args{
				cluster: c2,
				meta:    *extraMeta,
				patchNodes: &PatchNodes{
					Operation: "remove",
					Nodes: []v1.WorkerNode{
						{
							ID: "4cf1ad74-704c-4290-a523-e524e930245d",
						},
					},
					Role: "worker",
				},
			},
			wantErr: nil,
		},
		{
			name: "test add master node operation",
			arg: args{
				cluster: c2,
				meta:    *extraMeta,
				patchNodes: &PatchNodes{
					Operation: "remove",
					Nodes: []v1.WorkerNode{
						{
							ID: "1e3ea00f-1403-46e5-a486-70e4cb29d541",
						},
					},
					Role: "master",
				},
			},
			wantErr: ErrInvalidNodesRole,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.arg.patchNodes.MakeOperation(test.arg.meta, test.arg.cluster)
			if err != nil && err != test.wantErr && !IgnoreError(err) {
				t.Errorf(" MakeOperation() error: %v ", err)
			}
		})
	}

}

const (
	availableMasterError = "no master node available"
)

func IgnoreError(err error) bool {
	return strings.Contains(err.Error(), availableMasterError)
}
