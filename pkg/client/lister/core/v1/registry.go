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

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var _ RegistryLister = (*registryLister)(nil)

type RegistryLister interface {
	// List lists all Registry in the indexer.
	List(selector labels.Selector) (ret []*v1.Registry, err error)
	// Get retrieves the Registry from the index for a given name.
	Get(name string) (*v1.Registry, error)
	RegistryListerExpansion
}

type registryLister struct {
	indexer cache.Indexer
}

func NewRegistryLister(indexer cache.Indexer) RegistryLister {
	return &registryLister{indexer: indexer}
}

func (c *registryLister) List(selector labels.Selector) (ret []*v1.Registry, err error) {
	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Registry))
	})
	return ret, err
}

func (c *registryLister) Get(name string) (*v1.Registry, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("registry"), name)
	}
	return obj.(*v1.Registry), nil
}
