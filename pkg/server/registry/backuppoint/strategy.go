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

package backuppoint

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
	_ rest.RESTCreateStrategy = BackupPointStrategy{}
	_ rest.RESTUpdateStrategy = BackupPointStrategy{}
	_ rest.RESTDeleteStrategy = BackupPointStrategy{}
)

type BackupPointStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (s BackupPointStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (s BackupPointStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func NewStrategy(typer runtime.ObjectTyper) BackupPointStrategy {
	return BackupPointStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.BackupPoint)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a BackupPoint")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.BackupPoint) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

func MatchBackupPoint(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (BackupPointStrategy) NamespaceScoped() bool {
	return false
}

func (BackupPointStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (BackupPointStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (BackupPointStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (BackupPointStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (BackupPointStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (BackupPointStrategy) Canonicalize(obj runtime.Object) {
}

func (BackupPointStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
