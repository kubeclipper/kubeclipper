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

package internal

import (
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
)

type NewInformerFunc func(clientset.Interface, time.Duration) cache.SharedIndexInformer

type SharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	InformerFor(obj runtime.Object, newFunc NewInformerFunc) cache.SharedIndexInformer
}

func NewProxyWatcher(w watch.Interface, cacheSize int) watch.Interface {
	ch := make(chan watch.Event, cacheSize)
	s := watch.NewProxyWatcher(ch)
	go func() {
		defer w.Stop()
		for {
			select {
			case <-s.StopChan():
				return
			case event, ok := <-w.ResultChan():
				if !ok {
					return
				}
				e := watch.Event{
					Type: event.Type,
				}
				obj := event.Object
				if co, ok := obj.(runtime.CacheableObject); ok {
					e.Object = co.GetObject()
				} else {
					e.Object = event.Object
				}
				if s.Stopping() {
					return
				}
				ch <- e
			}
		}
	}()
	return s
}
