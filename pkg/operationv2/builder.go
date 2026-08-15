package operationv2

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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
func CreateFromCore(
	ctx context.Context,
	writer OperationWriter,
	nodes NodeReader,
	cluster *corev1.Cluster,
	plan *corev1.Operation,
) (*operations.Operation, error) {
	op, err := FromCoreOperation(ctx, plan, cluster, nodes)
	if err != nil {
		return nil, err
	}
	return writer.CreateOperation(ctx, op)
}

// FromCoreOperation converts the existing business plan representation into
// the v2 execution resource. The returned object is the only Operation that
// should be persisted.
func FromCoreOperation(
	ctx context.Context,
	plan *corev1.Operation,
	cluster *corev1.Cluster,
	nodes NodeReader,
) (*operations.Operation, error) {
	if err := validateConversionInputs(plan, cluster); err != nil {
		return nil, err
	}
	timeout, err := operationTimeout(plan)
	if err != nil {
		return nil, err
	}
	steps, err := convertSteps(ctx, plan.Steps, nodes)
	if err != nil {
		return nil, err
	}
	labels := make(map[string]string, len(plan.Labels))
	maps.Copy(labels, plan.Labels)
	return newOperation(plan, cluster, labels, timeout, steps), nil
}

func validateConversionInputs(plan *corev1.Operation, cluster *corev1.Cluster) error {
	if plan == nil || cluster == nil || plan.Name == "" || cluster.Name == "" || cluster.UID == "" {
		return fmt.Errorf("operation name and Cluster name/UID are required")
	}
	return nil
}

func operationTimeout(plan *corev1.Operation) (time.Duration, error) {
	timeout := operations.DefaultOperationTimeout
	if raw := plan.Labels[common.LabelTimeoutSeconds]; raw != "" {
		seconds, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || seconds <= 0 {
			return 0, fmt.Errorf("invalid operation timeout %q", raw)
		}
		timeout = time.Duration(seconds) * time.Second
	}
	return timeout, nil
}

func convertSteps(ctx context.Context, source []corev1.Step, nodes NodeReader) ([]operations.OperationStep, error) {
	nodeCache := make(map[string]*corev1.Node)
	steps := make([]operations.OperationStep, 0, len(source))
	for index := range source {
		converted, err := convertStep(ctx, source[index].DeepCopy(), nodes, nodeCache)
		if err != nil {
			return nil, err
		}
		addPreviousStepInput(&converted, steps)
		steps = append(steps, converted)
	}
	return steps, nil
}

func convertStep(
	ctx context.Context,
	step *corev1.Step,
	nodes NodeReader,
	cache map[string]*corev1.Node,
) (operations.OperationStep, error) {
	if step.ID == "" {
		return operations.OperationStep{}, fmt.Errorf("step has no ID")
	}
	targets, err := resolveTargets(ctx, step, nodes, cache)
	if err != nil {
		return operations.OperationStep{}, err
	}
	step.Nodes = nil
	payload, err := json.Marshal(operationv2.CommandStepPayload{Step: *step})
	if err != nil {
		return operations.OperationStep{}, err
	}
	return operations.OperationStep{ID: step.ID, Targets: targets, Executor: operationv2.CommandStepExecutorName,
		Payload: runtime.RawExtension{Raw: payload}, RetryLimit: min(step.RetryTimes, operations.MaxRetryLimit)}, nil
}

func resolveTargets(
	ctx context.Context,
	step *corev1.Step,
	nodes NodeReader,
	cache map[string]*corev1.Node,
) ([]operations.NodeReference, error) {
	targets := make([]operations.NodeReference, 0, len(step.Nodes))
	for _, target := range step.Nodes {
		node := cache[target.ID]
		if node == nil {
			var err error
			node, err = nodes.GetNodeEx(ctx, target.ID, "")
			if err != nil {
				return nil, fmt.Errorf("resolve Node %q for step %q: %w", target.ID, step.ID, err)
			}
			cache[target.ID] = node
		}
		targets = append(targets, operations.NodeReference{Name: node.Name, UID: node.UID})
	}
	return targets, nil
}

func addPreviousStepInput(step *operations.OperationStep, steps []operations.OperationStep) {
	if len(steps) == 0 || len(steps[len(steps)-1].Targets) == 0 {
		return
	}
	previous := steps[len(steps)-1]
	step.Inputs = []operations.StepInput{{Field: "lastTaskReply", FromStepID: previous.ID,
		FromNodeUID: previous.Targets[0].UID, OutputKey: "response"}}
}

func newOperation(
	plan *corev1.Operation,
	cluster *corev1.Cluster,
	labels map[string]string,
	timeout time.Duration,
	steps []operations.OperationStep,
) *operations.Operation {
	return &operations.Operation{
		TypeMeta:   metav1.TypeMeta{APIVersion: operations.SchemeGroupVersion.String(), Kind: operations.KindOperation},
		ObjectMeta: metav1.ObjectMeta{Name: plan.Name, Labels: labels},
		Spec: operations.OperationSpec{
			TargetRef: operations.ObjectReference{Kind: corev1.KindCluster, Name: cluster.Name, UID: cluster.UID},
			Action:    plan.Labels[common.LabelOperationAction], DesiredState: operations.OperationDesiredStateActive,
			Timeout: metav1.Duration{Duration: timeout}, Steps: steps,
		},
		Status: operations.OperationStatus{Phase: operations.OperationPending},
	}
}
