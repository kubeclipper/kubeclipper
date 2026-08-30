/*
 *
 *  * Copyright 2026 KubeClipper Authors.
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

package operationv2

import (
	"context"
	"testing"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

// recordingStorage records which of the store's call paths reach it. Methods
// that are not overridden panic via the embedded nil interface, which also
// documents that the store must only call the interfaces it declares.
type recordingStorage struct {
	rest.StandardStorage
	listCalls int
	getCalls  int
	listObj   runtime.Object
	getObj    runtime.Object
}

func (r *recordingStorage) List(_ context.Context, _ *metainternalversion.ListOptions) (runtime.Object, error) {
	r.listCalls++
	if r.listObj == nil {
		panic("recordingStorage requires listObj")
	}
	return r.listObj, nil
}

func (r *recordingStorage) Get(_ context.Context, _ string, _ *metav1.GetOptions) (runtime.Object, error) {
	r.getCalls++
	if r.getObj == nil {
		panic("recordingStorage requires getObj")
	}
	return r.getObj, nil
}

// Safety-boundary reads (target ordering, attempt creation, lock ownership
// checks) must be served by the quorum storage; the cacher-backed storages
// are reserved for routine reads and public routes.
func TestSafetyBoundaryReadsUseStrongStorage(t *testing.T) {
	ops := &recordingStorage{listObj: &operations.OperationList{}, getObj: &operations.Operation{}}
	tasks := &recordingStorage{listObj: &operations.OperationTaskList{}}
	locks := &recordingStorage{getObj: &operations.ExecutionLock{}}
	opsStrong := &recordingStorage{listObj: &operations.OperationList{}}
	tasksStrong := &recordingStorage{listObj: &operations.OperationTaskList{}}
	locksStrong := &recordingStorage{getObj: &operations.ExecutionLock{}}

	s, err := NewStore(StoreOptions{
		Operations:       ops,
		Tasks:            tasks,
		Locks:            locks,
		OperationsStrong: opsStrong,
		TasksStrong:      tasksStrong,
		LocksStrong:      locksStrong,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, err := s.ListOperations(ctx, "target-uid", ""); return err },
		func() error { _, err := s.ListTasksByOperationUID(ctx, "op-uid", ""); return err },
		func() error { _, err := s.ListTasksByNode(ctx, "node-1", ""); return err },
		func() error { _, err := s.GetLock(ctx, "lock-1", ""); return err },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	if ops.listCalls != 0 || tasks.listCalls != 0 || locks.getCalls != 0 {
		t.Fatalf("safety-boundary reads reached cacher-backed storages: ops=%d tasks=%d locks=%d", ops.listCalls, tasks.listCalls, locks.getCalls)
	}
	if opsStrong.listCalls != 1 || tasksStrong.listCalls != 2 || locksStrong.getCalls != 1 {
		t.Fatalf("strong storage calls = ops:%d tasks:%d locks:%d, want 1, 2 and 1", opsStrong.listCalls, tasksStrong.listCalls, locksStrong.getCalls)
	}
}

// Routine reads keep the cacher-backed storages so agent watch traffic stays
// off etcd.
func TestRoutineReadsUseCacherBackedStorage(t *testing.T) {
	ops := &recordingStorage{listObj: &operations.OperationList{}, getObj: &operations.Operation{}}
	tasks := &recordingStorage{listObj: &operations.OperationTaskList{}}
	s, err := NewStore(StoreOptions{
		Operations: ops,
		Tasks:      tasks,
		Locks:      &recordingStorage{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, call := range []func() error{
		func() error { _, err := s.GetOperation(ctx, "op-1", ""); return err },
		func() error { _, err := s.ListOperationsWithOptions(ctx, &metav1.ListOptions{}); return err },
		func() error { _, err := s.ListTasksWithOptions(ctx, "node-1", &metav1.ListOptions{}); return err },
	} {
		if err := call(); err != nil {
			t.Fatal(err)
		}
	}
	if ops.getCalls != 1 || ops.listCalls != 1 || tasks.listCalls != 1 {
		t.Fatalf("cacher-backed storage calls = get:%d ops-list:%d tasks-list:%d", ops.getCalls, ops.listCalls, tasks.listCalls)
	}
}

// StoreOptions without dedicated strong storages must keep working (test
// fakes and legacy wiring) by falling back to the injected storages.
func TestStrongStoragesFallBackToInjected(t *testing.T) {
	ops := &recordingStorage{listObj: &operations.OperationList{}}
	tasks := &recordingStorage{listObj: &operations.OperationTaskList{}}
	locks := &recordingStorage{getObj: &operations.ExecutionLock{}}
	s, err := NewStore(StoreOptions{Operations: ops, Tasks: tasks, Locks: locks})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListOperations(context.Background(), "target-uid", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ListTasksByOperationUID(context.Background(), "op-uid", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetLock(context.Background(), "lock-1", ""); err != nil {
		t.Fatal(err)
	}
	if ops.listCalls != 1 || tasks.listCalls != 1 || locks.getCalls != 1 {
		t.Fatalf("fallback storage calls = ops:%d tasks:%d locks:%d, want 1, 1 and 1", ops.listCalls, tasks.listCalls, locks.getCalls)
	}
}
