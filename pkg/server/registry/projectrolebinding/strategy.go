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

// Package projectrolebinding implement ProjectRoleBinding's StandardStorage
package projectrolebinding

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

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

var (
	_ rest.RESTCreateStrategy = ProjectRoleBindingStrategy{}
	_ rest.RESTUpdateStrategy = ProjectRoleBindingStrategy{}
	_ rest.RESTDeleteStrategy = ProjectRoleBindingStrategy{}
)

// ProjectRoleBindingStrategy implement ProjectRoleBinding Strategy.
type ProjectRoleBindingStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// NewStrategy return a ProjectRoleBindingStrategy
func NewStrategy(typer runtime.ObjectTyper) ProjectRoleBindingStrategy {
	return ProjectRoleBindingStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs return a ProjectRoleBindingS's label and selectable fields.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.ProjectRoleBinding)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a ProjectRoleBinding")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

// SelectableFields return obj's fields.Set.
func SelectableFields(obj *v1.ProjectRoleBinding) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

// MatchProjectRoleBinding return a  SelectionPredicate.
func MatchProjectRoleBinding(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// NamespaceScoped return is ProjectRoleBinding namespaced.
func (ProjectRoleBindingStrategy) NamespaceScoped() bool {
	return false
}

// PrepareForCreate hook trigger before create
func (ProjectRoleBindingStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

// PrepareForUpdate hook trigger before update
func (ProjectRoleBindingStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

// Validate check obj is valid.
func (ProjectRoleBindingStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// AllowCreateOnUpdate return is ProjectRoleBinding allow create on update
func (ProjectRoleBindingStrategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate return is ProjectRoleBinding allow unconditiona on update
func (ProjectRoleBindingStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize make obj canonicalize
func (ProjectRoleBindingStrategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate validate before update
func (ProjectRoleBindingStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// WarningsOnCreate returns warnings for the given create.
func (ProjectRoleBindingStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns warnings for the given update.
func (ProjectRoleBindingStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
