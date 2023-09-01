package kc

import (
	"reflect"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestNodesList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Node
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Node{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e7520b25-5528-4eb9-9ab9-56e3b7c44ba5",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e694294e-1800-429e-8bdb-b264704e0d6b",
						},
					},
				},
				TotalCount: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodesList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodesList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Node
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Node{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e7520b25-5528-4eb9-9ab9-56e3b7c44ba5",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e694294e-1800-429e-8bdb-b264704e0d6b",
						},
					},
				},
				TotalCount: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodesList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestNodesList_TablePrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Node
		TotalCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Node{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e7520b25-5528-4eb9-9ab9-56e3b7c44ba5",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "e694294e-1800-429e-8bdb-b264704e0d6b",
						},
					},
				},
				TotalCount: 2,
			},
			want: []string{"ID", "hostname", "region", "ip", "os/arch", "cpu", "mem"},
			want1: [][]string{
				{"e7520b25-5528-4eb9-9ab9-56e3b7c44ba5", "", "", "", "/", "0", "0"},
				{"e694294e-1800-429e-8bdb-b264704e0d6b", "", "", "", "/", "0", "0"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NodesList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestUsersList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.User
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UsersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestUsersList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.User
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UsersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestUsersList_TablePrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.User
		TotalCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.User{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "user2",
						},
					},
				},
			},
			want: []string{"name", "display_name", "email", "phone", "role", "state", "create_timestamp"},
			want1: [][]string{
				{"user1", "", "", "", "", "", "0001-01-01 00:00:00 +0000 UTC"},
				{"user2", "", "", "", "", "", "0001-01-01 00:00:00 +0000 UTC"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UsersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestUsersList_userState(t *testing.T) {
	sName := "California"
	sta := &sName
	type fields struct {
		Items      []iamv1.User
		TotalCount int
	}
	type args struct {
		state *iamv1.UserState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "state is not empty",
			fields: fields{},
			args: args{
				state: (*iamv1.UserState)(sta),
			},
			want: *sta,
		},
		{
			name:   "state is empty",
			fields: fields{},
			args: args{
				state: nil,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UsersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			if got := n.userState(tt.args.state); got != tt.want {
				t.Errorf("userState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClustersList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Cluster
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Cluster{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ClustersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClustersList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Cluster
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Cluster{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "cluster2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ClustersList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRoleList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.GlobalRole
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.GlobalRole{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &RoleList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRoleList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.GlobalRole
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.GlobalRole{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &RoleList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRoleList_TablePrint(t *testing.T) {
	type fields struct {
		Items      []iamv1.GlobalRole
		TotalCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base",
			fields: fields{
				Items: []iamv1.GlobalRole{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "role2",
						},
					},
				},
			},
			want: []string{"name", "is_template", "internal", "hidden", "create_timestamp"},
			want1: [][]string{
				{"role1", "false", "false", "false", "0001-01-01 00:00:00 +0000 UTC"},
				{"role2", "false", "false", "false", "0001-01-01 00:00:00 +0000 UTC"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &RoleList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_isMapKeyExist(t *testing.T) {
	type args struct {
		maps map[string]string
		key  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "key exist",
			args: args{
				maps: map[string]string{
					"key1": "value1",
				},
				key: "key1",
			},
			want: "value1",
		},
		{
			name: "key is not exist",
			args: args{
				maps: map[string]string{
					"key1": "value1",
				},
				key: "key2",
			},
			want: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMapKeyExist(tt.args.maps, tt.args.key); got != tt.want {
				t.Errorf("isMapKeyExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComponentMetas_YAMLPrint(t *testing.T) {
	type fields struct {
		Node            string
		PackageMetadata scheme.PackageMetadata
		TotalCount      int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Node: "e694294e-1800-429e-8bdb-b264704e0d6b",
				PackageMetadata: scheme.PackageMetadata{
					Addons: []scheme.MetaResource{
						{
							Name: "nfs-csi",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ComponentMetas{
				Node:            tt.fields.Node,
				PackageMetadata: tt.fields.PackageMetadata,
				TotalCount:      tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestComponentMetas_JSONPrint(t *testing.T) {
	type fields struct {
		Node            string
		PackageMetadata scheme.PackageMetadata
		TotalCount      int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Node: "e694294e-1800-429e-8bdb-b264704e0d6b",
				PackageMetadata: scheme.PackageMetadata{
					Addons: []scheme.MetaResource{
						{
							Name: "nfs-csi",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ComponentMetas{
				Node:            tt.fields.Node,
				PackageMetadata: tt.fields.PackageMetadata,
				TotalCount:      tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestComponentMetas_TablePrint(t *testing.T) {
	type fields struct {
		Node            string
		PackageMetadata scheme.PackageMetadata
		TotalCount      int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base",
			fields: fields{
				Node: "e694294e-1800-429e-8bdb-b264704e0d6b",
				PackageMetadata: scheme.PackageMetadata{
					Addons: []scheme.MetaResource{
						{
							Name: "nfs-csi",
						},
					},
				},
			},
			want: []string{"e694294e-1800-429e-8bdb-b264704e0d6b", "type", "name", "version", "arch"},
			want1: [][]string{
				{"1.", "", "nfs-csi", "", ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ComponentMetas{
				Node:            tt.fields.Node,
				PackageMetadata: tt.fields.PackageMetadata,
				TotalCount:      tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestConfigMapList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.ConfigMap
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.ConfigMap{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ConfigMapList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestConfigMapList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.ConfigMap
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.ConfigMap{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ConfigMapList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestConfigMapList_TablePrint(t *testing.T) {
	type fields struct {
		Items      []corev1.ConfigMap
		TotalCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.ConfigMap{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "configmap2",
						},
					},
				},
			},
			want: []string{"name", "create_timestamp"},
			want1: [][]string{
				{"configmap1", "0001-01-01 00:00:00 +0000 UTC"},
				{"configmap2", "0001-01-01 00:00:00 +0000 UTC"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &ConfigMapList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTemplateList_YAMLPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Template
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Template{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &TemplateList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.YAMLPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestTemplateList_JSONPrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Template
		TotalCount int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Items: []corev1.Template{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &TemplateList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			_, err := n.JSONPrint()
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONPrint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestTemplateList_TablePrint(t *testing.T) {
	type fields struct {
		Items      []corev1.Template
		TotalCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
		want1  [][]string
	}{
		{
			name: "base test",
			fields: fields{
				Items: []corev1.Template{
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template1",
						},
					},
					{
						ObjectMeta: v1.ObjectMeta{
							Name: "template2",
						},
					},
				},
			},
			want: []string{"name", "create_timestamp"},
			want1: [][]string{
				{"template1", "0001-01-01 00:00:00 +0000 UTC"},
				{"template2", "0001-01-01 00:00:00 +0000 UTC"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &TemplateList{
				Items:      tt.fields.Items,
				TotalCount: tt.fields.TotalCount,
			}
			got, got1 := n.TablePrint()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TablePrint() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("TablePrint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
