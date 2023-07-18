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

package operation

import (
	"context"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubeclipper/kubeclipper/pkg/models"

	"k8s.io/apimachinery/pkg/watch"

	"go.uber.org/zap"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var _ Operator = (*operationOperator)(nil)

type operationOperator struct {
	storage rest.StandardStorage
}

func NewOperationOperator(operationStorage rest.StandardStorage) Operator {
	return &operationOperator{
		storage: operationStorage,
	}
}

func (l *operationOperator) ListOperations(ctx context.Context, query *query.Query) (*v1.OperationList, error) {
	list, err := models.List(ctx, l.storage, query)
	if err != nil {
		return nil, err
	}
	list.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("OperationList"))
	return list.(*v1.OperationList), nil
}

func (l *operationOperator) GetOperationEx(ctx context.Context, name string, resourceVersion string) (*v1.Operation, error) {
	op, err := models.GetV2(ctx, l.storage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return op.(*v1.Operation), nil
}

func (l *operationOperator) ListOperationsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, l.storage, query, l.operationFilter, nil, nil)
}

func (l *operationOperator) GetOperation(ctx context.Context, name string) (*v1.Operation, error) {
	return l.GetOperationEx(ctx, name, "")
}

func (l *operationOperator) CreateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	ctx = genericapirequest.WithNamespace(ctx, operation.Namespace)
	obj, err := l.storage.Create(ctx, operation, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Operation), nil
}

func (l *operationOperator) UpdateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	ctx = genericapirequest.WithNamespace(ctx, operation.Namespace)
	obj, wasCreated, err := l.storage.Update(ctx, operation.Name, rest.DefaultUpdatedObjectInfo(operation), nil, nil, false, &metav1.UpdateOptions{})
	if wasCreated {
		logger.Debug("operation not exist, use create instead of update", zap.String("name", operation.Name))
	}
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Operation), nil
}

func (l *operationOperator) UpdateOperationStatus(ctx context.Context, name string, status *v1.OperationStatus) (*v1.Operation, error) {
	op, err := l.GetOperation(ctx, name)
	if err != nil {
		return nil, err
	}
	op.Status = *status
	return l.UpdateOperation(ctx, op)
}

func (l *operationOperator) WatchOperations(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, l.storage, query)
}

func (l *operationOperator) DeleteOperation(ctx context.Context, name string) error {
	var err error
	_, _, err = l.storage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (l *operationOperator) DeleteOperationCollection(ctx context.Context, query *query.Query) error {
	if _, err := l.storage.DeleteCollection(ctx, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{}, &metainternalversion.ListOptions{
		LabelSelector: query.GetLabelSelector(),
		FieldSelector: query.GetFieldSelector(),
	}); err != nil {
		return err
	}
	return nil
}

func (l *operationOperator) operationFilter(obj runtime.Object, _ *query.Query) []runtime.Object {
	records, ok := obj.(*v1.OperationList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(records.Items))
	for index := range records.Items {
		objs = append(objs, &records.Items[index])
	}
	return objs
}
