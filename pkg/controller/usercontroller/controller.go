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

package usercontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

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

type UserReconciler struct {
	UserLister            iamListerV1.UserLister
	LoginRecordLister     iamListerV1.LoginRecordLister
	UserWriter            iam.UserWriter
	AuthenticationOptions *options.AuthenticationOptions
}

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !r.AuthenticationOptions.AuthenticateRateLimiterEnable {
		return ctrl.Result{}, nil
	}
	log := logger.FromContext(ctx)
	log.Info("in user reconcile")

	user, err := r.UserLister.Get(req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get user with name", zap.String("user", req.Name), zap.Error(err))
		return ctrl.Result{}, err
	}
	// blocked user, check if need to unblock user
	if user.Status.State != nil && *user.Status.State == iamv1.UserAuthLimitExceeded {
		if user.Status.LastTransitionTime != nil &&
			user.Status.LastTransitionTime.Add(r.AuthenticationOptions.AuthenticateRateLimiterDuration).Before(time.Now()) {
			// unblock user
			active := iamv1.UserActive
			user.Status = iamv1.UserStatus{
				State:              &active,
				LastTransitionTime: &metav1.Time{Time: time.Now()},
			}
			_, err := r.UserWriter.UpdateUser(ctx, user)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	// normal user, check user's login records see if we need to block
	records, err := r.LoginRecordLister.List(labels.SelectorFromSet(labels.Set{common.LabelUserReference: user.Name}))
	if err != nil {
		log.Error("Failed to get login record list", zap.Error(err))
		return ctrl.Result{}, err
	}

	// count failed login attempts during last AuthenticateRateLimiterDuration
	now := time.Now()
	failedLoginAttempts := 0
	for _, loginRecord := range records {
		if !loginRecord.Spec.Success &&
			loginRecord.CreationTimestamp.Add(r.AuthenticationOptions.AuthenticateRateLimiterDuration).After(now) {
			failedLoginAttempts++
		}
	}

	// block user if failed login attempts exceeds maximum tries setting
	if failedLoginAttempts >= r.AuthenticationOptions.AuthenticateRateLimiterMaxTries {
		limitExceed := iamv1.UserAuthLimitExceeded
		user.Status = iamv1.UserStatus{
			State:              &limitExceed,
			Reason:             fmt.Sprintf("Failed login attempts exceed %d in last %s", failedLoginAttempts, r.AuthenticationOptions.AuthenticateRateLimiterDuration),
			LastTransitionTime: &metav1.Time{Time: time.Now()},
		}

		_, err = r.UserWriter.UpdateUser(ctx, user)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	// block user for AuthenticateRateLimiterDuration duration, after that put it back to the queue to unblock
	if user.Status.State != nil && *user.Status.State == iamv1.UserAuthLimitExceeded {
		return ctrl.Result{Requeue: true, RequeueAfter: r.AuthenticationOptions.AuthenticateRateLimiterDuration}, nil
	}
	return ctrl.Result{}, nil
}

func (r *UserReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("user", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("user-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}

	if err = c.Watch(source.NewKindWithCache(&iamv1.User{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}
