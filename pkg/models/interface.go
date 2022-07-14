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

package models

import (
	"context"
	"sort"

	"go.uber.org/zap"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type Object interface {
	runtime.Object
	metav1.Object
}

type ObjectList interface {
	runtime.Object
	metav1.ListInterface
}

type Reader[OBJ Object, LIST ObjectList] interface {
	List(ctx context.Context, query *query.Query) (LIST, error)
	Watch(ctx context.Context, query *query.Query) (watch.Interface, error)
	Get(ctx context.Context, name string) (OBJ, error)
	GetEx(ctx context.Context, name string, resourceVersion string) (OBJ, error)
	ListEx(ctx context.Context, query *query.Query) (*PageableResponse, error)
}

type Writer[OBJ Object, LIST ObjectList] interface {
	Create(ctx context.Context, obj OBJ) (OBJ, error)
	Update(ctx context.Context, obj OBJ) (OBJ, error)
	Delete(ctx context.Context, name string) error
	DeleteCollection(ctx context.Context, query *query.Query) error
}

var _ Reader[Object, ObjectList] = (*Operator[Object, ObjectList])(nil)
var _ Writer[Object, ObjectList] = (*Operator[Object, ObjectList])(nil)

type Operator[OBJ Object, LIST ObjectList] struct {
	storage         rest.StandardStorage
	ListKind        string
	Kind            string
	listToSliceFunc ListToObjectSliceFunction
	compareFunc     CompareFunc
	mutatingFunc    MutatingFunc
}

func (o *Operator[OBJ, LIST]) List(ctx context.Context, query *query.Query) (LIST, error) {
	var result LIST
	list, err := o.storage.List(ctx, &metainternalversion.ListOptions{
		LabelSelector:        query.GetLabelSelector(),
		FieldSelector:        query.GetFieldSelector(),
		Limit:                query.Limit,
		Continue:             query.Continue,
		ResourceVersion:      query.ResourceVersion,
		ResourceVersionMatch: metav1.ResourceVersionMatch(query.ResourceVersionMatch),
		Watch:                query.Watch,
		AllowWatchBookmarks:  query.AllowWatchBookmarks,
		TimeoutSeconds:       query.TimeoutSeconds,
	})
	if err != nil {
		return result, err
	}
	list.GetObjectKind().SetGroupVersionKind(v1.SchemeGroupVersion.WithKind(o.ListKind))
	result, _ = list.(LIST)
	return result, nil
}

func (o *Operator[OBJ, LIST]) Watch(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return o.storage.Watch(ctx, &metainternalversion.ListOptions{
		TypeMeta:             metav1.TypeMeta{},
		LabelSelector:        query.GetLabelSelector(),
		FieldSelector:        query.GetFieldSelector(),
		Limit:                query.Limit,
		Continue:             query.Continue,
		ResourceVersion:      query.ResourceVersion,
		ResourceVersionMatch: metav1.ResourceVersionMatch(query.ResourceVersionMatch),
		Watch:                query.Watch,
		AllowWatchBookmarks:  query.AllowWatchBookmarks,
		TimeoutSeconds:       query.TimeoutSeconds,
	})
}

func (o *Operator[OBJ, LIST]) Get(ctx context.Context, name string) (OBJ, error) {
	return o.GetEx(ctx, name, "")
}

func (o *Operator[OBJ, LIST]) GetEx(ctx context.Context, name string, resourceVersion string) (OBJ, error) {
	var obj OBJ
	result, err := Get(ctx, o.storage, name, resourceVersion)
	if err != nil {
		return obj, err
	}
	obj, _ = result.(OBJ)
	return obj, nil
}

func (o *Operator[OBJ, LIST]) ListEx(ctx context.Context, q *query.Query) (*PageableResponse, error) {
	if o.mutatingFunc == nil {
		o.mutatingFunc = DefaultMutatingFunc
	}
	if o.compareFunc == nil {
		o.compareFunc = DefaultCompareFunc
	}
	list, err := o.storage.List(ctx, &metainternalversion.ListOptions{
		LabelSelector:        q.GetLabelSelector(),
		FieldSelector:        q.GetFieldSelector(),
		ResourceVersion:      "0",
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return nil, err
	}
	// search by fuzzy
	objs := o.listToSliceFunc(list, q)
	totalCount := len(objs)
	// sort
	sort.Slice(objs, func(i, j int) bool {
		if q.Reverse {
			return !o.compareFunc(objs[i], objs[j], q.OrderBy)
		}
		return o.compareFunc(objs[i], objs[j], q.OrderBy)
	})
	// limit offset
	start, end := q.Pagination.GetValidPagination(totalCount)

	objs = objs[start:end]
	items := make([]interface{}, 0, len(objs))
	for i := range objs {
		items = append(items, o.mutatingFunc(objs[i]))
	}
	return &PageableResponse{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

func (o *Operator[OBJ, LIST]) Create(ctx context.Context, obj OBJ) (OBJ, error) {
	var result OBJ
	created, err := o.storage.Create(ctx, obj, nil, &metav1.CreateOptions{})
	if err != nil {
		return result, err
	}
	result, _ = created.(OBJ)
	return result, nil
}

func (o *Operator[OBJ, LIST]) Update(ctx context.Context, obj OBJ) (OBJ, error) {
	var result OBJ
	updated, wasCreated, err := o.storage.Update(ctx, obj.GetName(), rest.DefaultUpdatedObjectInfo(obj),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return result, err
	}
	if wasCreated {
		logger.Debug("cluster not exist, use create instead of update", zap.String("cluster", obj.GetName()))
	}
	result, _ = updated.(OBJ)
	return result, nil
}

func (o *Operator[OBJ, LIST]) Delete(ctx context.Context, name string) error {
	var err error
	_, _, err = o.storage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (o *Operator[OBJ, LIST]) DeleteCollection(ctx context.Context, query *query.Query) error {
	if _, err := o.storage.DeleteCollection(ctx, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{}, &metainternalversion.ListOptions{
		LabelSelector: query.GetLabelSelector(),
		FieldSelector: query.GetFieldSelector(),
	}); err != nil {
		return err
	}
	return nil
}

func (o *Operator[OBJ, LIST]) RawStorage() rest.StandardStorage {
	return o.storage
}
