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

package platform

import (
	"context"
	"strings"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"

	"go.uber.org/zap"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"k8s.io/apiserver/pkg/registry/rest"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var _ Operator = (*platformOperator)(nil)

type platformOperator struct {
	storage      rest.StandardStorage
	eventStorage rest.StandardStorage
}

func NewPlatformOperator(operationStorage rest.StandardStorage, eventStorage rest.StandardStorage) Operator {
	return &platformOperator{
		storage:      operationStorage,
		eventStorage: eventStorage,
	}
}

func (p *platformOperator) GetPlatformSetting(ctx context.Context) (*v1.PlatformSetting, error) {
	list, err := p.storage.List(ctx, &metainternalversion.ListOptions{
		ResourceVersion:      "0",
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return nil, err
	}
	result, _ := list.(*v1.PlatformSettingList)
	if len(result.Items) == 0 {
		return &v1.PlatformSetting{}, nil
	}

	return &result.Items[0], nil
}

func (p *platformOperator) CreatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	ctx = genericapirequest.WithNamespace(ctx, platformSetting.Namespace)
	obj, err := p.storage.Create(ctx, platformSetting, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*v1.PlatformSetting), nil
}

func (p *platformOperator) UpdatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	ctx = genericapirequest.WithNamespace(ctx, platformSetting.Namespace)
	obj, wasCreated, err := p.storage.Update(ctx, platformSetting.Name, rest.DefaultUpdatedObjectInfo(platformSetting), nil, nil, false, &metav1.UpdateOptions{})
	if wasCreated {
		logger.Debug("platform not exist, use create instead of update", zap.String("name", platformSetting.Name))
	}
	if err != nil {
		return nil, err
	}
	return obj.(*v1.PlatformSetting), nil
}

func (p *platformOperator) ListEvents(ctx context.Context, query *query.Query) (*v1.EventList, error) {
	list, err := models.List(ctx, p.eventStorage, query)
	if err != nil {
		return nil, err
	}
	return list.(*v1.EventList), nil
}

func (p *platformOperator) GetEvent(ctx context.Context, name string) (*v1.Event, error) {
	return p.GetEventEx(ctx, name, "")
}

func (p *platformOperator) CreateEvent(ctx context.Context, Event *v1.Event) (*v1.Event, error) {
	ctx = genericapirequest.WithNamespace(ctx, Event.Namespace)
	obj, err := p.eventStorage.Create(ctx, Event, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Event), nil
}

func (p *platformOperator) DeleteEvent(ctx context.Context, name string) error {
	var err error
	_, _, err = p.eventStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (p *platformOperator) DeleteEventCollection(ctx context.Context, query *query.Query) error {
	if _, err := p.eventStorage.DeleteCollection(ctx, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{}, &metainternalversion.ListOptions{
		LabelSelector: query.GetLabelSelector(),
		FieldSelector: query.GetFieldSelector(),
	}); err != nil {
		return err
	}
	return nil
}

func (p *platformOperator) GetEventEx(ctx context.Context, name string, resourceVersion string) (*v1.Event, error) {
	op, err := models.GetV2(ctx, p.eventStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return op.(*v1.Event), nil
}

func (p *platformOperator) ListEventsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, p.eventStorage, query, p.eventFilter, nil, nil)
}

func (p *platformOperator) eventFilter(obj runtime.Object, q *query.Query) []runtime.Object {
	records, ok := obj.(*v1.EventList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(records.Items))
	for index := range records.Items {
		selected := true
		for k, v := range q.FuzzySearch {
			if !eventCustomFilter(&records.Items[index], k, v) {
				selected = false
			}
		}
		if selected {
			objs = append(objs, &records.Items[index])
		}
	}
	return objs
}

func eventCustomFilter(ev *v1.Event, key, value string) bool {
	switch key {
	case "ip":
		if !strings.Contains(ev.SourceIP, value) {
			return false
		}
	case "username":
		if !strings.Contains(ev.Username, value) {
			return false
		}
	}
	return true
}
