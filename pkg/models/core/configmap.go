/*
 *
 *  * Copyright 2022 KubeClipper Authors.
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

package core

import (
	"context"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type ConfigMapReaderEx interface {
	GetConfigMapEx(ctx context.Context, name string, resourceVersion string) (*v1.ConfigMap, error)
	ListConfigMapsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type ConfigMapReader interface {
	ListConfigMaps(ctx context.Context, query *query.Query) (*v1.ConfigMapList, error)
	WatchConfigMaps(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetConfigMap(ctx context.Context, name string) (*v1.ConfigMap, error)
	ConfigMapReaderEx
}

type ConfigMapWriter interface {
	CreateConfigMap(ctx context.Context, configmap *v1.ConfigMap) (*v1.ConfigMap, error)
	UpdateConfigMap(ctx context.Context, configmap *v1.ConfigMap) (*v1.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, name string) error
}

func (o operator) ListConfigMaps(ctx context.Context, query *query.Query) (*v1.ConfigMapList, error) {
	list, err := models.List(ctx, o.cmStorage, query)
	if err != nil {
		return nil, err
	}
	list.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
	return list.(*v1.ConfigMapList), nil
}

func (o operator) WatchConfigMaps(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, o.cmStorage, query)
}

func (o operator) GetConfigMap(ctx context.Context, name string) (*v1.ConfigMap, error) {
	return o.GetConfigMapEx(ctx, name, "")
}

func (o operator) GetConfigMapEx(ctx context.Context, name string, resourceVersion string) (*v1.ConfigMap, error) {
	cm, err := models.GetV2(ctx, o.cmStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return cm.(*v1.ConfigMap), nil
}

func (o operator) ListConfigMapsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, o.cmStorage, query, o.configmapFuzzyFilter, nil, nil)
}

func (o operator) CreateConfigMap(ctx context.Context, configmap *v1.ConfigMap) (*v1.ConfigMap, error) {
	ctx = genericapirequest.WithNamespace(ctx, configmap.Namespace)
	obj, err := o.cmStorage.Create(ctx, configmap, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*v1.ConfigMap), nil
}

func (o operator) UpdateConfigMap(ctx context.Context, configmap *v1.ConfigMap) (*v1.ConfigMap, error) {
	ctx = genericapirequest.WithNamespace(ctx, configmap.Namespace)
	obj, wasCreated, err := o.cmStorage.Update(ctx, configmap.Name, rest.DefaultUpdatedObjectInfo(configmap),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if wasCreated {
		logger.Debug("configmap not exist, use create instead of update", zap.String("configmap", configmap.Name))
	}
	return obj.(*v1.ConfigMap), nil
}

func (o operator) DeleteConfigMap(ctx context.Context, name string) error {
	var err error
	_, _, err = o.cmStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (o operator) configmapFuzzyFilter(obj runtime.Object, q *query.Query) []runtime.Object {
	cms, ok := obj.(*v1.ConfigMapList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(cms.Items))
	for index, cm := range cms.Items {
		selected := true
		for k, v := range q.FuzzySearch {
			if !models.ObjectMetaFilter(cm.ObjectMeta, k, v) {
				selected = false
			}
		}
		if selected {
			objs = append(objs, &cms.Items[index])
		}
	}
	return objs
}
