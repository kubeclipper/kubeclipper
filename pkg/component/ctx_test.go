package component

import (
	"reflect"
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestNodeList_GetNodeIDs(t *testing.T) {
	tests := []struct {
		name      string
		l         NodeList
		wantNodes []string
	}{
		{
			name: "base",
			l: NodeList{
				{
					ID: "Node1",
				},
				{
					ID: "Node2",
				},
				{
					ID: "Node3",
				},
			},
			wantNodes: []string{"Node1", "Node2", "Node3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotNodes := tt.l.GetNodeIDs(); !reflect.DeepEqual(gotNodes, tt.wantNodes) {
				t.Errorf("GetNodeIDs() = %v, want %v", gotNodes, tt.wantNodes)
			}
		})
	}
}

func TestExtraMetadata_GetAllNodeIDs(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "base",
			fields: fields{
				Masters: NodeList{
					{
						ID: "Master1",
					},
					{
						ID: "Master2",
					},
					{
						ID: "Master3",
					},
				},
				Workers: NodeList{
					{
						ID: "Node1",
					},
					{
						ID: "Node2",
					},
					{
						ID: "Node3",
					},
				},
			},
			want: []string{"Master1", "Master2", "Master3", "Node1", "Node2", "Node3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetAllNodeIDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllNodeIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetAllNodes(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name      string
		fields    fields
		wantNodes NodeList
	}{
		{
			name: "base",
			fields: fields{
				Masters: NodeList{
					{
						ID: "Master1",
					},
					{
						ID: "Master2",
					},
					{
						ID: "Master3",
					},
				},
				Workers: NodeList{
					{
						ID: "Node1",
					},
					{
						ID: "Node2",
					},
					{
						ID: "Node3",
					},
				},
			},
			wantNodes: NodeList{
				{
					ID: "Master1",
				},
				{
					ID: "Master2",
				},
				{
					ID: "Master3",
				},
				{
					ID: "Node1",
				},
				{
					ID: "Node2",
				},
				{
					ID: "Node3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if gotNodes := e.GetAllNodes(); !reflect.DeepEqual(gotNodes, tt.wantNodes) {
				t.Errorf("GetAllNodes() = %v, want %v", gotNodes, tt.wantNodes)
			}
		})
	}
}

