package projectcontroller

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

import (
	"context"

	"github.com/kubeclipper/kubeclipper/pkg/models/tenant"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1lister "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	tenantv1Lister "github.com/kubeclipper/kubeclipper/pkg/client/lister/tenant/v1"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type ProjectReconciler struct {
	ProjectLister tenantv1Lister.ProjectLister
	ProjectWriter tenant.ProjectWriter

	NodeLister corev1lister.NodeLister
	NodeWriter cluster.NodeWriter
	// TODO projectRole & projectRoleBinding
}

func (r *ProjectReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("project", controller.Options{
		MaxConcurrentReconciles: 1, // must run serialize
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("project-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}

	if err = c.Watch(source.NewKindWithCache(&tenantv1.Project{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	mgr.AddRunnable(c)
	return nil
}

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	project, err := r.ProjectLister.Get(req.Name)
	if err != nil {
		// project not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error("failed to get project", zap.Error(err))
		return ctrl.Result{}, err
	}

	if project.ObjectMeta.DeletionTimestamp.IsZero() {
		// insert finalizers
		if !sets.NewString(project.ObjectMeta.Finalizers...).Has(v1.ProjectFinalizer) {
			project.ObjectMeta.Finalizers = append(project.ObjectMeta.Finalizers, v1.ProjectFinalizer)
		}
		project, err = r.ProjectWriter.UpdateProject(context.TODO(), project)
		if err != nil {
			log.Error("update project,add finalizer", zap.Error(err))
			return ctrl.Result{}, err
		}
	} else {
		// The object is being deleted
		if sets.NewString(project.ObjectMeta.Finalizers...).Has(v1.ProjectFinalizer) {
			// delete projectRole and RoleBinding
			if err = r.deleteRoleAndRoleBinding(ctx, project); err != nil {
				log.Error("failed to delete role roleBinding", zap.Error(err))
				return ctrl.Result{}, err
			}

			// when cleanup remove our project finalizer
			finalizers := sets.NewString(project.ObjectMeta.Finalizers...)
			finalizers.Delete(v1.ProjectFinalizer)
			project.ObjectMeta.Finalizers = finalizers.List()
			if _, err = r.ProjectWriter.UpdateProject(ctx, project); err != nil {
				log.Error("failed to delete finalizer", zap.Error(err))
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err = r.initProjectRole(ctx, project); err != nil {
		log.Error("failed to init project role", zap.Error(err))
		return ctrl.Result{}, err
	}

	if err = r.initManagerRoleBinding(ctx, project); err != nil {
		log.Error("failed to init manager role binding", zap.Error(err))
		return ctrl.Result{}, err
	}

	if err = r.syncNodeLabel(ctx, project); err != nil {
		log.Error("failed to sync node label", zap.Error(err))
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) initProjectRole(ctx context.Context, p *tenantv1.Project) error {
	log := logger.FromContext(ctx)
	log.Info("initProjectRole")
	// TODO：initProjectRole
	// check if all base role exist,if not creat it

	return nil
}

func (r *ProjectReconciler) initManagerRoleBinding(ctx context.Context, p *tenantv1.Project) error {
	log := logger.FromContext(ctx)
	log.Info("initManagerRoleBinding")

	// TODO：bind admin role to project manager
	// check if projectRoleBinding for project.manager exist,if not creat it
	// ManagerRoleBinding use a fixed name($projectName-admin),if project.manager changed,we just update it.

	return nil
}

func (r *ProjectReconciler) deleteRoleAndRoleBinding(ctx context.Context, p *tenantv1.Project) error {
	log := logger.FromContext(ctx)
	log.Info("deleteRoleAndRoleBinding")

	// TODO: delete projectRole and RoleBinding
	// delete projectRole and projectRoleBinding which in this project
	return nil
}

func (r *ProjectReconciler) syncNodeLabel(ctx context.Context, p *tenantv1.Project) error {
	// list nodes in this project
	requirement, err := labels.NewRequirement(common.LabelProject, selection.Equals, []string{p.Name})
	if err != nil {
		return err
	}
	list, err := r.NodeLister.List(labels.NewSelector().Add(*requirement))
	if err != nil {
		return err
	}
	// remove label when node leave project
	for _, node := range list {
		set := sets.NewString(p.Spec.Nodes...)
		if !set.Has(node.Name) {
			delete(node.Labels, common.LabelProject)
			if _, err = r.NodeWriter.UpdateNode(ctx, node); err != nil {
				return pkgerrors.WithMessagef(err, "delete node %s's label", node.Name)
			}
		}
	}
	// add label when node join project
	for _, nodeID := range p.Spec.Nodes {
		node, err := r.NodeLister.Get(nodeID)
		if err != nil {
			return err
		}
		if node.Labels[common.LabelProject] != p.Name {
			node.Labels[common.LabelProject] = p.Name
			if _, err = r.NodeWriter.UpdateNode(ctx, node); err != nil {
				return err
			}
		}
	}
	return nil
}
