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

//go:generate mockgen -destination mock/mock_lease.go -source interface.go Operator

package lease

import (
	"context"

	"k8s.io/apimachinery/pkg/watch"

	coordinationv1 "k8s.io/api/coordination/v1"

	"github.com/kubeclipper/kubeclipper/pkg/query"
)

type Operator interface {
	LeaseReader
	LeaseWriter
}

type LeaseReader interface {
	ListLeases(ctx context.Context, query *query.Query) (*coordinationv1.LeaseList, error)
	WatchLease(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetLease(ctx context.Context, name string, resourceVersion string) (*coordinationv1.Lease, error)
	GetLeaseWithNamespace(ctx context.Context, name string, namespace string) (*coordinationv1.Lease, error)
	LeaseReaderEx
}

type LeaseReaderEx interface {
	GetLeaseWithNamespaceEx(ctx context.Context, name string, namespace, resourceVersion string) (*coordinationv1.Lease, error)
	// ListLeasesEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type LeaseWriter interface {
	CreateLease(ctx context.Context, lease *coordinationv1.Lease) (*coordinationv1.Lease, error)
	UpdateLease(ctx context.Context, lease *coordinationv1.Lease) (*coordinationv1.Lease, error)
}
