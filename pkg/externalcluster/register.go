package externalcluster

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

func GetClusterProvider(providerName string) (ProviderFactory, error) {
	v, ok := providerFactories[providerName]
	if !ok {
		return nil, fmt.Errorf("this cluster provider(%s) is not support", providerName)
	}
	return v, nil
}

type ProviderFactory interface {
	ClusterType() string
	CreateClusterManage(cm *v1.ConfigMap) (ClusterManage, error)
}

type ClusterManage interface {
	ClusterInfo() (cluster *v1.Cluster, err error)
}
