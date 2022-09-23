package clustermanage

import (
	"context"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

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
