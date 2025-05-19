package clusteroperation

import (
	"context"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"
)

func GetCriStep(ctx context.Context, c *v1.Cluster, cluOp cluster.Operator, action v1.StepAction, nodes []v1.StepNode) ([]v1.Step, error) {
	switch c.ContainerRuntime.Type {
	case v1.CRIDocker:
		r := cri.DockerRunnable{}
		err := r.InitStep(ctx, c, nodes)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	case v1.CRIContainerd:
		registries, err := utils.GetClusterCRIRegistries(c, cluOp)
		if err != nil {
			return nil, err
		}
		r := cri.ContainerdRunnable{}
		err = r.InitStep(ctx, c, nodes, registries)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	}
	return nil, fmt.Errorf("%v type CRI is not supported", c.ContainerRuntime.Type)
}
