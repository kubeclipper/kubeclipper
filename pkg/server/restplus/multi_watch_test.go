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
	"strconv"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type testType string

func (obj testType) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (obj testType) DeepCopyObject() runtime.Object   { return obj }

func TestNewMultiWatcher(t *testing.T) {
	w1 := watch.NewFake()
	w2 := watch.NewFake()
	m := NewMultiWatcher([]watch.Interface{w1, w2})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			w1.Add(testType("watcher1" + strconv.Itoa(i)))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			w2.Add(testType("watcher2" + strconv.Itoa(i)))
		}
	}()

	go func() {
		wg.Wait()
		m.Stop()
	}()

	resultChan := m.ResultChan()
	for event := range resultChan {
		t.Log("event:", event)
	}
}
