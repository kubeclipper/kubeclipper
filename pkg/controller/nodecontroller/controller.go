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

package nodecontroller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

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

type NodeReconciler struct {
	NodeLister    listerv1.NodeLister
	ClusterLister listerv1.ClusterLister

	NodeWriter cluster.NodeWriter
}

func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	log.Info("in node reconcile")

	node, err := r.NodeLister.Get(req.Name)
	if err != nil {
		// node not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get node with name", zap.String("node", req.Name), zap.Error(err))
		return ctrl.Result{}, err
	}

	if !node.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		return ctrl.Result{}, nil
	}
	if err = r.syncNodeRole(ctx, node); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *NodeReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("node", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("node-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Node{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	mgr.AddRunnable(c)
	return nil
}

func (r *NodeReconciler) syncNodeRole(ctx context.Context, node *v1.Node) error {
	cluName, ok := node.Labels[common.LabelClusterName]
	if !ok {
		return nil
	}
	clu, err := r.ClusterLister.Get(cluName)
	if err != nil {
		if errors.IsNotFound(err) {
			delete(node.Labels, common.LabelClusterName)
			_, err = r.NodeWriter.UpdateNode(context.TODO(), node)
			return err
		}
		return err
	}
	if err = r.updateNodeRoleIfNotEqual(ctx, clu, node, common.NodeRoleMaster); err != nil {
		return err
	}
	return r.updateNodeRoleIfNotEqual(ctx, clu, node, common.NodeRoleWorker)
}

func (r *NodeReconciler) updateNodeRoleIfNotEqual(ctx context.Context, clu *v1.Cluster, node *v1.Node, nodeRole common.NodeRole) error {
	var (
		err   error
		nodes sets.Set[string]
	)
	switch nodeRole {
	case common.NodeRoleMaster:
		nodes = sets.New(clu.Masters.GetNodeIDs()...)
	case common.NodeRoleWorker:
		nodes = sets.New(clu.Workers.GetNodeIDs()...)
	default:
		return fmt.Errorf("unsupported ")
	}
	if nodes.Has(node.Name) {
		if node.Labels[common.LabelNodeRole] != string(nodeRole) {
			node.Labels[common.LabelNodeRole] = string(nodeRole)
			_, err = r.NodeWriter.UpdateNode(ctx, node)
		}
	}
	return err
}
