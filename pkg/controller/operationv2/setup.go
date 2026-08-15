/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package operationv2

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	operationslister "github.com/kubeclipper/kubeclipper/pkg/client/lister/operations/v1alpha1"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func (r *OperationReconciler) SetupWithManager(mgr manager.Manager, factory informers.SharedInformerFactory) error {
	operationInformer := factory.Operations().V1alpha1().Operations()
	taskInformer := factory.Operations().V1alpha1().OperationTasks()
	// Informers must be materialized before Kind sources ask the shared cache
	// for them during controller startup.
	operationInformer.Informer()
	taskInformer.Informer()

	c, err := controller.NewUnmanaged("operation-v2", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("operation-v2-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err := c.Watch(
		source.NewKindWithCache(&operations.Operation{}, factory),
		handler.EnqueueRequestsFromMapFunc(mapTargetOperations(operationInformer.Lister())),
	); err != nil {
		return err
	}
	if err := c.Watch(
		source.NewKindWithCache(&operations.OperationTask{}, factory),
		handler.EnqueueRequestsFromMapFunc(mapTaskOperation),
	); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}

func mapTargetOperations(lister operationslister.OperationLister) handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		op, ok := object.(*operations.Operation)
		if !ok {
			return nil
		}
		result := []reconcile.Request{{NamespacedName: types.NamespacedName{Name: op.Name}}}
		all, err := lister.List(labels.Everything())
		if err != nil {
			return result
		}
		seen := map[string]struct{}{op.Name: {}}
		for _, candidate := range all {
			if candidate.Spec.TargetRef.UID != op.Spec.TargetRef.UID {
				continue
			}
			if _, exists := seen[candidate.Name]; exists {
				continue
			}
			seen[candidate.Name] = struct{}{}
			result = append(result, reconcile.Request{NamespacedName: types.NamespacedName{Name: candidate.Name}})
		}
		return result
	}
}

func mapTaskOperation(object client.Object) []reconcile.Request {
	task, ok := object.(*operations.OperationTask)
	if !ok || task.Spec.OperationRef.Name == "" {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: task.Spec.OperationRef.Name}}}
}
