package clusteroperation

import (
	"context"
	"fmt"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"
)

func getCriStep(ctx context.Context, c *v1.Cluster, action v1.StepAction, nodes []v1.StepNode) ([]v1.Step, error) {
	switch c.ContainerRuntime.Type {
	case v1.CRIDocker:
		r := cri.DockerRunnable{}
		err := r.InitStep(ctx, c, nodes)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	case v1.CRIContainerd:
		r := cri.ContainerdRunnable{}
		err := r.InitStep(ctx, c, nodes)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	}
	return nil, fmt.Errorf("%v type CRI is not supported", c.ContainerRuntime.Type)
}
