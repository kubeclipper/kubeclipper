package clusteroperation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kubeclipper/kubeclipper/pkg/clustermanage/kubeadm"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
)

const (
	NodesOperationAdd    = "add"
	NodesOperationRemove = "remove"
)

var (
	ErrInvalidNodesOperation      = errors.New("invalid nodes patch operation")
	ErrInvalidNodesRole           = errors.New("invalid node role")
	ErrZeroNode                   = errors.New("zero node")
	ErrUninstallNotExistComponent = errors.New("the component is not installed in the current cluster")
	ErrInstallExistingComponent   = errors.New("the component has been installed in the current cluster")
)

type NodesPatchOperation string

type PatchNodes struct {
	Operation    NodesPatchOperation   `json:"operation"`
	Nodes        corev1.WorkerNodeList `json:"nodes"`
	ConvertNodes []component.Node      `json:"convertNodes"`
	Role         common.NodeRole       `json:"role"`
}

var _ Interface = (*NodeOperation)(nil)

type NodeOperation struct{}

func (n *NodeOperation) Builder(options Options) (*corev1.Operation, error) {
	var pn PatchNodes
	if err := json.Unmarshal(options.pendingOperation.ExtraData, &pn); err != nil {
		return nil, err
	}
	// Add the worker node to extra
	if pn.Role == common.NodeRoleWorker {
		options.extra.Workers = append(options.extra.Workers, pn.ConvertNodes...)
	}
	op, err := pn.MakeOperation(*options.extra, options.cluster)
	if err != nil {
		return nil, err
	}

	op.Labels[common.LabelTimeoutSeconds] = options.pendingOperation.Timeout
	op.Labels[common.LabelOperationID] = options.pendingOperation.OperationID
	op.Status.Status = corev1.OperationStatusPending

	return op, nil
}

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

// MakeOperation Must be called after MakeCompare.
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
		// remove nodes from cluster
		// filter nodes in cluster
		// nodes to be removed
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
