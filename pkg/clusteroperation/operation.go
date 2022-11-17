package clusteroperation

import (
	"github.com/kubeclipper/kubeclipper/pkg/component"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Options struct {
	cluster          *v1.Cluster
	pendingOperation v1.PendingOperation
	extra            *component.ExtraMetadata
}

type Interface interface {
	Builder(options Options) (*v1.Operation, error)
}

func BuildOperationAdapter(cluster *v1.Cluster, pendingOp v1.PendingOperation, extra *component.ExtraMetadata) (*v1.Operation, error) {
	var instance Interface
	switch pendingOp.OperationType {
	case v1.OperationAddNodes, v1.OperationRemoveNodes:
		instance = &NodeOperation{}
		return instance.Builder(Options{
			cluster:          cluster,
			pendingOperation: pendingOp,
			extra:            extra,
		})
	}

	return nil, nil
}
