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
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"k8s.io/apimachinery/pkg/watch"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/query"
)

type PageableResponse struct {
	Items      []interface{} `json:"items" description:"paging data"`
	TotalCount int           `json:"totalCount" description:"total count"`
}

// Deprecated
func Get(ctx context.Context, s rest.StandardStorage, name string, resourceVersion string) (runtime.Object, error) {
	obj, err := s.Get(ctx, name, &metav1.GetOptions{
		ResourceVersion: resourceVersion,
	})
	if err != nil {
		return nil, err
	}
	return obj.DeepCopyObject(), nil

}

func GetV2(ctx context.Context, s rest.StandardStorage, name string, resourceVersion string, mutatingFunc MutatingFunc) (runtime.Object, error) {
	obj, err := s.Get(ctx, name, &metav1.GetOptions{
		ResourceVersion: resourceVersion,
	})
	if err != nil {
		return nil, err
	}
	if mutatingFunc != nil {
		return mutatingFunc(obj), nil
	}
	return obj, nil
}

func List(ctx context.Context, s rest.StandardStorage, q *query.Query) (runtime.Object, error) {
	return s.List(ctx, &metainternalversion.ListOptions{
		LabelSelector:        q.GetLabelSelector(),
		FieldSelector:        q.GetFieldSelector(),
		Limit:                q.Limit,
		Continue:             q.Continue,
		ResourceVersion:      q.ResourceVersion,
		ResourceVersionMatch: metav1.ResourceVersionMatch(q.ResourceVersionMatch),
		Watch:                q.Watch,
		AllowWatchBookmarks:  q.AllowWatchBookmarks,
		TimeoutSeconds:       q.TimeoutSeconds,
	})
}

func Watch(ctx context.Context, s rest.StandardStorage, q *query.Query) (watch.Interface, error) {
	return s.Watch(ctx, &metainternalversion.ListOptions{
		TypeMeta:             metav1.TypeMeta{},
		LabelSelector:        q.GetLabelSelector(),
		FieldSelector:        q.GetFieldSelector(),
		Limit:                q.Limit,
		Continue:             q.Continue,
		ResourceVersion:      q.ResourceVersion,
		ResourceVersionMatch: metav1.ResourceVersionMatch(q.ResourceVersionMatch),
		Watch:                q.Watch,
		AllowWatchBookmarks:  q.AllowWatchBookmarks,
		TimeoutSeconds:       q.TimeoutSeconds,
	})
}

func ListExV2(ctx context.Context, s rest.StandardStorage, q *query.Query, litToSliceFunc ListToObjectSliceFunction, compareFunc CompareFunc, mutatingFunc MutatingFunc) (*PageableResponse, error) {
	if mutatingFunc == nil {
		mutatingFunc = DefaultMutatingFunc
	}
	if compareFunc == nil {
		compareFunc = DefaultCompareFunc
	}
	list, err := s.List(ctx, &metainternalversion.ListOptions{
		LabelSelector:        q.GetLabelSelector(),
		FieldSelector:        q.GetFieldSelector(),
		ResourceVersion:      "0",
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	})
	if err != nil {
		return nil, err
	}
	list = list.DeepCopyObject()
	objs := litToSliceFunc(list, q)
	totalCount := len(objs)
	// sort
	sort.Slice(objs, func(i, j int) bool {
		if q.Reverse {
			return !compareFunc(objs[i], objs[j], q.OrderBy)
		}
		return compareFunc(objs[i], objs[j], q.OrderBy)
	})
	// limit offset
	start, end := q.Pagination.GetValidPagination(totalCount)

	objs = objs[start:end]
	items := make([]interface{}, 0, len(objs))
	for i := range objs {
		items = append(items, mutatingFunc(objs[i]))
	}
	return &PageableResponse{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

// DefaultList a func to build a PageableResponse
func DefaultList(object runtime.Object, q *query.Query, litToSliceFunc ListToObjectSliceFunction, compareFunc CompareFunc, mutatingFunc MutatingFunc) (*PageableResponse, error) {
	if mutatingFunc == nil {
		mutatingFunc = DefaultMutatingFunc
	}
	if compareFunc == nil {
		compareFunc = DefaultCompareFunc
	}

	object = object.DeepCopyObject()
	objs := litToSliceFunc(object, q)
	totalCount := len(objs)
	// sort
	sort.Slice(objs, func(i, j int) bool {
		if q.Reverse {
			return !compareFunc(objs[i], objs[j], q.OrderBy)
		}
		return compareFunc(objs[i], objs[j], q.OrderBy)
	})
	// limit offset
	start, end := q.Pagination.GetValidPagination(totalCount)

	objs = objs[start:end]
	items := make([]interface{}, 0, len(objs))
	for i := range objs {
		items = append(items, mutatingFunc(objs[i]))
	}
	return &PageableResponse{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

// DefaultListMerge a func to build a PageableResponse from many obj
func DefaultListMerge(objects []runtime.Object, litToSliceFunc ListToObjectSliceFunction, compareFunc CompareFunc, mutatingFunc MutatingFunc) (*PageableResponse, error) {
	if mutatingFunc == nil {
		mutatingFunc = DefaultMutatingFunc
	}
	if compareFunc == nil {
		compareFunc = DefaultCompareFunc
	}
	objs := make([]runtime.Object, 0)
	q := query.New()
	for _, object := range objects {
		obj := litToSliceFunc(object.DeepCopyObject(), q)
		objs = append(objs, obj...)
	}
	totalCount := len(objs)
	// sort
	sort.Slice(objs, func(i, j int) bool {
		if q.Reverse {
			return !compareFunc(objs[i], objs[j], q.OrderBy)
		}
		return compareFunc(objs[i], objs[j], q.OrderBy)
	})
	// limit offset
	start, end := q.Pagination.GetValidPagination(totalCount)

	objs = objs[start:end]
	items := make([]interface{}, 0, len(objs))
	for i := range objs {
		items = append(items, mutatingFunc(objs[i]))
	}
	return &PageableResponse{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

type ListToObjectSliceFunction func(runtime.Object, *query.Query) []runtime.Object
type CompareFunc func(left runtime.Object, right runtime.Object, orderBy string) bool
type FilterFunc func(obj runtime.Object) bool
type MutatingFunc func(obj runtime.Object) runtime.Object

var DefaultMutatingFunc MutatingFunc = func(obj runtime.Object) runtime.Object {
	return obj
}

var DefaultCompareFunc CompareFunc = func(left runtime.Object, right runtime.Object, _ string) bool {
	leftMeta, _ := meta.Accessor(left)
	rightMeta, _ := meta.Accessor(right)
	if leftMeta.GetCreationTimestamp().Time.Equal(rightMeta.GetCreationTimestamp().Time) {
		return leftMeta.GetName() > rightMeta.GetName()
	}
	return leftMeta.GetCreationTimestamp().Time.After(rightMeta.GetCreationTimestamp().Time)
}

func ObjectMetaFilter(obj metav1.ObjectMeta, key, value string) bool {
	switch key {
	case "name":
		if !strings.Contains(obj.Name, value) {
			return false
		}
	case "display-name":
		if !strings.Contains(obj.Annotations[common.AnnotationDisplayName], value) {
			return false
		}
	case "description":
		if !strings.Contains(obj.Annotations[common.AnnotationDescription], value) {
			return false
		}
	}
	return true
}
