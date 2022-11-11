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

// Package projectrole implement ProjectRole's StandardStorage
package projectrole

import (
	"context"
	"fmt"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

var (
	_ rest.RESTCreateStrategy = ProjectRoleStrategy{}
	_ rest.RESTUpdateStrategy = ProjectRoleStrategy{}
	_ rest.RESTDeleteStrategy = ProjectRoleStrategy{}
)

// ProjectRoleStrategy implement ProjectRoleStrategy.
type ProjectRoleStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// NewStrategy return a ProjectRoleStrategy
func NewStrategy(typer runtime.ObjectTyper) ProjectRoleStrategy {
	return ProjectRoleStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs return a ProjectRoleS's label and selectable fields.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.ProjectRole)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a ProjectRole")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

// SelectableFields return obj's fields.Set.
func SelectableFields(obj *v1.ProjectRole) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

// MatchProjectRole return a  SelectionPredicate.
func MatchProjectRole(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// NamespaceScoped return is ProjectRole namespaced.
func (ProjectRoleStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate hook trigger before create
func (ProjectRoleStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

// PrepareForUpdate hook trigger before update
func (ProjectRoleStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

// Validate check obj is valid.
func (ProjectRoleStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// AllowCreateOnUpdate return is ProjectRole allow create on update
func (ProjectRoleStrategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate return is ProjectRole allow unconditiona on update
func (ProjectRoleStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize make obj canonicalize
func (ProjectRoleStrategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate validate before update
func (ProjectRoleStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the given create.
func (ProjectRoleStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns warnings for the given update.
func (ProjectRoleStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
