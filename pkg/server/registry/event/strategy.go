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

package event

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var (
	_ rest.RESTCreateStrategy = EventStrategy{}
	_ rest.RESTUpdateStrategy = EventStrategy{}
	_ rest.RESTDeleteStrategy = EventStrategy{}
)

type EventStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) EventStrategy {
	return EventStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.Event)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Event")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.Event) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"userID":   obj.UserID,
		"username": obj.Username,
		"type":     obj.Type,
		"resource": obj.Resource,
		"verb":     obj.Verb,
		"ip":       obj.SourceIP,
	}, &obj.ObjectMeta, false)
}

func MatchEvent(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (EventStrategy) NamespaceScoped() bool {
	return false
}

func (EventStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (EventStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (EventStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (EventStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (EventStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (EventStrategy) Canonicalize(obj runtime.Object) {
}

func (EventStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (s EventStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (s EventStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
