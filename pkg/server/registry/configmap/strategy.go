package configmap

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
	_ rest.RESTCreateStrategy = ConfigMapStrategy{}
	_ rest.RESTUpdateStrategy = ConfigMapStrategy{}
	_ rest.RESTDeleteStrategy = ConfigMapStrategy{}
)

type ConfigMapStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) ConfigMapStrategy {
	return ConfigMapStrategy{typer, names.SimpleNameGenerator}
}

func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	c, ok := obj.(*v1.ConfigMap)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a ConfigMap")
	}
	return c.ObjectMeta.Labels, SelectableFields(c), nil
}

func SelectableFields(obj *v1.ConfigMap) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}

func MatchConfigMap(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

func (ConfigMapStrategy) NamespaceScoped() bool {
	return false
}

func (ConfigMapStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (ConfigMapStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (ConfigMapStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (ConfigMapStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ConfigMapStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (ConfigMapStrategy) Canonicalize(obj runtime.Object) {
}

func (ConfigMapStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (ConfigMapStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (ConfigMapStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
