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

// Package project implement Project's StandardStorage
package project

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

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"
)

var (
	_ rest.RESTCreateStrategy = ProjectStrategy{}
	_ rest.RESTUpdateStrategy = ProjectStrategy{}
	_ rest.RESTDeleteStrategy = ProjectStrategy{}
)

// ProjectStrategy implement projectStrategy.
type ProjectStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// NewStrategy return a ProjectStrategy
func NewStrategy(typer runtime.ObjectTyper) ProjectStrategy {
	return ProjectStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs return a ProjectS's label and selectable fields.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.Project)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Project")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

// SelectableFields return obj's fields.Set.
func SelectableFields(obj *v1.Project) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

// MatchProject return a  SelectionPredicate.
func MatchProject(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// NamespaceScoped return is Project namespaced.
func (ProjectStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate hook trigger before create
func (ProjectStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

// PrepareForUpdate hook trigger before update
func (ProjectStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

// Validate check obj is valid.
func (ProjectStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// AllowCreateOnUpdate return is Project allow create on update
func (ProjectStrategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate return is Project allow unconditiona on update
func (ProjectStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize make obj canonicalize
func (ProjectStrategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate validate before update
func (ProjectStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the given create.
func (ProjectStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns warnings for the given update.
func (ProjectStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
