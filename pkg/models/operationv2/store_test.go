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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	opsStrong := &recordingStorage{listObj: &operations.OperationList{}, getObj: &operations.Operation{}}
	tasksStrong := &recordingStorage{listObj: &operations.OperationTaskList{}, getObj: &operations.OperationTask{}}
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
	concreteStore, ok := s.(*store)
	if !ok {
		t.Fatalf("NewStore() returned %T, want *store", s)
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
	operation, err := concreteStore.getOperationStrong(ctx, "op-1")
	if err != nil {
		t.Fatal(err)
	}
	if operation == nil {
		t.Fatal("strong operation storage returned nil")
	}
	task, err := concreteStore.getTaskStrong(ctx, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if task == nil {
		t.Fatal("strong task storage returned nil")
	}
	if ops.listCalls != 0 || ops.getCalls != 0 || tasks.listCalls != 0 || tasks.getCalls != 0 || locks.getCalls != 0 {
		t.Fatalf("safety-boundary reads reached cacher-backed storages: ops=list:%d get:%d tasks=list:%d get:%d locks=get:%d", ops.listCalls, ops.getCalls, tasks.listCalls, tasks.getCalls, locks.getCalls)
	}
	if opsStrong.listCalls != 1 || opsStrong.getCalls != 1 || tasksStrong.listCalls != 2 || tasksStrong.getCalls != 1 || locksStrong.getCalls != 1 {
		t.Fatalf("strong storage calls = ops:list:%d get:%d tasks:list:%d get:%d locks:get:%d, want 1, 1, 2, 1 and 1", opsStrong.listCalls, opsStrong.getCalls, tasksStrong.listCalls, tasksStrong.getCalls, locksStrong.getCalls)
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

// recordingDeletes records which objects the store deletes and serves
// configurable list/get views so the quorum re-check can be exercised.
type recordingDeletes struct {
	rest.StandardStorage
	listObj  runtime.Object
	getObj   runtime.Object
	deleted  []string
	notFound map[string]bool
}

func (r *recordingDeletes) List(_ context.Context, _ *metainternalversion.ListOptions) (runtime.Object, error) {
	if r.listObj == nil {
		panic("recordingDeletes requires listObj")
	}
	return r.listObj, nil
}

func (r *recordingDeletes) Get(_ context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	if r.notFound[name] {
		return nil, apierrors.NewNotFound(schema.GroupResource{}, name)
	}
	if r.getObj == nil {
		panic("recordingDeletes requires getObj")
	}
	return r.getObj, nil
}

func (r *recordingDeletes) Delete(
	_ context.Context, name string, _ rest.ValidateObjectFunc, _ *metav1.DeleteOptions,
) (runtime.Object, bool, error) {
	r.deleted = append(r.deleted, name)
	return nil, true, nil
}

// CleanupByTargetUID must refuse while any Operation or Task of the target is
// still active, and must purge tasks, operations and a stale execution lock
// once everything reached a terminal phase. The terminal-state gate must use
// the Get path (quorum), not the possibly stale list view.
func newCleanupStore(ops, tasks, locks *recordingDeletes) (Store, error) {
	return NewStore(StoreOptions{Operations: ops, Tasks: tasks, Locks: locks})
}

func newCleanupOp(phase operations.OperationPhase) *operations.Operation {
	return &operations.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "op-1", UID: "op-uid-1"},
		Spec:       operations.OperationSpec{TargetRef: operations.ObjectReference{Kind: "Cluster", UID: "target-uid", Name: "cluster"}},
		Status:     operations.OperationStatus{Phase: phase},
	}
}

func newCleanupTask(phase operations.TaskPhase) *operations.OperationTask {
	return &operations.OperationTask{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", UID: "task-uid-1"},
		Spec:       operations.OperationTaskSpec{OperationRef: operations.ObjectReference{UID: "op-uid-1"}},
		Status:     operations.OperationTaskStatus{Phase: phase},
	}
}

func TestCleanupByTargetUIDRefusesNonTerminalOperation(t *testing.T) {
	ops := &recordingDeletes{
		listObj: &operations.OperationList{Items: []operations.Operation{
			*newCleanupOp(operations.OperationRunning),
		}},
		getObj: newCleanupOp(operations.OperationRunning),
	}
	s, err := newCleanupStore(ops, &recordingDeletes{listObj: &operations.OperationTaskList{}}, &recordingDeletes{listObj: &operations.ExecutionLockList{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CleanupByTargetUID(context.Background(), "target-uid"); err == nil {
		t.Fatal("cleanup must refuse while an operation is non-terminal")
	}
	if len(ops.deleted) != 0 {
		t.Fatalf("refused cleanup still deleted %v", ops.deleted)
	}
}

func TestCleanupByTargetUIDRefusesActiveTaskOnQuorumGet(t *testing.T) {
	ops := &recordingDeletes{
		listObj: &operations.OperationList{Items: []operations.Operation{
			*newCleanupOp(operations.OperationSucceeded),
		}},
		getObj: newCleanupOp(operations.OperationSucceeded),
	}
	tasks := &recordingDeletes{
		listObj: &operations.OperationTaskList{Items: []operations.OperationTask{
			*newCleanupTask(operations.TaskSucceeded),
		}},
		getObj: newCleanupTask(operations.TaskRunning),
	}
	s, err := newCleanupStore(ops, tasks, &recordingDeletes{listObj: &operations.ExecutionLockList{}})
	if err != nil {
		t.Fatal(err)
	}
	// The list view reports the task as Succeeded while the quorum Get still
	// sees Running: the stale list must not enable the delete.
	if err := s.CleanupByTargetUID(context.Background(), "target-uid"); err == nil {
		t.Fatal("cleanup must refuse on the quorum view of an active task")
	}
	if len(tasks.deleted) != 0 {
		t.Fatalf("refused cleanup still deleted %v", tasks.deleted)
	}
}

func TestCleanupByTargetUIDPurgesTerminalHistoryIdempotently(t *testing.T) {
	ops := &recordingDeletes{
		listObj: &operations.OperationList{Items: []operations.Operation{
			*newCleanupOp(operations.OperationSucceeded),
		}},
		getObj: newCleanupOp(operations.OperationSucceeded),
	}
	tasks := &recordingDeletes{
		listObj: &operations.OperationTaskList{Items: []operations.OperationTask{
			*newCleanupTask(operations.TaskSucceeded),
		}},
		getObj: newCleanupTask(operations.TaskSucceeded),
	}
	locks := &recordingDeletes{
		listObj: &operations.ExecutionLockList{Items: []operations.ExecutionLock{{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster-stale-lock"},
			Spec:       operations.ExecutionLockSpec{TargetRef: operations.ObjectReference{Kind: "Cluster", UID: "target-uid"}},
		}}},
	}
	s, err := newCleanupStore(ops, tasks, locks)
	if err != nil {
		t.Fatal(err)
	}
	for round := range 2 {
		if err := s.CleanupByTargetUID(context.Background(), "target-uid"); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
	}
	if len(tasks.deleted) != 2 || tasks.deleted[0] != "task-1" {
		t.Fatalf("task deletes = %v", tasks.deleted)
	}
	if len(ops.deleted) != 2 || ops.deleted[0] != "op-1" {
		t.Fatalf("operation deletes = %v", ops.deleted)
	}
	if len(locks.deleted) != 2 || locks.deleted[0] != "cluster-stale-lock" {
		t.Fatalf("lock deletes = %v", locks.deleted)
	}
}
