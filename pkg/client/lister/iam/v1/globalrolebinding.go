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

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type GlobalRoleBindingLister interface {
	// List lists all Users in the indexer.
	List(selector labels.Selector) (ret []*iamv1.GlobalRoleBinding, err error)
	// Get retrieves the User from the index for a given name.
	Get(name string) (*iamv1.GlobalRoleBinding, error)
	GlobalRoleBindingListerExpansion
}

type globalRoleBindingLister struct {
	indexer cache.Indexer
}

func NewGlobalRoleBindingLister(indexer cache.Indexer) GlobalRoleBindingLister {
	return &globalRoleBindingLister{indexer: indexer}
}

func (c *globalRoleBindingLister) List(selector labels.Selector) (ret []*iamv1.GlobalRoleBinding, err error) {
	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*iamv1.GlobalRoleBinding))
	})
	return ret, err
}

func (c *globalRoleBindingLister) Get(name string) (*iamv1.GlobalRoleBinding, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(iamv1.Resource("globalrolebinding"), name)
	}
	return obj.(*iamv1.GlobalRoleBinding), nil
}
