/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package source

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/predicate"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source/internal"
)

// Kind is used to provide a source of events originating inside the cluster from Watches (e.g. Pod Create).
type Kind struct {
	// Type is the type of object to watch.  e.g. &v1.Pod{}
	Type client.Object

	// cache used to watch APIs
	cache informers.InformerCache

	// started may contain an error if one was encountered during startup. If its closed and does not
	// contain an error, startup and syncing finished.
	started     chan error
	startCancel func()
}

func NewKindWithCache(object client.Object, cache informers.InformerCache) SyncingSource {
	return &Kind{Type: object, cache: cache}
}

var _ SyncingSource = &Kind{}

// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
// to enqueue reconcile.Requests.
func (ks *Kind) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {
	// Type should have been specified by the user.
	if ks.Type == nil {
		return fmt.Errorf("must specify Kind.Type")
	}

	// cache should have been injected before Start was called
	if ks.cache == nil {
		return fmt.Errorf("must call CacheInto on Kind before calling Start")
	}

	// cache.GetInformer will block until its context is cancelled if the cache was already started and it can not
	// sync that informer (most commonly due to RBAC issues).
	ctx, ks.startCancel = context.WithCancel(ctx)
	ks.started = make(chan error)
	go func() {
		var (
			i       cache.SharedIndexInformer
			lastErr error
		)

		// Tries to get an informer until it returns true,
		// an error or the specified context is cancelled or expired.
		if err := wait.PollImmediateUntilWithContext(ctx, 10*time.Second, func(ctx context.Context) (bool, error) {
			// Lookup the Informer from the Cache and add an EventHandler which populates the Queue
			i, lastErr = ks.cache.GetInformer(ctx, ks.Type)
			if lastErr != nil {
				switch {
				case runtime.IsMissingKind(lastErr):
					log.Error("informer must be init before use it", zap.Error(lastErr))
				default:
					log.Error("failed to get informer from cache", zap.Error(lastErr))
				}
				return false, nil // Retry.
			}
			return true, nil
		}); err != nil {
			if lastErr != nil {
				ks.started <- fmt.Errorf("failed to get informer from cache: %w", lastErr)
				return
			}
			ks.started <- err
			return
		}

		_, err := i.AddEventHandler(internal.EventHandler{Queue: queue, EventHandler: handler, Predicates: prct})
		if err != nil {
			return
		}
		if !ks.cache.WaitForCacheSync(ctx) {
			// Would be great to return something more informative here
			ks.started <- errors.New("cache did not sync")
		}
		close(ks.started)
	}()

	return nil
}

func (ks *Kind) String() string {
	if ks.Type != nil {
		return fmt.Sprintf("kind source: %T", ks.Type)
	}
	return "kind source: unknown type"
}

// WaitForSync implements SyncingSource to allow controllers to wait with starting
// workers until the cache is synced.
func (ks *Kind) WaitForSync(ctx context.Context) error {
	select {
	case err := <-ks.started:
		return err
	case <-ctx.Done():
		ks.startCancel()
		if ctx.Err() == context.Canceled {
			return nil
		}
		return errors.New("timed out waiting for cache to be synced")
	}
}
