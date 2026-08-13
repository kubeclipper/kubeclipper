/*
 *
 *  * Copyright 2026 KubeClipper Authors.
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

package operationv2

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func NewOperationStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newOperationStrategy(scheme)
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &operations.Operation{} },
		NewListFunc:              func() runtime.Object { return &operations.OperationList{} },
		DefaultQualifiedResource: operations.Resource(operations.ResourceOperations),
		CreateStrategy:           strategy,
		UpdateStrategy:           strategy,
		DeleteStrategy:           strategy,
		TableConvertor:           rest.NewDefaultTableConvertor(operations.Resource(operations.ResourceOperations)),
		Storage:                  genericregistry.DryRunnableStorage{},
	}
	if err := store.CompleteWithOptions(&generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: attrsForOperation}); err != nil {
		return nil, err
	}
	return store, nil
}

func NewTaskStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newTaskStrategy(scheme)
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &operations.OperationTask{} },
		NewListFunc:              func() runtime.Object { return &operations.OperationTaskList{} },
		DefaultQualifiedResource: operations.Resource(operations.ResourceTasks),
		CreateStrategy:           strategy,
		UpdateStrategy:           strategy,
		DeleteStrategy:           strategy,
		TableConvertor:           rest.NewDefaultTableConvertor(operations.Resource(operations.ResourceTasks)),
		Storage:                  genericregistry.DryRunnableStorage{},
	}
	if err := store.CompleteWithOptions(&generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: attrsForTask}); err != nil {
		return nil, err
	}
	return store, nil
}

func NewLockStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newLockStrategy(scheme)
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &operations.ExecutionLock{} },
		NewListFunc:              func() runtime.Object { return &operations.ExecutionLockList{} },
		DefaultQualifiedResource: operations.Resource(operations.ResourceLocks),
		CreateStrategy:           strategy,
		UpdateStrategy:           strategy,
		DeleteStrategy:           strategy,
		TableConvertor:           rest.NewDefaultTableConvertor(operations.Resource(operations.ResourceLocks)),
		Storage:                  genericregistry.DryRunnableStorage{},
	}
	if err := store.CompleteWithOptions(&generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: attrsForLock}); err != nil {
		return nil, err
	}
	return store, nil
}
