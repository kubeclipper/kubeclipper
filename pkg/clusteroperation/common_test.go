package clusteroperation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

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
			Type:        "containerd",
			Version:     "1.6.4",
			DataRootDir: "/var/lib/containerd",
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
)

func Test_getCriStep(t *testing.T) {
	runnable1 := &cri.DockerRunnable{
		InsecureRegistry: []string{
			"172.20.150.138:5000",
		},
		Base: cri.Base{
			Version:     "19.03.12",
			Offline:     true,
			DataRootDir: "/var/lib/docker",
		},
	}
	runnable2 := &cri.ContainerdRunnable{
		LocalRegistry: "172.20.150.138:5000",
		Base: cri.Base{
			Version:     "1.6.4",
			Offline:     true,
			DataRootDir: "/var/lib/containerd",
		},
	}
	runtimeBytes1, _ := json.Marshal(runnable1)
	runtimeBytes2, _ := json.Marshal(runnable2)
	type args struct {
		ctx    context.Context
		c      *v1.Cluster
		action v1.StepAction
		nodes  []v1.StepNode
	}
	tests := []struct {
		name    string
		args    args
		want    []v1.Step
		wantErr bool
	}{
		{
			name: "get cri install step with runtime is docker",
			args: args{
				ctx:    context.TODO(),
				c:      c1,
				action: v1.ActionInstall,
				nodes: []v1.StepNode{
					{
						ID: "4cf1ad74-704c-4290-a523-e524e930245d",
					},
					{
						ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
					},
				},
			},
			want: []v1.Step{
				{
					ID:         strutil.GetUUID(),
					Name:       "installRuntime",
					Timeout:    metav1.Duration{Duration: 10 * time.Minute},
					ErrIgnore:  false,
					RetryTimes: 1,
					Nodes: []v1.StepNode{
						{
							ID: "4cf1ad74-704c-4290-a523-e524e930245d",
						},
						{
							ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
						},
					},
					Action: v1.ActionInstall,
					Commands: []v1.Command{
						{
							Type:          v1.CommandCustom,
							Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, "docker", "v1", component.TypeStep),
							CustomCommand: runtimeBytes1,
						},
					},
				},
			},
		},
		{
			name: "get cri uninstall step with runtime is docker",
			args: args{
				ctx:    context.TODO(),
				c:      c2,
				action: v1.ActionUninstall,
				nodes: []v1.StepNode{
					{
						ID: "4cf1ad74-704c-4290-a523-e524e930245d",
					},
					{
						ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
					},
				},
			},
			want: []v1.Step{
				{
					ID:         strutil.GetUUID(),
					Name:       "uninstallRuntime",
					Timeout:    metav1.Duration{Duration: 10 * time.Minute},
					ErrIgnore:  false,
					RetryTimes: 1,
					Nodes: []v1.StepNode{
						{
							ID: "4cf1ad74-704c-4290-a523-e524e930245d",
						},
						{
							ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
						},
					},
					Action: v1.ActionUninstall,
					Commands: []v1.Command{
						{
							Type:          v1.CommandCustom,
							Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, "docker", "v1", component.TypeStep),
							CustomCommand: runtimeBytes1,
						},
					},
				},
			},
		},
		{
			name: "get cri install step with runtime is containerd",
			args: args{
				ctx:    context.TODO(),
				c:      c2,
				action: v1.ActionInstall,
				nodes: []v1.StepNode{
					{
						ID: "4cf1ad74-704c-4290-a523-e524e930245d",
					},
					{
						ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
					},
				},
			},
			want: []v1.Step{
				{
					ID:         strutil.GetUUID(),
					Name:       "installRuntime",
					Timeout:    metav1.Duration{Duration: 10 * time.Minute},
					ErrIgnore:  false,
					RetryTimes: 1,
					Nodes: []v1.StepNode{
						{
							ID: "4cf1ad74-704c-4290-a523-e524e930245d",
						},
						{
							ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
						},
					},
					Action: v1.ActionInstall,
					Commands: []v1.Command{
						{
							Type:          v1.CommandCustom,
							Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, "containerd", "v1", component.TypeStep),
							CustomCommand: runtimeBytes2,
						},
					},
				},
			},
		},
		{
			name: "get cri uninstall step with runtime is containerd",
			args: args{
				ctx:    context.TODO(),
				c:      c2,
				action: v1.ActionUninstall,
				nodes: []v1.StepNode{
					{
						ID: "4cf1ad74-704c-4290-a523-e524e930245d",
					},
					{
						ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
					},
				},
			},
			want: []v1.Step{
				{
					ID:         strutil.GetUUID(),
					Name:       "uninstallRuntime",
					Timeout:    metav1.Duration{Duration: 10 * time.Minute},
					ErrIgnore:  false,
					RetryTimes: 1,
					Nodes: []v1.StepNode{
						{
							ID: "4cf1ad74-704c-4290-a523-e524e930245d",
						},
						{
							ID: "ae4ba282-27f9-4a93-8fe9-63f786781d48",
						},
					},
					Action: v1.ActionUninstall,
					Commands: []v1.Command{
						{
							Type:          v1.CommandCustom,
							Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, "containerd", "v1", component.TypeStep),
							CustomCommand: runtimeBytes2,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getCriStep(tt.args.ctx, tt.args.c, tt.args.action, tt.args.nodes)
			if err != nil {
				t.Errorf(" getCriStep() error: %v", err)
			}
		})
	}
}
