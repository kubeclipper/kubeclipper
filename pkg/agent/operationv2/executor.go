/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package operationv2

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

// Executor reconciles one immutable Task spec. Implementations must be safe to
// call again for the same Task UID after an agent or API failure.
type Executor interface {
	Reconcile(ctx context.Context, task *operations.OperationTask, log io.Writer) (operations.TaskResult, error)
}

type Registry struct {
	mu        sync.RWMutex
	executors map[string]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]Executor)}
}

func (r *Registry) Register(name string, executor Executor) error {
	if name == "" {
		return fmt.Errorf("executor name is required")
	}
	if executor == nil {
		return fmt.Errorf("executor %q is nil", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.executors[name]; exists {
		return fmt.Errorf("executor %q is already registered", name)
	}
	r.executors[name] = executor
	return nil
}

func (r *Registry) Get(name string) (Executor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	executor, ok := r.executors[name]
	return executor, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.executors))
	for name := range r.executors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
