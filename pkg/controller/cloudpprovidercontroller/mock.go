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

package cloudpprovidercontroller

import (
	"context"
	"reflect"
	"time"

	pkgerrors "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage/mock"

	"github.com/kubeclipper/kubeclipper/pkg/controller/utils"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const cloudProviderInterval = time.Hour * 4

type Reconciler struct {
	ClusterLister listerv1.ClusterLister
	ClusterWriter cluster.ClusterWriter

	CloudProviderLister listerv1.CloudProviderLister
	CloudProviderWriter cluster.CloudProviderWriter

	NodeLister listerv1.NodeLister
	NodeWriter cluster.NodeWriter

	ConfigmapLister listerv1.ConfigMapLister
	ConfigmapWriter core.ConfigMapWriter
}

func (r *Reconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("cloudprovider-rancher", controller.Options{
		MaxConcurrentReconciles: 1, // must run serialize
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("cloudprovider-rancher-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}

	if err = c.Watch(source.NewKindWithCache(&v1.CloudProvider{}, cache), handler.EnqueueRequestsFromMapFunc(r.findObjectsForCloudProvider)); err != nil {
		return err
	}

	mgr.AddRunnable(c)
	return nil
}

func (r *Reconciler) findObjectsForCloudProvider(objProvider client.Object) []reconcile.Request {
	provider, err := r.CloudProviderLister.Get(objProvider.GetName())
	if err != nil {
		return []reconcile.Request{}
	}
	// NOTE: per controller just watch one kind of provider
	if provider.Type != "mock" {
		return []reconcile.Request{}
	}
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name: provider.Name,
		}},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)
	defer utils.Trace(log, "rancher provider reconcile")

	provider, err := r.CloudProviderLister.Get(req.Name)
	if err != nil {
		// cloud provider not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("failed to get cloud provider", zap.Error(err))
		return ctrl.Result{}, err
	}

	mockProvider := mock.NewProvider(provider)

	if provider.ObjectMeta.DeletionTimestamp.IsZero() {
		if GetCondition(provider.Status, v1.CloudProviderProgressing) == nil {
			condition := NewCondition(v1.CloudProviderProgressing, v1.ConditionTrue, v1.CloudProviderCreated, "provider created success")
			SetCondition(&provider.Status, *condition)
		}
		// insert finalizers
		if !sets.NewString(provider.ObjectMeta.Finalizers...).Has(v1.CloudProviderFinalizer) {
			provider.ObjectMeta.Finalizers = append(provider.ObjectMeta.Finalizers, v1.CloudProviderFinalizer)
		}
		provider, err = r.CloudProviderWriter.UpdateCloudProvider(context.TODO(), provider)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted
		if sets.NewString(provider.ObjectMeta.Finalizers...).Has(v1.CloudProviderFinalizer) {
			// when cloudProvider deleted,we need delete this provider's clusters from kc,and clean kc-agent on all nodes.
			if err = mockProvider.Cleanup(ctx); err != nil {
				condition := NewCondition(v1.CloudProviderReady, v1.ConditionFalse, v1.CloudProviderTerminateFailed, convert(err))
				log.Error("failed to cleanup", zap.Error(err))
				newProvider := provider.DeepCopy()
				SetCondition(&newProvider.Status, *condition)
				if reflect.DeepEqual(provider, newProvider) {
					return ctrl.Result{}, err
				}
				if _, err = r.CloudProviderWriter.UpdateCloudProvider(ctx, newProvider); err != nil {
					return ctrl.Result{}, pkgerrors.WithMessage(err, "record error msg")
				}
				// ignore error,if update
				return ctrl.Result{}, nil
			}

			// when cleanup remove our cloudProvider finalizer
			finalizers := sets.NewString(provider.ObjectMeta.Finalizers...)
			finalizers.Delete(v1.CloudProviderFinalizer)
			provider.ObjectMeta.Finalizers = finalizers.List()
			if _, err = r.CloudProviderWriter.UpdateCloudProvider(ctx, provider); err != nil {
				log.Error("failed to delete finalizer", zap.Error(err))
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 	if provider created or updated,we need sync data.
	if err = mockProvider.Sync(ctx); err != nil {
		log.Error("failed to sync", zap.Error(err))
		newProvider := provider.DeepCopy()
		condition := NewCondition(v1.CloudProviderReady, v1.ConditionFalse, v1.CloudProviderSyncFailed, convert(err))
		SetCondition(&newProvider.Status, *condition)
		if reflect.DeepEqual(provider, newProvider) {
			return ctrl.Result{}, err
		}
		if _, err = r.CloudProviderWriter.UpdateCloudProvider(ctx, newProvider); err != nil {
			return ctrl.Result{}, pkgerrors.WithMessage(err, "record error msg")
		}
		// ignore error,if update
		return ctrl.Result{}, nil
	}

	newProvider := provider.DeepCopy()
	condition := NewCondition(v1.CloudProviderReady, v1.ConditionTrue, v1.CloudProviderSyncSucceed, "sync successful")
	SetCondition(&newProvider.Status, *condition)
	if !reflect.DeepEqual(provider, newProvider) {
		if _, err = r.CloudProviderWriter.UpdateCloudProvider(ctx, newProvider); err != nil {
			return ctrl.Result{}, pkgerrors.WithMessage(err, "update status to successful")
		}
	}
	return ctrl.Result{RequeueAfter: cloudProviderInterval}, nil
}

// convert err to a human-readable msg
func convert(err error) string {
	return err.Error()
}

// NewCondition creates a new deployment condition.
func NewCondition(condType v1.CloudProviderConditionType, status v1.ConditionStatus, reason, message string) *v1.CloudProviderCondition {
	return &v1.CloudProviderCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetCondition returns the condition with the provided type.
func GetCondition(status v1.CloudProviderStatus, condType v1.CloudProviderConditionType) *v1.CloudProviderCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetCondition updates the deployment to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetCondition(status *v1.CloudProviderStatus, condition v1.CloudProviderCondition) {
	currentCond := GetCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// RemoveCondition removes the deployment condition with the provided type.
func RemoveCondition(status *v1.CloudProviderStatus, condType v1.CloudProviderConditionType) {
	status.Conditions = filterOutCondition(status.Conditions, condType)
}

// filterOutCondition returns a new slice of deployment conditions without conditions with the provided type.
func filterOutCondition(conditions []v1.CloudProviderCondition, condType v1.CloudProviderConditionType) []v1.CloudProviderCondition {
	var newConditions []v1.CloudProviderCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
