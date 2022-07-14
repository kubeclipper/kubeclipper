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

//go:generate mockgen -destination mock/mock_operation.go -source interface.go Operator

package operation

import (
	"context"

	"github.com/kubeclipper/kubeclipper/pkg/models"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Reader interface {
	ListOperations(ctx context.Context, query *query.Query) (*v1.OperationList, error)
	WatchOperations(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetOperation(ctx context.Context, name string) (*v1.Operation, error)
	ReaderEx
}

type ReaderEx interface {
	GetOperationEx(ctx context.Context, name string, resourceVersion string) (*v1.Operation, error)
	ListOperationsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type Writer interface {
	DeleteOperation(ctx context.Context, name string) error
	CreateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error)
	UpdateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error)
	UpdateOperationStatus(ctx context.Context, name string, status *v1.OperationStatus) (*v1.Operation, error)
	DeleteOperationCollection(ctx context.Context, query *query.Query) error
}

type Operator interface {
	Reader
	Writer
}
