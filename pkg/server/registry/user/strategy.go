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

package user

import (
	"context"
	"fmt"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

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
	_ rest.RESTCreateStrategy = UserStrategy{}
	_ rest.RESTUpdateStrategy = UserStrategy{}
	_ rest.RESTDeleteStrategy = UserStrategy{}
)

type UserStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (s UserStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (s UserStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func NewStrategy(typer runtime.ObjectTyper) UserStrategy {
	return UserStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*iamv1.User)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a User")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *iamv1.User) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"spec.email":   obj.Spec.Email,
		"spec.phone":   obj.Spec.Phone,
		"status.state": string(*obj.Status.State),
	}, &obj.ObjectMeta, false)
}

func MatchUser(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (UserStrategy) NamespaceScoped() bool {
	return false
}

func (UserStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (UserStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (UserStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (UserStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (UserStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (UserStrategy) Canonicalize(obj runtime.Object) {
}

func (UserStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
