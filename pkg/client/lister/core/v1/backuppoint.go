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

var _ BackupPointLister = (*backupPointLister)(nil)

type BackupPointLister interface {
	// List lists all Backups in the indexer.
	List(selector labels.Selector) (ret []*v1.BackupPoint, err error)
	// Get retrieves the Backup from the index for a given name.
	Get(name string) (*v1.BackupPoint, error)
	BackupPointListerExpansion
}

type backupPointLister struct {
	indexer cache.Indexer
}

func NewBackupPointLister(indexer cache.Indexer) BackupPointLister {
	return &backupPointLister{indexer: indexer}
}

func (l *backupPointLister) List(selector labels.Selector) (ret []*v1.BackupPoint, err error) {
	err = cache.ListAll(l.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.BackupPoint))
	})
	return ret, err
}

func (l *backupPointLister) Get(name string) (*v1.BackupPoint, error) {
	obj, exists, err := l.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("backuppoint"), name)
	}
	return obj.(*v1.BackupPoint), nil
}
