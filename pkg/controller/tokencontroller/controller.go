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

package tokencontroller

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	iamListerV1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/iam/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type TokenReconciler struct {
	TokenLister iamListerV1.TokenLister
	TokenWriter iam.TokenWriter
}

func (t *TokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)
	log.Debug("in token reconcile")
	token, err := t.TokenLister.Get(req.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get token", zap.Error(err))
		return ctrl.Result{}, err
	}
	token = token.DeepCopy()
	if !token.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, err
	}
	if token.Spec.TTL == nil || *token.Spec.TTL == 0 {
		log.Debug("token has no ttl")
		return ctrl.Result{}, nil
	}
	ttl := time.Duration(*token.Spec.TTL) * time.Second
	expireAt := metav1.NewTime(token.CreationTimestamp.Add(ttl))
	if token.Status.ExpiresAt == nil || token.Status.ExpiresAt.IsZero() || !token.Status.ExpiresAt.Equal(&expireAt) {
		token.Status.ExpiresAt = &expireAt
		log.Debug("token expires at is zeor, update it", zap.Time("createAt", token.CreationTimestamp.Time), zap.Time("expireAt", expireAt.Time))
		_, err = t.TokenWriter.UpdateToken(ctx, token)
		return ctrl.Result{}, err
	}
	now := metav1.Now()
	if now.After(token.Status.ExpiresAt.Time) {
		log.Debug("token expires, delete it")
		err = t.TokenWriter.DeleteToken(ctx, token.Name)
		return ctrl.Result{}, err
	}
	waitToDelete := token.Status.ExpiresAt.Sub(now.Time)
	log.Debug("token requeue later", zap.Duration("requeueAt", waitToDelete))
	return ctrl.Result{
		RequeueAfter: waitToDelete,
	}, nil
}

func (t *TokenReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("token", controller.Options{
		MaxConcurrentReconciles: 1,
		Reconciler:              t,
		Log:                     mgr.GetLogger().WithName("token-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&iamv1.Token{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}
