/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package restplus

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
)

type multiWatcher struct {
	watchers []watch.Interface
	stop     chan struct{}

	ch      chan watch.Event
	stopped bool
	sync.Mutex
}

func NewMultiWatcher(watchers []watch.Interface) watch.Interface {
	m := &multiWatcher{
		watchers: watchers,
		ch:       make(chan watch.Event),
		stop:     make(chan struct{}),
	}

	// merge watchers
	var wg sync.WaitGroup
	wg.Add(len(m.watchers))

	for _, watcher := range m.watchers {
		go func(w watch.Interface) {
			defer wg.Done()
			events := w.ResultChan()
			for event := range events {
				m.ch <- event
			}
		}(watcher)
	}
	go func() {
		wg.Wait()
		// when all watcher closed,send a signal to stop chan
		m.stop <- struct{}{}
	}()
	return m
}

func (m *multiWatcher) Stop() {
	m.Lock()
	defer m.Unlock()

	if m.stopped {
		return
	}

	m.stopped = true
	for _, watcher := range m.watchers {
		watcher.Stop()
	}

	// waiting for all watcher closed
	<-m.stop
	close(m.ch)
}

func (m *multiWatcher) ResultChan() <-chan watch.Event {
	return m.ch
}
