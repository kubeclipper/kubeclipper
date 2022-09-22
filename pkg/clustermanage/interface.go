package clustermanage

import "context"

type CloudProvider interface {
	Sync(ctx context.Context) error
	Cleanup(tx context.Context) error
}
