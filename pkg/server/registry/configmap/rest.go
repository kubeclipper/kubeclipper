package configmap

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func NewStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc: func() runtime.Object {
			return &v1.ConfigMap{}
		},
		NewListFunc: func() runtime.Object {
			return &v1.ConfigMapList{}
		},
		DefaultQualifiedResource: v1.Resource("configmaps"),
		KeyRootFunc:              nil,
		KeyFunc:                  nil,
		ObjectNameFunc:           nil,
		TTLFunc:                  nil,
		PredicateFunc:            MatchConfigMap,
		EnableGarbageCollection:  false,
		DeleteCollectionWorkers:  0,
		Decorator:                nil,
		CreateStrategy:           strategy,
		BeginCreate:              nil,
		AfterCreate:              nil,
		UpdateStrategy:           strategy,
		BeginUpdate:              nil,
		AfterUpdate:              nil,
		DeleteStrategy:           strategy,
		AfterDelete:              nil,
		ReturnDeletedObject:      false,
		ShouldDeleteDuringUpdate: nil,
		TableConvertor:           rest.NewDefaultTableConvertor(v1.Resource("configmaps")),
		ResetFieldsStrategy:      nil,
		Storage:                  genericregistry.DryRunnableStorage{},
		StorageVersioner:         nil,
		DestroyFunc:              nil,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
