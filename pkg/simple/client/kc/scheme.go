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

package kc

import (
	"fmt"
	"strconv"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"

	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var _ printer.ResourcePrinter = (*NodesList)(nil)

type NodesList struct {
	Items      []v1.Node `json:"items" description:"paging data"`
	TotalCount int       `json:"totalCount,omitempty" description:"total count"`
}

func (n *NodesList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

func (n *NodesList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *NodesList) TablePrint() ([]string, [][]string) {
	headers := []string{"ID", "hostname", "region", "ip", "os/arch", "cpu", "mem"}
	var data [][]string
	for _, node := range n.Items {
		cpu := node.Status.Capacity[v1.ResourceCPU]
		mem := node.Status.Capacity[v1.ResourceMemory]
		data = append(data, []string{node.Name,
			node.Labels[common.LabelHostname],
			node.Labels[common.LabelTopologyRegion],
			node.Status.Ipv4DefaultIP,
			fmt.Sprintf("%s/%s", node.Labels[common.LabelOSStable], node.Labels[common.LabelArchStable]),
			cpu.String(),
			mem.String()})
	}
	return headers, data
}

var _ printer.ResourcePrinter = (*UsersList)(nil)

type UsersList struct {
	Items      []iamv1.User `json:"items" description:"paging data"`
	TotalCount int          `json:"totalCount,omitempty" description:"total count"`
}

func (n *UsersList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *UsersList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "display_name", "email", "phone", "role", "state", "create_timestamp"}
	var data [][]string
	for _, user := range n.Items {
		data = append(data, []string{user.Name,
			user.Spec.DisplayName,
			user.Spec.Email,
			user.Spec.Phone,
			user.Annotations[common.RoleAnnotation],
			n.userState(user.Status.State),
			user.CreationTimestamp.String()})
	}
	return headers, data
}

func (n *UsersList) userState(state *iamv1.UserState) string {
	if state == nil {
		return ""
	}
	return string(*state)
}

func (n *UsersList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

var _ printer.ResourcePrinter = (*ClustersList)(nil)

type ClustersList struct {
	Items      []v1.Cluster `json:"items" description:"paging data"`
	TotalCount int          `json:"totalCount,omitempty" description:"total count"`
}

func (n *ClustersList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

func (n *ClustersList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "region", "master_count", "worker_count", "status", "apiserver_certs_expiration", "create_timestamp"}
	var data [][]string
	for _, cluster := range n.Items {
		var ace string
		for _, cert := range cluster.Status.Certifications {
			if cert.Name == "apiserver" {
				ace = cert.ExpirationTime.String()
			}
		}
		data = append(data, []string{cluster.Name,
			cluster.Labels[common.LabelTopologyRegion],
			strconv.FormatInt(int64(len(cluster.Masters)), 10),
			strconv.FormatInt(int64(len(cluster.Workers)), 10),
			string(cluster.Status.Phase),
			ace,
			cluster.CreationTimestamp.String()})
	}
	return headers, data
}

func (n *ClustersList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

var _ printer.ResourcePrinter = (*RoleList)(nil)

type RoleList struct {
	Items      []iamv1.GlobalRole `json:"items" description:"paging data"`
	TotalCount int                `json:"totalCount,omitempty" description:"total count"`
}

func (n *RoleList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

func (n *RoleList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "is_template", "internal", "hidden", "create_timestamp"}
	var data [][]string
	for _, role := range n.Items {
		data = append(data, []string{role.Name,
			isMapKeyExist(role.Labels, common.LabelRoleTemplate),
			isMapKeyExist(role.Annotations, common.AnnotationInternal),
			isMapKeyExist(role.Labels, common.LabelHidden),
			role.CreationTimestamp.String()})
	}
	return headers, data
}

func isMapKeyExist(maps map[string]string, key string) string {
	if r, ok := maps[key]; ok {
		return r
	}
	return "false"
}

func (n *RoleList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

type ComponentMetas struct {
	Node                   string `json:"-"`
	scheme.PackageMetadata `json:"items" description:"paging data"`
	TotalCount             int `json:"totalCount,omitempty" description:"total count"`
}

func (n *ComponentMetas) JSONPrint() ([]byte, error) {
	if len(n.PackageMetadata.Addons) == 1 {
		return printer.JSONPrinter(n.PackageMetadata.Addons[0])
	}
	return printer.JSONPrinter(n)
}

func (n *ComponentMetas) YAMLPrint() ([]byte, error) {
	if len(n.PackageMetadata.Addons) == 1 {
		return printer.YAMLPrinter(n.PackageMetadata.Addons[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *ComponentMetas) TablePrint() ([]string, [][]string) {
	n.PackageMetadata.AddonsSort()
	headers := []string{n.Node, "type", "name", "version", "arch"}
	var data [][]string
	for index, resource := range n.PackageMetadata.Addons {
		data = append(data, []string{fmt.Sprintf("%d.", index+1), resource.Type, resource.Name, resource.Version, resource.Arch})
	}
	return headers, data
}

type ComponentMeta struct {
	Rules  []map[string]interface{} `json:"rules"`
	Addons []scheme.MetaResource    `json:"addons"`
}

type BackupList struct {
	Items      []v1.Backup `json:"items" description:"paging data"`
	TotalCount int         `json:"totalCount,omitempty" description:"total count"`
}

type BackupPointList struct {
	Items      []v1.BackupPoint `json:"items" description:"paging data"`
	TotalCount int              `json:"totalCount,omitempty" description:"total count"`
}

type ConfigMapList struct {
	Items      []v1.ConfigMap `json:"items" description:"paging data"`
	TotalCount int            `json:"totalCount,omitempty" description:"total count"`
}

func (n *ConfigMapList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

func (n *ConfigMapList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *ConfigMapList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "create_timestamp"}
	var data [][]string
	for _, cm := range n.Items {
		data = append(data, []string{cm.Name, cm.CreationTimestamp.String()})
	}
	return headers, data
}

type TemplateList struct {
	Items      []v1.Template `json:"items" description:"paging data"`
	TotalCount int           `json:"totalCount,omitempty" description:"total count"`
}

func (n *TemplateList) JSONPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.JSONPrinter(n.Items[0])
	}
	return printer.JSONPrinter(n)
}

func (n *TemplateList) YAMLPrint() ([]byte, error) {
	if len(n.Items) == 1 {
		return printer.YAMLPrinter(n.Items[0])
	}
	return printer.YAMLPrinter(n)
}

func (n *TemplateList) TablePrint() ([]string, [][]string) {
	headers := []string{"name", "create_timestamp"}
	var data [][]string
	for _, cm := range n.Items {
		data = append(data, []string{cm.Name, cm.CreationTimestamp.String()})
	}
	return headers, data
}

var _ printer.ResourcePrinter = (*OperationList)(nil)

type OperationList struct {
	Items      []v1.Operation `json:"items" description:"paging data"`
	TotalCount int            `json:"totalCount,omitempty" description:"total count"`
}

func (n *OperationList) YAMLPrint() ([]byte, error) {
	panic("implement me")
}

func (n *OperationList) TablePrint() ([]string, [][]string) {
	panic("implement me")
}

func (n *OperationList) JSONPrint() ([]byte, error) {
	panic("implement me")
}
