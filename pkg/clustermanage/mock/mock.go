package mock

import (
	"context"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type Provider struct {
	Provider *v1.CloudProvider
}

func NewProvider(provider *v1.CloudProvider) clustermanage.CloudProvider {
	return Provider{
		Provider: provider,
	}
}

func (p Provider) Sync(ctx context.Context) error {
	logger.Info("mock provider sync")
	return nil
}

func (p Provider) Cleanup(ctx context.Context) error {
	logger.Info("mock provider cleanup")
	return nil
}

func (p Provider) PreCheck(ctx context.Context) (bool, error) {
	logger.Info("mock provider preCheck")
	return true, nil
}
func (p Provider) GetKubeConfig(ctx context.Context, clusterName string) (string, error) {
	logger.Info("mock provider getKubeConfig")
	return "", nil
}

func (p Provider) GetCertification(ctx context.Context, clusterName string) ([]v1.Certification, error) {
	logger.Info("mock provider getCertification")
	return nil, nil
}
