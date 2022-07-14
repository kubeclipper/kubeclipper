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

package loginrecord

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
	_ rest.RESTCreateStrategy = LoginRecordStrategy{}
	_ rest.RESTUpdateStrategy = LoginRecordStrategy{}
	_ rest.RESTDeleteStrategy = LoginRecordStrategy{}
)

type LoginRecordStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) LoginRecordStrategy {
	return LoginRecordStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.LoginRecord)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a LoginRecord")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.LoginRecord) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"ip":       obj.Spec.SourceIP,
		"type":     string(obj.Spec.Type),
		"provider": obj.Spec.Provider,
	}, &obj.ObjectMeta, false)
}

func MatchLoginRecord(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (LoginRecordStrategy) NamespaceScoped() bool {
	return false
}

func (LoginRecordStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (LoginRecordStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (LoginRecordStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (LoginRecordStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (LoginRecordStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (LoginRecordStrategy) Canonicalize(obj runtime.Object) {
}

func (LoginRecordStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (s LoginRecordStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (s LoginRecordStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
