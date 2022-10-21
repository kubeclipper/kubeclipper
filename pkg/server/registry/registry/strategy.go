package registry

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
	_ rest.RESTCreateStrategy = RegistryStrategy{}
	_ rest.RESTUpdateStrategy = RegistryStrategy{}
	_ rest.RESTDeleteStrategy = RegistryStrategy{}
)

type RegistryStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) RegistryStrategy {
	return RegistryStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.Registry)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a Registry")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.Registry) fields.Set {
	return generic.AddObjectMetaFieldsSet(fields.Set{
		"name": obj.Name,
	}, &obj.ObjectMeta, false)
}

func MatchRegistry(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (RegistryStrategy) NamespaceScoped() bool {
	return false
}

func (RegistryStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (RegistryStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (RegistryStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (RegistryStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (RegistryStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (RegistryStrategy) Canonicalize(obj runtime.Object) {
}

func (RegistryStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (RegistryStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (RegistryStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
