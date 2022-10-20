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

package loginrecordcontroller

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	iamListerV1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/iam/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type LoginRecordReconciler struct {
	UserLister            iamListerV1.UserLister
	UserWriter            iam.UserWriter
	LoginRecordLister     iamListerV1.LoginRecordLister
	LoginRecordWriter     iam.LoginRecordWriter
	AuthenticationOptions *options.AuthenticationOptions
}

func (r *LoginRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)
	log.Info("in loginrecord reconcile")

	loginRecord, err := r.LoginRecordLister.Get(req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get login record with name", zap.String("user", req.Name), zap.Error(err))
		return ctrl.Result{}, err
	}

	if !loginRecord.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		// Our finalizer has finished, so the reconciler can do nothing.
		return ctrl.Result{}, nil
	}

	user, err := r.userForLoginRecord(loginRecord)
	if err != nil {
		// delete orphan object
		if errors.IsNotFound(err) {
			return ctrl.Result{}, r.LoginRecordWriter.DeleteLoginRecord(context.TODO(), loginRecord.Name)
		}
		return ctrl.Result{}, err
	}

	if err = r.updateUserLastLoginTime(user, loginRecord); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// updateUserLastLoginTime accepts a login object and set user lastLoginTime field
func (r *LoginRecordReconciler) updateUserLastLoginTime(user *iamv1.User, loginRecord *iamv1.LoginRecord) error {
	// update lastLoginTime
	if user.DeletionTimestamp.IsZero() &&
		(user.Status.LastLoginTime == nil || user.Status.LastLoginTime.Before(&loginRecord.CreationTimestamp)) {
		user.Status.LastLoginTime = &loginRecord.CreationTimestamp
		_, err := r.UserWriter.UpdateUser(context.Background(), user)
		return err
	}
	return nil
}

func (r *LoginRecordReconciler) userForLoginRecord(loginRecord *iamv1.LoginRecord) (*iamv1.User, error) {
	username, ok := loginRecord.Labels[common.LabelUserReference]
	if !ok || len(username) == 0 {
		return nil, errors.NewNotFound(iamv1.Resource("user"), username)
	}
	return r.UserLister.Get(username)
}

func (r *LoginRecordReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("loginrecord", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("loginrecord-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&iamv1.LoginRecord{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}
