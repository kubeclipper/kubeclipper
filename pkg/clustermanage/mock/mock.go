package mock

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type Provider struct {
	log logger.Logging
}

func NewProvider() clustermanage.CloudProvider {
	return Provider{}
}

func (Provider) Sync(ctx context.Context) error {
	log.Info("mock provider sync")
	return nil
}

func (Provider) Cleanup(ctx context.Context) error {
	log.Info("mock provider cleanup")
	return nil
}
