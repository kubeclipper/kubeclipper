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
	"errors"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage/kubeadm"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type (
	NodesPatchOperation string
)

const (
	NodesOperationAdd    = "add"
	NodesOperationRemove = "remove"
)

type PatchNodes struct {
	Operation NodesPatchOperation   `json:"operation"`
	Nodes     corev1.WorkerNodeList `json:"nodes"`
	Role      common.NodeRole       `json:"role"`
}

type PatchComponents struct {
	Uninstall bool           `json:"uninstall"`
	Addons    []corev1.Addon `json:"addons"`
}

var (
	ErrInvalidNodesOperation      = errors.New("invalid nodes patch operation")
	ErrInvalidNodesRole           = errors.New("invalid node role")
	ErrZeroNode                   = errors.New("zero node")
	ErrUninstallNotExistComponent = errors.New("the component is not installed in the current cluster")
	ErrInstallExistingComponent   = errors.New("the component has been installed in the current cluster")
)

// MakeCompare compares and filters node to be operated with master/worker nodes in cluster already,
// it also modifies master/worker nodes in cluster object.
func (p *PatchNodes) MakeCompare(cluster *corev1.Cluster) error {
	switch p.Role {
	case common.NodeRoleMaster:
		return p.makeMasterCompare(cluster)
	case common.NodeRoleWorker:
		return p.makeWorkerCompare(cluster)
	default:
		return ErrInvalidNodesRole
	}
}

func (p *PatchNodes) makeMasterCompare(cluster *corev1.Cluster) error {
	// support later
	return ErrInvalidNodesRole
}

func (p *PatchNodes) makeWorkerCompare(cluster *corev1.Cluster) error {
	// Compare between nodes unfiltered and existed worker nodes in cluster.
	switch p.Operation {
	case NodesOperationAdd:
		// Add nodes to cluster.
		// Check nodes in cluster already.
		// Filter out nodes to be added.
		p.Nodes = p.Nodes.Complement(cluster.Workers...)
		cluster.Workers = append(cluster.Workers, p.Nodes...)
	case NodesOperationRemove:
		// Remove nodes from cluster.
		// Filter out nodes in cluster already.
		// Filter out nodes to be removed.
		// TODO: if len(p.nodes)==0, should return error
		p.Nodes = cluster.Workers.Intersect(p.Nodes...)
		cluster.Workers = cluster.Workers.Complement(p.Nodes...)
	default:
		return ErrInvalidNodesOperation
	}
	return nil
}

// Must be called after MakeCompare.
func (p *PatchNodes) MakeOperation(extra component.ExtraMetadata, cluster *corev1.Cluster) (*corev1.Operation, error) {
	pType, ok := cluster.Labels[common.LabelClusterProviderType]
	if ok {
		switch pType {
		case kubeadm.ProviderKubeadm:
			return p.doMakeOperation(extra, cluster)
		default:
			return nil, errors.New("invalid provider type")
		}
	}
	return p.doMakeOperation(extra, cluster)
}

func (p *PatchNodes) doMakeOperation(extra component.ExtraMetadata, cluster *corev1.Cluster) (*corev1.Operation, error) {
	switch p.Role {
	case common.NodeRoleMaster:
		return nil, ErrInvalidNodesRole
	case common.NodeRoleWorker:
		return p.makeWorkerOperation(extra, cluster)
	default:
		return nil, ErrInvalidNodesRole
	}
}

func (p *PatchNodes) makeWorkerOperation(extra component.ExtraMetadata, cluster *corev1.Cluster) (*corev1.Operation, error) {
	// no node need to be operated
	if len(p.Nodes) == 0 {
		return nil, ErrZeroNode
	}

	// make operation for adding worker nodes to cluster
	op := &corev1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = map[string]string{
		common.LabelClusterName: cluster.Name,
		// v1.LabelTopologyRegion ???
	}
	ctx := context.TODO()
	ctx = component.WithExtraMetadata(ctx, extra)
	stepNodes := []corev1.StepNode{}
	workerIPs := extra.GetWorkerNodeIP()
	for _, nodeID := range p.Nodes.GetNodeIDs() {
		stepNode := corev1.StepNode{
			ID:       nodeID,
			IPv4:     workerIPs[nodeID],
			Hostname: extra.GetWorkerHostname(nodeID),
		}
		stepNodes = append(stepNodes, stepNode)
	}

	var action corev1.StepAction
	switch p.Operation {
	case NodesOperationAdd:
		// add nodes to cluster
		// check nodes in cluster
		// nodes to be added

		// p.Nodes = cluster.Workers.Complement(p.Nodes...)
		// cluster.Workers = append(cluster.Workers, p.Nodes...)
		action = corev1.ActionInstall
		op.Labels[common.LabelOperationAction] = corev1.OperationAddNodes

		// container runtime
		steps, err := getCriStep(ctx, cluster, action, stepNodes)
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, steps...)

		// kubernetes
		steps, err = p.getPackageSteps(cluster, action, stepNodes)
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, steps...)

		// join component
		gen := k8s.GenNode{}
		err = gen.InitStepper(&extra, cluster, p.Role.String()).MakeInstallSteps(&extra, stepNodes, p.Role.String())
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, gen.GetSteps(action)...)
	case NodesOperationRemove:
		// // remove nodes from cluster
		// // filter nodes in cluster
		// // nodes to be removed
		// p.Nodes = cluster.Workers.Intersect(p.Nodes...)
		// cluster.Workers = cluster.Workers.Complement(p.Nodes...)
		action = corev1.ActionUninstall
		op.Labels[common.LabelOperationAction] = corev1.OperationRemoveNodes

		gen := k8s.GenNode{}
		err := gen.InitStepper(&extra, cluster, p.Role.String()).MakeUninstallSteps(&extra, stepNodes)
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, gen.GetSteps(action)...)

		// kubernetes
		steps, err := p.getPackageSteps(cluster, action, stepNodes)
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, steps...)

		// container runtime
		steps, err = getCriStep(ctx, cluster, action, stepNodes)
		if err != nil {
			return nil, err
		}
		op.Steps = append(op.Steps, steps...)
	default:
		return nil, ErrInvalidNodesOperation
	}

	return op, nil
}

