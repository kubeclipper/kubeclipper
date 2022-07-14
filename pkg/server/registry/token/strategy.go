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

package token

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
	_ rest.RESTCreateStrategy = TokenStrategy{}
	_ rest.RESTUpdateStrategy = TokenStrategy{}
	_ rest.RESTDeleteStrategy = TokenStrategy{}
)

type TokenStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (s TokenStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (s TokenStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func NewStrategy(typer runtime.ObjectTyper) TokenStrategy {
	return TokenStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.Token)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Token")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.Token) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"spec.tokenType": string(obj.Spec.TokenType),
		"spec.username":  obj.Spec.Username,
		"spec.issuer":    obj.Spec.Issuer,
		"spec.token":     obj.Spec.Token,
	}, &obj.ObjectMeta, false)
}

func MatchToken(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (TokenStrategy) NamespaceScoped() bool {
	return false
}

func (TokenStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (TokenStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (TokenStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (TokenStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (TokenStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (TokenStrategy) Canonicalize(obj runtime.Object) {
}

func (TokenStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
