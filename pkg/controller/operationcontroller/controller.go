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

package operationcontroller

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/client-go/util/retry"

	"github.com/kubeclipper/kubeclipper/pkg/clusteroperation"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/service"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type OperationReconciler struct {
	CmdDelivery       service.CmdDelivery
	ClusterLister     listerv1.ClusterLister
	ClusterOperator   cluster.Operator
	OperationLister   listerv1.OperationLister
	OperationWriter   operation.Writer
	OperationOperator operation.Operator
}

func (r *OperationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	op, err := r.OperationLister.Get(req.Name)
	if err != nil {
		// operation not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get operation with name", zap.Error(err))
		return ctrl.Result{}, err
	}

	cluName := op.Labels[common.LabelClusterName]
	if cluName == "" {
		// TODO: throw a error here ?
		log.Warn("unexpected error, operation should always has a cluster name label",
			zap.String("operation", req.Name))
		return ctrl.Result{}, nil
	}

	if op.ObjectMeta.DeletionTimestamp.IsZero() {
		if !sets.NewString(op.ObjectMeta.Finalizers...).Has(v1.OperationFinalizer) {
			op.ObjectMeta.Finalizers = append(op.ObjectMeta.Finalizers, v1.OperationFinalizer)
			if op, err = r.OperationWriter.UpdateOperation(ctx, op); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if sets.NewString(op.ObjectMeta.Finalizers...).Has(v1.OperationFinalizer) {
			// The object is being deleted
			_, err := r.ClusterLister.Get(cluName)
			if err != nil && errors.IsNotFound(err) {
				// cluster has been deleted
				finalizers := sets.NewString(op.ObjectMeta.Finalizers...)
				finalizers.Delete(v1.OperationFinalizer)
				op.ObjectMeta.Finalizers = finalizers.List()
				_, err = r.OperationWriter.UpdateOperation(ctx, op)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, fmt.Errorf("cluster %s is being deleted", cluName)
		}
		return ctrl.Result{}, nil
	}

	_, err = r.ClusterLister.Get(cluName)
	if err != nil {
		if errors.IsNotFound(err) {
			// cluster has been deleted
			// delete operation which belongs to this cluster
			return ctrl.Result{}, r.OperationWriter.DeleteOperation(ctx, op.Name)
		}
		log.Error("Failed to get cluster with name", zap.String("cluster", cluName), zap.Error(err))
		return ctrl.Result{}, err
	}

	if err = r.distributeOperation(ctx, log, op); err != nil {
		log.Error("Failed to distribute operation", zap.String("cluster", cluName), zap.String("operation", op.Name), zap.Error(err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OperationReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("operation", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("operation-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Operation{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	mgr.AddRunnable(c)
	return nil
}

func (r *OperationReconciler) distributeOperation(ctx context.Context, log logger.Logging, op *v1.Operation) error {
	cluName := op.Labels[common.LabelClusterName]
	if op.Status.Status == v1.OperationStatusPending {
		op.Status.Status = v1.OperationStatusRunning
		if _, err := r.OperationWriter.UpdateOperation(ctx, op); err != nil {
			return err
		}

		go func() {
			if err := r.CmdDelivery.DeliverTaskOperation(context.TODO(), op, nil); err != nil {
				log.Error("distribute task error", zap.Error(err))
			}
		}()
	}
	if op.Status.Status == v1.OperationStatusFailed {
		times, _ := strconv.Atoi(op.Labels[common.LabelOperationRetry])
		op.Labels[common.LabelOperationRetry] = strconv.Itoa(times + 1)
		automaticRetry, err := clusteroperation.AutomaticRetry(ctx, op, r.OperationOperator)
		if err != nil {
			log.Error("automatic retry failure", zap.String("cluster", cluName), zap.String("operation", op.Name), zap.Error(err))
			return err
		}
		log.Infof("automatic retry result: %v, current-times: %s", automaticRetry, strconv.Itoa(times+1))
		// automatic retry
		if automaticRetry {
			// get retry operation step
			deliverCtx, newOp, continueSteps, err := clusteroperation.Retry(op)
			if err != nil {
				log.Error("automatic retry failure, processing retry step error", zap.String("cluster", cluName), zap.String("operation", op.Name), zap.Error(err))
				return err
			}
			// TODO: update operation and cluster is non-atomic
			if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// get the latest cluster info
				clu, err := r.ClusterOperator.GetClusterEx(ctx, cluName, "0")
				if err != nil {
					return err
				}
				// update cluster status
				clu.Status.Phase = clusteroperation.GetClusterPhase(op.Labels[common.LabelOperationAction])
				if _, err = r.ClusterOperator.UpdateCluster(ctx, clu.DeepCopy()); err != nil {
					return err
				}
				// update operation
				newOp.Status.Status = v1.OperationStatusRunning
				if _, err = r.OperationWriter.UpdateOperation(ctx, newOp); err != nil {
					return err
				}
				return nil
			}); err != nil {
				log.Error("automatic retry, update cluster and operation failed", zap.String("cluster", cluName), zap.String("operation", op.Name), zap.Error(err))
				return err
			}

			// process delivery steps
			newOp.Steps = continueSteps
			go func() {
				if err = r.CmdDelivery.DeliverTaskOperation(deliverCtx, newOp, nil); err != nil {
					log.Error("automatic retry, distribute task error", zap.String("cluster", cluName), zap.String("operation", op.Name), zap.Error(err))
				}
			}()
		}
	}
	return nil
}
