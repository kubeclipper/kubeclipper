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

package lease

import (
	"context"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/models"

	"k8s.io/apimachinery/pkg/watch"

	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
)

var _ Operator = (*leaseOperator)(nil)

type leaseOperator struct {
	storage rest.StandardStorage
}

func NewLeaseOperator(leaseStorage rest.StandardStorage) Operator {
	return &leaseOperator{
		storage: leaseStorage,
	}
}

func (l *leaseOperator) ListLeases(ctx context.Context, query *query.Query) (*coordinationv1.LeaseList, error) {
	list, err := models.List(ctx, l.storage, query)
	if err != nil {
		return nil, err
	}
	list.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("LeaseList"))
	result, _ := list.(*coordinationv1.LeaseList)
	return result, nil
}

func (l *leaseOperator) GetLease(ctx context.Context, name string, resourceVersion string) (*coordinationv1.Lease, error) {
	obj, err := models.Get(ctx, l.storage, name, resourceVersion)
	if err != nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), nil
}

func (l *leaseOperator) WatchLease(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, l.storage, query)
}

func (l *leaseOperator) GetLeaseWithNamespaceEx(ctx context.Context, name string, namespace, resourceVersion string) (*coordinationv1.Lease, error) {
	ctx = genericapirequest.WithNamespace(ctx, namespace)
	op, err := models.Get(ctx, l.storage, name, resourceVersion)
	if err != nil {
		return nil, err
	}
	return op.(*coordinationv1.Lease), nil
}

func (l *leaseOperator) ListLeasesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	panic("implement me")
}

func (l *leaseOperator) GetLeaseWithNamespace(ctx context.Context, name string, namespace string) (*coordinationv1.Lease, error) {
	ctx = genericapirequest.WithNamespace(ctx, namespace)
	obj, err := models.Get(ctx, l.storage, name, "")
	if err != nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), nil
}

func (l *leaseOperator) CreateLease(ctx context.Context, lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	ctx = genericapirequest.WithNamespace(ctx, lease.Namespace)
	obj, err := l.storage.Create(ctx, lease, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), nil
}

func (l *leaseOperator) UpdateLease(ctx context.Context, lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	ctx = genericapirequest.WithNamespace(ctx, lease.Namespace)
	obj, wasCreated, err := l.storage.Update(ctx, lease.Name, rest.DefaultUpdatedObjectInfo(lease), nil, nil, true, &metav1.UpdateOptions{})
	if wasCreated {
		logger.Debug("lease not exist, use create instead of update", zap.String("lease_name", lease.Name), zap.String("lease_ns", lease.Namespace))
	}
	if err != nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), nil
}