func TestExtraMetadata_GetMasterHostname(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "base",
			fields: fields{
				Masters: NodeList{
					{
						ID:       "Master1",
						Hostname: "Host1",
					},
					{
						ID:       "Master2",
						Hostname: "Host2",
					},
				},
			},
			args: args{
				id: "Master1",
			},
			want: "Host1",
		},
		{
			name: "Master ID is not exist",
			fields: fields{
				Masters: NodeList{
					{
						ID:       "Master1",
						Hostname: "Host1",
					},
					{
						ID:       "Master2",
						Hostname: "Host2",
					},
				},
			},
			args: args{
				id: "Master3",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetMasterHostname(tt.args.id); got != tt.want {
				t.Errorf("GetMasterHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetWorkerHostname(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "base",
			fields: fields{
				Workers: NodeList{
					{
						ID:       "Node1",
						Hostname: "Host1",
					},
					{
						ID:       "Node2",
						Hostname: "Host2",
					},
				},
			},
			args: args{
				id: "Node1",
			},
			want: "Host1",
		},
		{
			name: "Worker ID is not exist",
			fields: fields{
				Workers: NodeList{
					{
						ID:       "Node1",
						Hostname: "Host1",
					},
					{
						ID:       "Node2",
						Hostname: "Host2",
					},
				},
			},
			args: args{
				id: "Node3",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetWorkerHostname(tt.args.id); got != tt.want {
				t.Errorf("GetWorkerHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetMasterNodeIP(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "base",
			fields: fields{
				Masters: NodeList{
					{
						ID:   "Master1",
						IPv4: "10.0.0.1",
					},
					{
						ID:   "Master2",
						IPv4: "10.0.0.2",
					},
				},
			},
			want: map[string]string{
				"Master1": "10.0.0.1",
				"Master2": "10.0.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetMasterNodeIP(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMasterNodeIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetWorkerNodeIP(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "base",
			fields: fields{
				Workers: NodeList{
					{
						ID:   "Node1",
						IPv4: "10.0.0.1",
					},
					{
						ID:   "Node2",
						IPv4: "10.0.0.2",
					},
				},
			},
			want: map[string]string{
				"Node1": "10.0.0.1",
				"Node2": "10.0.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetWorkerNodeIP(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkerNodeIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetMasterNodeClusterIP(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "base",
			fields: fields{
				Masters: NodeList{
					{
						ID:       "Master1",
						NodeIPv4: "10.0.0.1",
					},
					{
						ID:       "Master2",
						NodeIPv4: "10.0.0.2",
					},
				},
			},
			want: map[string]string{
				"Master1": "10.0.0.1",
				"Master2": "10.0.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetMasterNodeClusterIP(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMasterNodeClusterIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetWorkerNodeClusterIP(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "base",
			fields: fields{
				Workers: NodeList{
					{
						ID:       "Node1",
						NodeIPv4: "10.0.0.1",
					},
					{
						ID:       "Node2",
						NodeIPv4: "10.0.0.2",
					},
				},
			},
			want: map[string]string{
				"Node1": "10.0.0.1",
				"Node2": "10.0.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetWorkerNodeClusterIP(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkerNodeClusterIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_GetAvailableMasterNodes(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "base",
			fields: fields{
				ControlPlaneStatus: []v1.ControlPlaneHealth{
					{
						ID:     "Master1",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master2",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master3",
						Status: v1.ComponentUnhealthy,
					},
				},
			},
			want: []string{"Master1", "Master2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.GetAvailableMasterNodes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAvailableMasterNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtraMetadata_IsAllMasterAvailable(t *testing.T) {
	type fields struct {
		Masters            NodeList
		Workers            NodeList
		ClusterStatus      v1.ClusterPhase
		Offline            bool
		LocalRegistry      string
		CRI                string
		ClusterName        string
		KubeVersion        string
		OperationID        string
		OperationType      string
		KubeletDataDir     string
		ControlPlaneStatus []v1.ControlPlaneHealth
		Addons             []v1.Addon
		CNI                string
		CNINamespace       string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "base",
			fields: fields{
				ControlPlaneStatus: []v1.ControlPlaneHealth{
					{
						ID:     "Master1",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master2",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master3",
						Status: v1.ComponentHealthy,
					},
				},
			},
			want: true,
		},
		{
			name: "There exist unhealthy cluster",
			fields: fields{
				ControlPlaneStatus: []v1.ControlPlaneHealth{
					{
						ID:     "Master1",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master2",
						Status: v1.ComponentHealthy,
					},
					{
						ID:     "Master3",
						Status: v1.ComponentUnhealthy,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ExtraMetadata{
				Masters:            tt.fields.Masters,
				Workers:            tt.fields.Workers,
				ClusterStatus:      tt.fields.ClusterStatus,
				Offline:            tt.fields.Offline,
				LocalRegistry:      tt.fields.LocalRegistry,
				CRI:                tt.fields.CRI,
				ClusterName:        tt.fields.ClusterName,
				KubeVersion:        tt.fields.KubeVersion,
				OperationID:        tt.fields.OperationID,
				OperationType:      tt.fields.OperationType,
				KubeletDataDir:     tt.fields.KubeletDataDir,
				ControlPlaneStatus: tt.fields.ControlPlaneStatus,
				Addons:             tt.fields.Addons,
				CNI:                tt.fields.CNI,
				CNINamespace:       tt.fields.CNINamespace,
			}
			if got := e.IsAllMasterAvailable(); got != tt.want {
				t.Errorf("IsAllMasterAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
