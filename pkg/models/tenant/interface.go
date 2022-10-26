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

//go:generate mockgen -destination mock/mock_project.go -source interface.go Operator

package tenant

import (
	"context"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"
)

type Operator interface {
	ProjectReader
	ProjectWriter
}

type ProjectReader interface {
	GetProject(ctx context.Context, name string) (*v1.Project, error)
	ListProjects(ctx context.Context, query *query.Query) (*v1.ProjectList, error)
	WatchProjects(ctx context.Context, query *query.Query) (watch.Interface, error)
	ProjectReaderEx
}

type ProjectReaderEx interface {
	GetProjectEx(ctx context.Context, name string, resourceVersion string) (*v1.Project, error)
	ListProjectsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type ProjectWriter interface {
	CreateProject(ctx context.Context, r *v1.Project) (*v1.Project, error)
	UpdateProject(ctx context.Context, r *v1.Project) (*v1.Project, error)
	DeleteProject(ctx context.Context, name string) error
}
