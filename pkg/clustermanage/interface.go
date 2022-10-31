package clustermanage

import (
	"context"
	"fmt"

	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var providerFactories = make(map[string]ProviderFactory)

func RegisterProvider(factory ProviderFactory) {
	providerFactories[factory.ClusterType()] = factory
}

func GetProvider(operator Operator, cp v1.CloudProvider) (CloudProvider, error) {
	v, ok := providerFactories[cp.Type]
	if !ok {
		return nil, fmt.Errorf("this cluster provider(%s) is not support", cp.Type)
	}
	c, err := v.InitCloudProvider(operator, cp)
	if err != nil {
		logger.Errorf("init cloud provider(%s) failed: %v", cp.Name, err)
	}
	return c, err
}

type ProviderFactory interface {
	ClusterType() string
	InitCloudProvider(operator Operator, provider v1.CloudProvider) (CloudProvider, error)
}

type Operator struct {
	ClusterReader cluster.ClusterReader
	ClusterLister listerv1.ClusterLister
	ClusterWriter cluster.ClusterWriter

	CloudProviderLister listerv1.CloudProviderLister
	CloudProviderWriter cluster.CloudProviderWriter
	CloudProviderReader cluster.CloudProviderReader

	NodeLister listerv1.NodeLister
	NodeWriter cluster.NodeWriter

	ConfigmapLister listerv1.ConfigMapLister
	ConfigmapWriter core.ConfigMapWriter
}

type CloudProvider interface {
	Sync(ctx context.Context) error
	Cleanup(tx context.Context) error
	Expansion
}

type Expansion interface {
	PreCheck(ctx context.Context) (bool, error)
	GetKubeConfig(ctx context.Context, clusterName string) (string, error)
	GetCertification(ctx context.Context, clusterName string) ([]v1.Certification, error)
}
