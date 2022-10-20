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
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers/internalinterfaces"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
	"github.com/kubeclipper/kubeclipper/pkg/client/internal"
	iamv1Lister "github.com/kubeclipper/kubeclipper/pkg/client/lister/iam/v1"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

type LoginRecordInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() iamv1Lister.LoginRecordLister
}

type loginRecordInformer struct {
	factory          internal.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

func NewLoginRecordInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredLoginRecordInformer(client, resyncPeriod, indexers, nil)
}

func NewFilteredLoginRecordInformer(client clientset.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IamV1().LoginRecords().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.IamV1().LoginRecords().Watch(context.TODO(), options)
			},
		},
		&iamv1.LoginRecord{},
		resyncPeriod,
		indexers,
	)
}

func (f *loginRecordInformer) defaultInformer(client clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredLoginRecordInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *loginRecordInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&iamv1.LoginRecord{}, f.defaultInformer)
}

func (f *loginRecordInformer) Lister() iamv1Lister.LoginRecordLister {
	return iamv1Lister.NewLoginRecordLister(f.Informer().GetIndexer())
}
