package operationv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/agent/operationv2"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

type NodeReader interface {
	GetNodeEx(ctx context.Context, name, resourceVersion string) (*corev1.Node, error)
}

type OperationWriter interface {
	CreateOperation(ctx context.Context, op *operations.Operation) (*operations.Operation, error)
}

// CreateFromCore converts a business plan and persists only its v2 Operation.
// Business code can keep building the existing Step representation while the
// execution and status model remains entirely owned by Operation Engine v2.
func CreateFromCore(ctx context.Context, writer OperationWriter, nodes NodeReader, cluster *corev1.Cluster, plan *corev1.Operation) (*operations.Operation, error) {
	op, err := FromCoreOperation(ctx, plan, cluster, nodes)
	if err != nil {
		return nil, err
	}
	return writer.CreateOperation(ctx, op)
}

// FromCoreOperation converts the existing business plan representation into
// the v2 execution resource. The returned object is the only Operation that
// should be persisted.
func FromCoreOperation(ctx context.Context, plan *corev1.Operation, cluster *corev1.Cluster, nodes NodeReader) (*operations.Operation, error) {
	if plan == nil || cluster == nil || plan.Name == "" || cluster.Name == "" || cluster.UID == "" {
		return nil, fmt.Errorf("operation name and Cluster name/UID are required")
	}
	timeout := operations.DefaultOperationTimeout
	if raw := plan.Labels[common.LabelTimeoutSeconds]; raw != "" {
		seconds, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || seconds <= 0 {
			return nil, fmt.Errorf("invalid operation timeout %q", raw)
		}
		timeout = time.Duration(seconds) * time.Second
	}
	nodeCache := make(map[string]*corev1.Node)
	steps := make([]operations.OperationStep, 0, len(plan.Steps))
	for index := range plan.Steps {
		step := plan.Steps[index].DeepCopy()
		if step.ID == "" {
			return nil, fmt.Errorf("step %d has no ID", index)
		}
		targets := make([]operations.NodeReference, 0, len(step.Nodes))
		for _, target := range step.Nodes {
			node := nodeCache[target.ID]
			if node == nil {
				var err error
				node, err = nodes.GetNodeEx(ctx, target.ID, "")
				if err != nil {
					return nil, fmt.Errorf("resolve Node %q for step %q: %w", target.ID, step.ID, err)
				}
				nodeCache[target.ID] = node
			}
			targets = append(targets, operations.NodeReference{Name: node.Name, UID: node.UID})
		}
		step.Nodes = nil
		payload, err := json.Marshal(operationv2.CommandStepPayload{Step: *step})
		if err != nil {
			return nil, err
		}
		retryLimit := step.RetryTimes
		if retryLimit > operations.MaxRetryLimit {
			retryLimit = operations.MaxRetryLimit
		}
		converted := operations.OperationStep{
			ID: step.ID, Targets: targets, Executor: operationv2.CommandStepExecutorName,
			Payload: runtime.RawExtension{Raw: payload}, RetryLimit: retryLimit,
		}
		if index > 0 && len(steps[index-1].Targets) > 0 {
			converted.Inputs = []operations.StepInput{{
				Field: "lastTaskReply", FromStepID: steps[index-1].ID,
				FromNodeUID: steps[index-1].Targets[0].UID, OutputKey: "response",
			}}
		}
		steps = append(steps, converted)
	}
	labels := make(map[string]string, len(plan.Labels))
	for key, value := range plan.Labels {
		labels[key] = value
	}
	return &operations.Operation{
		TypeMeta:   metav1.TypeMeta{APIVersion: operations.SchemeGroupVersion.String(), Kind: operations.KindOperation},
		ObjectMeta: metav1.ObjectMeta{Name: plan.Name, Labels: labels},
		Spec: operations.OperationSpec{
			TargetRef: operations.ObjectReference{Kind: corev1.KindCluster, Name: cluster.Name, UID: cluster.UID},
			Action:    plan.Labels[common.LabelOperationAction], DesiredState: operations.OperationDesiredStateActive,
			Timeout: metav1.Duration{Duration: timeout}, Steps: steps,
		},
		Status: operations.OperationStatus{Phase: operations.OperationPending},
	}, nil
}
