package clusteroperation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// RetryTimes max number of operation retry
// TODO: Later, if necessary, can set a separate retry number for each operation ?
const RetryTimes = 3

type Options struct {
	cluster           *v1.Cluster
	pendingOperation  v1.PendingOperation
	extra             *component.ExtraMetadata
	additionalOptions *AdditionalOptions // additional information
}

// AdditionalOptions some types of operations require additional information, which can be put here
type AdditionalOptions struct{}

// Interface all operation types must implement this interface
type Interface interface {
	Builder() (*v1.Operation, error)
}

type OptionInterface interface {
}

// BuildOperationAdapter create an operation based on the operation type
func BuildOperationAdapter(
	cluster *v1.Cluster, pendingOp v1.PendingOperation,
	extra *component.ExtraMetadata, addition *AdditionalOptions,
) (*v1.Operation, error) {
	options := Options{
		cluster:           cluster,
		pendingOperation:  pendingOp,
		extra:             extra,
		additionalOptions: addition,
	}

	var instance Interface
	switch pendingOp.OperationType {
	case v1.OperationAddNodes, v1.OperationRemoveNodes:
		instance = NewNodeOperation(options)
	case v1.OperationInstallComponents, v1.OperationUninstallComponents:
	case v1.OperationCreateCluster:
	case v1.OperationDeleteCluster:
	case v1.OperationUpgradeCluster:
	case v1.OperationBackupCluster:
	case v1.OperationDeleteBackup:
	case v1.OperationRecoverCluster:
	case v1.OperationUpdateCertification:
	case v1.OperationUpdateAPIServerCertification:
		// TODO support all operations
	default:
		return &v1.Operation{}, fmt.Errorf("unsupported %s operation type", pendingOp.OperationType)
	}

	return instance.Builder()
}

// IsRetry whether the operation supports retry
func IsRetry(opType string) bool {
	switch opType {
	case v1.OperationBackupCluster, v1.OperationRecoverCluster, v1.OperationUpgradeCluster:
		return false
	}
	return true
}

// AutomaticRetry whether automatic retry is supported
// Automatic retry conditions:
// 1. operation support retry.
// 2. the maximum number of retries has not been exceeded.
// 3. the operation is the latest in the cluster.
// TODO: better retry judgment, perhaps it is made up of global and custom rules ?
func AutomaticRetry(ctx context.Context, op *v1.Operation, operator operation.Operator) (bool, error) {
	if !IsRetry(op.Labels[common.LabelOperationAction]) {
		return false, nil
	}
	times, _ := strconv.Atoi(op.Labels[common.LabelOperationRetry])
	// maximum number of retries exceeded
	if times > RetryTimes {
		return false, nil
	}
	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, op.Labels[common.LabelClusterName])
	q.Pagination.Offset = 0
	q.Pagination.Limit = 1
	// get the latest operations of the cluster
	resp, err := operator.ListOperationsEx(ctx, q)
	if err != nil {
		return false, err
	}
	if resp.TotalCount == 0 {
		return false, fmt.Errorf("no operation record is queried in %s cluster", op.Labels[common.LabelClusterName])
	}
	latestOp := resp.Items[0].(*v1.Operation)
	// if the current operation is not up-to-date, then failure will not be retried
	if latestOp.Name != op.Name {
		return false, nil
	}
	return true, nil
}

// Retry operation retry
func Retry(op *v1.Operation) (context.Context, *v1.Operation, []v1.Step, error) {
	if op.Status.Status == v1.OperationStatusSuccessful || op.Status.Status == v1.OperationStatusRunning {
		return nil, nil, nil, fmt.Errorf("only the latest faild operation can do a retry")
	}

	// error step index
	failedIndex := len(op.Status.Conditions) - 1
	ctx := component.WithRetry(context.TODO(), true)

	// if an installation error occurs, proceed directly from the current step
	var continueSteps []v1.Step
	if op.Steps[0].Action == v1.ActionInstall {
		findStepNode := func(nodes []v1.StepNode, nodeID string) v1.StepNode {
			for _, v := range nodes {
				if v.ID == nodeID {
					return v
				}
			}
			return v1.StepNode{}
		}
		var failedNodes []v1.StepNode
		successStatus := make([]v1.StepStatus, 0)
		for _, status := range op.Status.Conditions[failedIndex].Status {
			if status.Status == "" || status.Status == v1.StepStatusFailed {
				// select the nodes whose execution fails
				if node := findStepNode(op.Steps[failedIndex].Nodes, status.Node); node.ID != "" {
					failedNodes = append(failedNodes, node)
				}
				continue
			}
			successStatus = append(successStatus, status)
		}
		// the step to continue
		continueSteps = op.Steps[failedIndex:]

		// the node that failed to execute the task
		continueSteps[0].Nodes = failedNodes
		// set the current step to the auto retry flag so that the step log file can be cleared later
		continueSteps[0].AutomaticRetry = true

		if len(successStatus) != 0 {
			// failed status
			failedStepStatus := op.Status.Conditions[failedIndex]
			// retain successful status
			failedStepStatus.Status = successStatus
			// Remove the failed status and keep the successful status
			op.Status.Conditions = op.Status.Conditions[0:failedIndex]
			op.Status.Conditions = append(op.Status.Conditions, failedStepStatus)
		} else {
			op.Status.Conditions = op.Status.Conditions[0:failedIndex]
		}
		if failedIndex > 0 && op.Status.Conditions[failedIndex-1].Status[0].Response != nil {
			ctx = component.WithExtraData(ctx, op.Status.Conditions[failedIndex-1].Status[0].Response)
		}
	}

	// if an uninstallation error occurs, start over
	if op.Steps[0].Action == v1.ActionUninstall {
		continueSteps = op.Steps
		op.Status.Conditions = make([]v1.OperationCondition, 0)
	}
	return ctx, op, continueSteps, nil
}

// SupportConcurrent whether concurrency is supported
func SupportConcurrent(opType string) bool {
	switch opType {
	case v1.OperationAddNodes, v1.OperationRemoveNodes:
		return true
	default:
		return false
	}
}

// Executable the operation is committed for execution
func Executable(ctx context.Context, opType, cluName string, operator operation.Operator) (bool, error) {
	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s,%s=%s", common.LabelClusterName, cluName, common.LabelOperationAction, opType)
	q.FieldSelector = fmt.Sprintf("status.status=%s", v1.OperationStatusRunning)
	q.Pagination.Offset = 0
	q.Pagination.Limit = 1
	// gets whether the same type of operations are being performed in the cluster
	resp, err := operator.ListOperationsEx(ctx, q)
	if err != nil {
		return false, err
	}
	if resp.TotalCount == 0 {
		return true, nil
	}
	return SupportConcurrent(opType), nil
}

// GetClusterPhase get the cluster phase based on the type of operation
func GetClusterPhase(opType string) v1.ClusterPhase {
	switch opType {
	case v1.OperationCreateCluster:
		return v1.ClusterInstalling
	case v1.OperationDeleteCluster:
		return v1.ClusterTerminating
	case v1.OperationUpgradeCluster:
		return v1.ClusterUpgrading
	case v1.OperationBackupCluster, v1.OperationDeleteBackup:
		return v1.ClusterBackingUp
	case v1.OperationRecoverCluster:
		return v1.ClusterRestoring
	default:
		// OperationAddNodes,OperationRemoveNodes
		// OperationInstallComponents,OperationUninstallComponents
		// OperationUpdateCertification
		return v1.ClusterUpdating
	}
}
