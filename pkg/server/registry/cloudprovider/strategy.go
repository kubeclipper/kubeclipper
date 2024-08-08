/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package cloudprovider

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
	_ rest.RESTCreateStrategy = CloudProviderStrategy{}
	_ rest.RESTUpdateStrategy = CloudProviderStrategy{}
	_ rest.RESTDeleteStrategy = CloudProviderStrategy{}
)

type CloudProviderStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) CloudProviderStrategy {
	return CloudProviderStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.CloudProvider)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a CloudProvider")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.CloudProvider) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"name": obj.Name,
		"type": obj.Type,
	}, &obj.ObjectMeta, false)
}

func MatchCloudProvider(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (CloudProviderStrategy) NamespaceScoped() bool {
	return false
}

func (CloudProviderStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (CloudProviderStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (CloudProviderStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (CloudProviderStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (CloudProviderStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (CloudProviderStrategy) Canonicalize(obj runtime.Object) {
}

func (CloudProviderStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (CloudProviderStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (CloudProviderStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
