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

var _ BackupLister = (*backupLister)(nil)

type BackupLister interface {
	// List lists all Backups in the indexer.
	List(selector labels.Selector) (ret []*v1.Backup, err error)
	// Get retrieves the Backup from the index for a given name.
	Get(name string) (*v1.Backup, error)
	BackupListerExpansion
}

type backupLister struct {
	indexer cache.Indexer
}

func NewBackupLister(indexer cache.Indexer) BackupLister {
	return &backupLister{indexer: indexer}
}

func (l *backupLister) List(selector labels.Selector) (ret []*v1.Backup, err error) {
	err = cache.ListAll(l.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Backup))
	})
	return ret, err
}

func (l *backupLister) Get(name string) (*v1.Backup, error) {
	obj, exists, err := l.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("backup"), name)
	}
	return obj.(*v1.Backup), nil
}
