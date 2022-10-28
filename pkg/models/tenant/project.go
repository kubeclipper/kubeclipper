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

package tenant

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"k8s.io/apiserver/pkg/registry/rest"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"
)

var _ Operator = (*projectOperator)(nil)

type projectOperator struct {
	storage rest.StandardStorage
}

func NewProjectOperator(ss rest.StandardStorage) Operator {
	return &projectOperator{
		storage: ss,
	}
}

func (p *projectOperator) GetProject(ctx context.Context, name string) (*v1.Project, error) {
	return p.GetProjectEx(ctx, name, "")
}

func (p *projectOperator) ListProjects(ctx context.Context, query *query.Query) (*v1.ProjectList, error) {
	list, err := models.List(ctx, p.storage, query)
	if err != nil {
		return nil, err
	}
	return list.(*v1.ProjectList), nil
}

func (p *projectOperator) WatchProjects(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, p.storage, query)
}

func (p *projectOperator) GetProjectEx(ctx context.Context, name string, resourceVersion string) (*v1.Project, error) {
	op, err := models.GetV2(ctx, p.storage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return op.(*v1.Project), nil
}

func (p *projectOperator) ListProjectsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, p.storage, query, p.projectFilter, nil, nil)
}

func (p *projectOperator) CreateProject(ctx context.Context, project *v1.Project) (*v1.Project, error) {
	obj, err := p.storage.Create(ctx, project, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Project), nil
}

func (p *projectOperator) UpdateProject(ctx context.Context, project *v1.Project) (*v1.Project, error) {
	obj, wasCreated, err := p.storage.Update(ctx, project.Name, rest.DefaultUpdatedObjectInfo(project), nil, nil, false, &metav1.UpdateOptions{})
	if wasCreated {
		logger.Debug("project not exist, use create instead of update", zap.String("name", project.Name))
	}
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Project), nil
}

func (p *projectOperator) DeleteProject(ctx context.Context, name string) error {
	var err error
	_, _, err = p.storage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (p *projectOperator) projectFilter(obj runtime.Object, q *query.Query) []runtime.Object {
	projects, ok := obj.(*v1.ProjectList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(projects.Items))
	for index, project := range projects.Items {
		selected := true
		for k, v := range q.FuzzySearch {
			if !models.ObjectMetaFilter(project.ObjectMeta, k, v) {
				selected = false
			}
		}
		if selected {
			objs = append(objs, &projects.Items[index])
		}
	}
	return objs
}
