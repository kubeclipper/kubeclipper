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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func NewOperationStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newOperationStrategy(scheme)
	return newStorage(
		optsGetter, operations.ResourceOperations, &operations.Operation{}, &operations.OperationList{}, strategy, attrsForOperation,
	)
}

func NewTaskStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newTaskStrategy(scheme)
	return newStorage(
		optsGetter, operations.ResourceTasks, &operations.OperationTask{}, &operations.OperationTaskList{}, strategy, attrsForTask,
	)
}

func NewLockStorage(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.StandardStorage, error) {
	strategy := newLockStrategy(scheme)
	return newStorage(
		optsGetter, operations.ResourceLocks, &operations.ExecutionLock{}, &operations.ExecutionLockList{}, strategy, attrsForLock,
	)
}

func newStorage(
	optsGetter generic.RESTOptionsGetter,
	resourceName string,
	object, list runtime.Object,
	strategy rest.RESTCreateStrategy,
	attrs storage.AttrFunc,
) (rest.StandardStorage, error) {
	updateStrategy, ok := strategy.(rest.RESTUpdateStrategy)
	if !ok {
		return nil, fmt.Errorf("storage strategy must support update")
	}
	deleteStrategy, ok := strategy.(rest.RESTDeleteStrategy)
	if !ok {
		return nil, fmt.Errorf("storage strategy must support delete")
	}
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return object.DeepCopyObject() },
		NewListFunc:              func() runtime.Object { return list.DeepCopyObject() },
		DefaultQualifiedResource: operations.Resource(resourceName),
		CreateStrategy:           strategy,
		UpdateStrategy:           updateStrategy,
		DeleteStrategy:           deleteStrategy,
		TableConvertor:           rest.NewDefaultTableConvertor(operations.Resource(resourceName)),
		Storage:                  genericregistry.DryRunnableStorage{},
	}
	if err := store.CompleteWithOptions(&generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: attrs}); err != nil {
		return nil, err
	}
	return store, nil
}