func (p *PatchNodes) getPackageSteps(cluster *corev1.Cluster, action corev1.StepAction, pNodes []corev1.StepNode) ([]corev1.Step, error) {
	pack := &k8s.Package{}
	pack = pack.InitStepper(cluster)

	switch action {
	case corev1.ActionInstall:
		return pack.InstallSteps(pNodes)
	case corev1.ActionUninstall:
		return pack.UninstallSteps(pNodes)
	}

	return nil, fmt.Errorf("packageSteps dose not support action: %s", action)
}

// checkComponent check whether the component is installed in the current cluster
func (p *PatchComponents) checkComponents(cluster *corev1.Cluster) error {
	for _, v := range p.Addons {
		itf, ok := component.Load(fmt.Sprintf(component.RegisterFormat, v.Name, v.Version))
		if !ok {
			return fmt.Errorf("kubeclipper does not support %s-%s component", v.Name, v.Version)
		}
		meta := itf.GetComponentMeta(component.English)
		var installed bool
		for _, comp := range cluster.Addons {
			if comp.Name == v.Name {
				installed = true
			}
		}
		if p.Uninstall && !installed {
			return fmt.Errorf("%s-%s component is not installed in the current cluster", v.Name, v.Version)
		}
		// add component and component is unique
		if !p.Uninstall && installed && meta.Unique {
			return fmt.Errorf("%s-%s component has been installed in the current cluster", v.Name, v.Version)
		}
		// add component and component is not unique, check whether component StorageClassName is equal
		// TODO: Make validation logic specific to component configurations
		if !p.Uninstall && installed && !meta.Unique {
			var existedComps []corev1.Addon
			// components of the same type already exist in the collection cluster
			for _, comp := range cluster.Addons {
				if comp.Name == v.Name {
					existedComps = append(existedComps, comp)
				}
			}
			// resolves the config of current component to be operated
			currentCompMeta := itf.NewInstance()
			if err := json.Unmarshal(v.Config.Raw, currentCompMeta); err != nil {
				return fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
			}
			currentNewComp, _ := currentCompMeta.(component.Interface)
			for _, existedComp := range existedComps {
				// resolve that the cluster already has the same type of component config
				existCompItf, _ := component.Load(fmt.Sprintf(component.RegisterFormat, existedComp.Name, existedComp.Version))
				existCompMeta := existCompItf.NewInstance()
				if err := json.Unmarshal(existedComp.Config.Raw, existCompMeta); err != nil {
					return fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
				}
				existNewComp, _ := existCompMeta.(component.Interface)
				if currentNewComp.GetInstanceName() == existNewComp.GetInstanceName() {
					return fmt.Errorf("%s-%s component StorageClassName must be unique", v.Name, v.Version)
				}
			}
		}
	}
	return nil
}

// addOrRemoveComponentFromCluster update cluster components slice
func (p *PatchComponents) addOrRemoveComponentFromCluster(cluster *corev1.Cluster) (*corev1.Cluster, error) {
	if p.Uninstall {
		for _, v := range p.Addons {
			itf, ok := component.Load(fmt.Sprintf(component.RegisterFormat, v.Name, v.Version))
			if !ok {
				return nil, fmt.Errorf("kubeclipper does not support %s-%s component", v.Name, v.Version)
			}
			currentCompMeta := itf.NewInstance()
			if err := json.Unmarshal(v.Config.Raw, currentCompMeta); err != nil {
				return nil, fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
			}
			currentNewComp, _ := currentCompMeta.(component.Interface)
			for k, comp := range cluster.Addons {
				existCompItf, _ := component.Load(fmt.Sprintf(component.RegisterFormat, comp.Name, comp.Version))
				existCompMeta := existCompItf.NewInstance()
				if err := json.Unmarshal(comp.Config.Raw, existCompMeta); err != nil {
					return nil, fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
				}
				existNewComp, _ := existCompMeta.(component.Interface)
				if currentNewComp.GetInstanceName() == existNewComp.GetInstanceName() {
					cluster.Addons = append(cluster.Addons[:k], cluster.Addons[k+1:]...)
				}
			}
		}
		return cluster, nil
	}
	cluster.Addons = append(cluster.Addons, p.Addons...)
	return cluster, nil
}

type StepLog struct {
	Content      string          `json:"content,omitempty"`
	Node         string          `json:"node,omitempty"`
	Timeout      metav1.Duration `json:"timeout,omitempty"`
	Status       string          `json:"status,omitempty"`
	DeliverySize int64           `json:"deliverySize,omitempty"`
	LogSize      int64           `json:"logSize,omitempty"`
}

type ClusterUpgrade struct {
	Version       string `json:"version"`
	Offline       bool   `json:"offline"`
	LocalRegistry string `json:"localRegistry"`
}
