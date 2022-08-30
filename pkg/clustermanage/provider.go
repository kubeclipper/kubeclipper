package clustermanage

import (
	"fmt"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	ClusterKubeadm string = "kubeadm"
	// TODO: other cluster type

	FedTypeAgent = "agent"
	FedTypePod   = "pod"
)

var providerFactories = make(map[string]ProviderFactory)

func registerProvider(factory ProviderFactory) {
	providerFactories[factory.ClusterType()] = factory
}

func GetClusterProvider(name string) (ProviderFactory, error) {
	v, ok := providerFactories[name]
	if !ok {
		return nil, fmt.Errorf("this cluster type(%s) is not registered", name)
	}
	return v, nil
}

type ProviderFactory interface {
	ClusterType() string
	CreateClusterManage(spec v1.ProviderSpec) ClusterManage
}

type ClusterManage interface {
	ClusterInfo(name string, provider v1.ProviderSpec) (cluster *v1.Cluster, nodeMap map[string]string, err error)
}
