/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package regioncontroller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type RegionReconciler struct {
	NodeLister   listerv1.NodeLister
	RegionLister listerv1.RegionLister

	RegionWriter cluster.RegionWriter
}

// Reconcile implements controller.Reconciler.
// create region when node is created,delete region when all node in region is deleted.
func (r *RegionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)
	log.Info("in region reconcile")

	_, err := r.RegionLister.Get(req.Name)
	if err != nil {
		// region not exist,create it.
		if errors.IsNotFound(err) {
			if err = r.createRegion(ctx, req.Name); err != nil {
				log.Error("Failed to create region", zap.Error(err))
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get region with name", zap.String("region", req.Name), zap.Error(err))
		return ctrl.Result{}, err
	}

	if r.needDelete(req.Name) {
		// delete region if need.
		if err = r.RegionWriter.DeleteRegion(ctx, req.Name); err != nil {
			log.Error("Failed to check region delete", zap.String("region", req.Name), zap.Error(err))
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *RegionReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("region", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("region-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Node{}, cache), handler.EnqueueRequestsFromMapFunc(r.findObjectsForRegion)); err != nil {
		return err
	}

	mgr.AddRunnable(c)
	return nil
}

func (r *RegionReconciler) createRegion(ctx context.Context, region string) error {
	_, err := r.RegionWriter.CreateRegion(ctx, &v1.Region{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Region",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: region,
		},
	})
	return err
}

func (r *RegionReconciler) findObjectsForRegion(node client.Object) []reconcile.Request {
	// node deleted,need check is need delete region.
	if node.GetDeletionTimestamp() != nil || !node.GetDeletionTimestamp().IsZero() {
		return r.needDeleteRegions()
	}

	// node created/updated,need check is need create region.
	region, ok := node.GetLabels()[common.LabelTopologyRegion]
	if !ok {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: region,
		},
	}}
}

func (r *RegionReconciler) needDeleteRegions() []reconcile.Request {
	var requests []reconcile.Request

	regions, err := r.RegionLister.List(labels.Everything())
	if err != nil {
		return requests
	}

	for _, region := range regions {
		if r.needDelete(region.Name) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: region.Name,
				},
			})
		}
	}
	return requests
}

func (r *RegionReconciler) needDelete(region string) bool {
	requirement, err := labels.NewRequirement(common.LabelTopologyRegion, selection.Equals, []string{region})
	if err != nil {
		return false
	}
	list, err := r.NodeLister.List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return errors.IsNotFound(err)
	}
	return len(list) == 0
}
