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

type LoginRecordLister interface {
	// List lists all LoginRecords in the indexer.
	List(selector labels.Selector) (ret []*iamv1.LoginRecord, err error)
	// Get retrieves the LoginRecord from the index for a given name.
	Get(name string) (*iamv1.LoginRecord, error)
	LoginRecordListerExpansion
}

type loginRecordLister struct {
	indexer cache.Indexer
}

func NewLoginRecordLister(indexer cache.Indexer) LoginRecordLister {
	return &loginRecordLister{indexer: indexer}
}

func (c *loginRecordLister) List(selector labels.Selector) (ret []*iamv1.LoginRecord, err error) {
	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*iamv1.LoginRecord))
	})
	return ret, err
}

func (c *loginRecordLister) Get(name string) (*iamv1.LoginRecord, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(iamv1.Resource("loginrecord"), name)
	}
	return obj.(*iamv1.LoginRecord), nil
}
