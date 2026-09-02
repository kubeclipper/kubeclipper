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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

var testNow = time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)

func TestReconcileSerializesOperationsByTarget(t *testing.T) {
	store := newFakeStore()
	first := testOperation("first", "op-1", testNow, oneStep("step", 0, "node-1"))
	second := testOperation("second", "op-2", testNow.Add(time.Second), oneStep("step", 0, "node-1"))
	store.addOperation(first)
	store.addOperation(second)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow.Add(time.Minute) }}

	result, err := r.Reconcile(context.Background(), requestFor(second.Name))
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("later operation did not wait")
	}
	if len(store.locks) != 0 || len(store.tasks) != 0 {
		t.Fatalf("later operation acquired execution facts: locks=%d tasks=%d", len(store.locks), len(store.tasks))
	}
	if got := store.operation(second.Name).Status.Phase; got != operations.OperationPending {
		t.Fatalf("later operation phase = %s, want Pending", got)
	}
}

func TestSameSecondOperationsUseEtcdRevisionOrder(t *testing.T) {
	store := newFakeStore()
	// A later object's UID sorts first lexically. The etcd revision must still
	// keep the actual create order stable.
	first := testOperation("first", "z-op", testNow, oneStep("step", 0, "node-1"))
	second := testOperation("second", "a-op", testNow, oneStep("step", 0, "node-1"))
	store.addOperation(first)
	store.addOperation(second)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow.Add(time.Minute) }}

	result, err := r.Reconcile(context.Background(), requestFor(second.Name))
	if err != nil {
		t.Fatal(err)
	}
	if result.RequeueAfter == 0 || len(store.locks) != 0 {
		t.Fatal("later same-second operation bypassed its predecessor")
	}
}

func TestReconcileStepBarrierAndOutput(t *testing.T) {
	store := newFakeStore()
	op := testOperation("install", "op-1", testNow,
		oneStep("token", 0, "node-1"),
		operations.OperationStep{
			ID:       "join",
			Executor: "noop",
			Targets:  []operations.NodeReference{{Name: "node-2", UID: "node-2"}},
			Payload:  runtime.RawExtension{Raw: []byte(`{"role":"worker"}`)},
			Inputs: []operations.StepInput{{
				Field: "token", FromStepID: "token", FromNodeUID: "node-1", OutputKey: "joinToken",
			}},
		},
	)
	store.addOperation(op)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow }}

	reconcileOK(t, r, op.Name) // Lock and Running.
	reconcileOK(t, r, op.Name) // First-step Task.
	first := store.onlyTask(t)
	if first.Spec.StepID != "token" {
		t.Fatalf("created step %q before token", first.Spec.StepID)
	}
	store.setTaskTerminal(first.Name, operations.TaskSucceeded, map[string]string{"joinToken": "abcdef.0123456789"})
	reconcileOK(t, r, op.Name)

	tasks := store.taskSlice()
	if len(tasks) != 2 {
		t.Fatalf("tasks=%d, want 2", len(tasks))
	}
	var join *operations.OperationTask
	for index := range tasks {
		if tasks[index].Spec.StepID == "join" {
			join = &tasks[index]
		}
	}
	if join == nil {
		t.Fatal("second-step task was not created")
	}
	var payload map[string]string
	if err := json.Unmarshal(join.Spec.Payload.Raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["token"] != "abcdef.0123456789" || payload["role"] != "worker" {
		t.Fatalf("materialized payload = %#v", payload)
	}

	store.setTaskTerminal(join.Name, operations.TaskSucceeded, nil)
	reconcileOK(t, r, op.Name)
	finished := store.operation(op.Name)
	if finished.Status.Phase != operations.OperationSucceeded {
		t.Fatalf("phase=%s, want Succeeded", finished.Status.Phase)
	}
	if len(store.locks) != 0 {
		t.Fatal("terminal operation did not release its lock")
	}
}

func TestMissingStepOutputFailsWithoutCreatingTask(t *testing.T) {
	store := newFakeStore()
	firstStep := oneStep("token", 0, "node-1")
	secondStep := operations.OperationStep{
		ID: "join", Executor: "noop",
		Targets: []operations.NodeReference{{Name: "node-2", UID: "node-2"}},
		Payload: runtime.RawExtension{Raw: []byte(`{}`)},
		Inputs:  []operations.StepInput{{Field: "token", FromStepID: "token", FromNodeUID: "node-1", OutputKey: "missing"}},
	}
	op := runningOperation("install", firstStep, secondStep)
	store.addOperation(op)
	store.addOwnedLock(op)
	completed := testTask(op, &firstStep, firstStep.Targets[0], operations.TaskSucceeded)
	completed.Status.Result = &operations.TaskResult{Outputs: map[string]string{"another": "value"}}
	store.addTask(completed)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow }}

	reconcileOK(t, r, op.Name)
	got := store.operation(op.Name)
	if got.Status.Phase != operations.OperationFailed || got.Status.Reason != operations.OperationReasonInvalidExecutionFacts {
		t.Fatalf("status=%#v, want Failed/InvalidExecutionFacts", got.Status)
	}
	if len(store.tasks) != 1 {
		t.Fatal("task was created with an unresolved input")
	}
}

func TestAutomaticRetryCreatesNewTaskAttempt(t *testing.T) {
	store := newFakeStore()
	step := oneStep("install", 1, "node-1")
	op := runningOperation("install", step)
	store.addOperation(op)
	store.addOwnedLock(op)
	failed := testTask(op, &step, step.Targets[0], operations.TaskFailed)
	started := metav1.NewTime(testNow)
	failed.Status.StartedAt = &started
	store.addTask(failed)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow }}

	reconcileOK(t, r, op.Name)
	tasks := store.taskSlice()
	if len(tasks) != 2 {
		t.Fatalf("tasks=%d, want replacement attempt", len(tasks))
	}
	latest := tasks[0]
	if tasks[1].Spec.Attempt > latest.Spec.Attempt {
		latest = tasks[1]
	}
	if latest.Spec.Attempt != 1 || latest.Status.Phase != operations.TaskPending || latest.UID == failed.UID {
		t.Fatalf("replacement task = %#v", latest)
	}

	store.setTaskTerminal(latest.Name, operations.TaskFailed, nil)
	reconcileOK(t, r, op.Name)
	if got := store.operation(op.Name).Status.Phase; got != operations.OperationFailed {
		t.Fatalf("phase=%s, want Failed after retry exhaustion", got)
	}
}

func TestCancellationDoesNotStopRunningTask(t *testing.T) {
	store := newFakeStore()
	step := oneStep("install", 0, "node-1", "node-2")
	op := runningOperation("install", step)
	op.Spec.DesiredState = operations.OperationDesiredStateCancelled
	store.addOperation(op)
	store.addOwnedLock(op)
	running := testTask(op, &step, step.Targets[0], operations.TaskRunning)
	pending := testTask(op, &step, step.Targets[1], operations.TaskPending)
	store.addTask(running)
	store.addTask(pending)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow }}

	reconcileOK(t, r, op.Name)
	if got := store.task(running.Name).Status.Phase; got != operations.TaskRunning {
		t.Fatalf("running task was changed to %s", got)
	}
	if got := store.task(pending.Name).Status.Phase; got != operations.TaskCancelled {
		t.Fatalf("pending sibling phase=%s, want Canceled", got)
	}
	store.setTaskTerminal(running.Name, operations.TaskSucceeded, nil)
	reconcileOK(t, r, op.Name)
	if got := store.operation(op.Name).Status.Phase; got != operations.OperationCancelled {
		t.Fatalf("operation phase=%s, want Canceled", got)
	}
}

func TestDeadlineTimesOutRunningTaskAfterGrace(t *testing.T) {
	store := newFakeStore()
	step := oneStep("restore", 0, "node-1")
	op := runningOperation("restore", step)
	deadline := metav1.NewTime(testNow)
	op.Status.Deadline = &deadline
	store.addOperation(op)
	store.addOwnedLock(op)
	task := testTask(op, &step, step.Targets[0], operations.TaskRunning)
	store.addTask(task)
	now := testNow.Add(operations.ServerTerminationGrace - time.Second)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return now }}

	reconcileOK(t, r, op.Name)
	if got := store.task(task.Name).Status.Phase; got != operations.TaskRunning {
		t.Fatalf("task timed out before grace: %s", got)
	}
	now = testNow.Add(operations.ServerTerminationGrace)
	reconcileOK(t, r, op.Name)
	if got := store.task(task.Name).Status.Phase; got != operations.TaskTimedOut {
		t.Fatalf("task phase=%s, want TimedOut", got)
	}
	reconcileOK(t, r, op.Name)
	if got := store.operation(op.Name).Status.Phase; got != operations.OperationTimedOut {
		t.Fatalf("operation phase=%s, want TimedOut", got)
	}
}

func TestHumanRetryOnlyRunsUnsuccessfulNode(t *testing.T) {
	store := newFakeStore()
	step := oneStep("join", 0, "node-1", "node-2")
	op := runningOperation("join", step)
	op.Status.Phase = operations.OperationFailed
	op.Status.ObservedRetryGeneration = 0
	op.Spec.RetryGeneration = 1
	store.addOperation(op)
	succeeded := testTask(op, &step, step.Targets[0], operations.TaskSucceeded)
	failed := testTask(op, &step, step.Targets[1], operations.TaskFailed)
	store.addTask(succeeded)
	store.addTask(failed)
	r := &OperationReconciler{Store: store, Now: func() time.Time { return testNow.Add(time.Minute) }}

	reconcileOK(t, r, op.Name) // Reopen as Pending.
	if reopened := store.operation(op.Name); reopened.Status.Phase != operations.OperationPending ||
		reopened.Status.ObservedRetryGeneration != 1 {
		t.Fatalf("retry reopen status=%#v, want Pending generation 1", reopened.Status)
	}
	reconcileOK(t, r, op.Name) // Start the new generation and deadline.
	reconcileOK(t, r, op.Name) // Create retry task.
	tasks := store.taskSlice()
	if len(tasks) != 3 {
		t.Fatalf("tasks=%d, want one new attempt", len(tasks))
	}
	var retried *operations.OperationTask
	for index := range tasks {
		if tasks[index].Spec.RetryGeneration == 1 {
			retried = &tasks[index]
		}
	}
	if retried == nil || retried.Spec.NodeRef.UID != "node-2" || retried.Spec.Attempt != 1 {
		t.Fatalf("unexpected retry task: %#v", retried)
	}
}

func TestTaskAndLockNamesAreStableAndDistinct(t *testing.T) {
	first := TaskName("op", 0, "step", "node", 0)
	if first != TaskName("op", 0, "step", "node", 0) {
		t.Fatal("task name is not stable")
	}
	if first == TaskName("op", 0, "step", "node", 1) {
		t.Fatal("attempt is not represented in task name")
	}
	if LockName("Cluster", "cluster-a") == LockName("Cluster", "cluster-b") {
		t.Fatal("different targets share a lock name")
	}
}

func reconcileOK(t *testing.T, reconciler *OperationReconciler, name string) reconcile.Result {
	t.Helper()
	result, err := reconciler.Reconcile(context.Background(), requestFor(name))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requestFor(name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: name}}
}

func oneStep(id string, retryLimit int32, nodes ...string) operations.OperationStep {
	targets := make([]operations.NodeReference, 0, len(nodes))
	for _, node := range nodes {
		targets = append(targets, operations.NodeReference{Name: node, UID: types.UID(node)})
	}
	return operations.OperationStep{
		ID:         id,
		Executor:   "noop",
		RetryLimit: retryLimit,
		Targets:    targets,
		Payload:    runtime.RawExtension{Raw: []byte(`{}`)},
	}
}

func testOperation(name string, uid types.UID, created time.Time, steps ...operations.OperationStep) *operations.Operation {
	return &operations.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: uid, CreationTimestamp: metav1.NewTime(created)},
		Spec: operations.OperationSpec{
			TargetRef:    operations.ObjectReference{Kind: "Cluster", Name: "cluster", UID: "cluster-1"},
			Action:       "Install",
			DesiredState: operations.OperationDesiredStateActive,
			Timeout:      metav1.Duration{Duration: time.Hour},
			Steps:        steps,
		},
		Status: operations.OperationStatus{Phase: operations.OperationPending},
	}
}

func runningOperation(name string, steps ...operations.OperationStep) *operations.Operation {
	op := testOperation(name, "op-1", testNow, steps...)
	deadline := metav1.NewTime(testNow.Add(time.Hour))
	started := metav1.NewTime(testNow.Add(-time.Minute))
	op.Status = operations.OperationStatus{Phase: operations.OperationRunning, Deadline: &deadline, StartedAt: &started}
	return op
}

func testTask(
	op *operations.Operation,
	step *operations.OperationStep,
	node operations.NodeReference,
	phase operations.TaskPhase,
) *operations.OperationTask {
	const generation int64 = 0
	const attempt int32 = 0
	deadline := op.Status.Deadline
	if deadline == nil {
		value := metav1.NewTime(testNow.Add(time.Hour))
		deadline = &value
	}
	return &operations.OperationTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:              TaskName(op.UID, generation, step.ID, node.UID, attempt),
			UID:               types.UID(fmt.Sprintf("task-%s-%d", node.UID, attempt)),
			CreationTimestamp: metav1.NewTime(testNow.Add(time.Duration(attempt) * time.Second)),
		},
		Spec: operations.OperationTaskSpec{
			OperationRef: operations.ObjectReference{Kind: operations.KindOperation, Name: op.Name, UID: op.UID},
			StepID:       step.ID, NodeRef: node, RetryGeneration: generation, Attempt: attempt,
			Executor: step.Executor, Payload: step.Payload, Deadline: *deadline,
		},
		Status: operations.OperationTaskStatus{Phase: phase},
	}
}

type fakeStore struct {
	operations map[string]*operations.Operation
	tasks      map[string]*operations.OperationTask
	locks      map[string]*operations.ExecutionLock
	nextRV     int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		operations: map[string]*operations.Operation{},
		tasks:      map[string]*operations.OperationTask{},
		locks:      map[string]*operations.ExecutionLock{},
		nextRV:     1,
	}
}

func (f *fakeStore) rv() string {
	value := strconv.FormatInt(f.nextRV, 10)
	f.nextRV++
	return value
}

func (f *fakeStore) addOperation(op *operations.Operation) {
	operationCopy := op.DeepCopy()
	operationCopy.ResourceVersion = f.rv()
	f.operations[operationCopy.Name] = operationCopy
}

func (f *fakeStore) addTask(task *operations.OperationTask) {
	taskCopy := task.DeepCopy()
	taskCopy.ResourceVersion = f.rv()
	if taskCopy.UID == "" {
		taskCopy.UID = types.UID(taskCopy.Name + "-uid")
	}
	f.tasks[taskCopy.Name] = taskCopy
}

func (f *fakeStore) addOwnedLock(op *operations.Operation) {
	name := LockName(op.Spec.TargetRef.Kind, op.Spec.TargetRef.UID)
	f.locks[name] = &operations.ExecutionLock{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid"), ResourceVersion: f.rv()},
		Spec: operations.ExecutionLockSpec{
			TargetRef: op.Spec.TargetRef,
			HolderRef: operations.ObjectReference{Kind: operations.KindOperation, Name: op.Name, UID: op.UID},
		},
	}
}

func (f *fakeStore) GetOperation(_ context.Context, name, _ string) (*operations.Operation, error) {
	op, exists := f.operations[name]
	if !exists {
		return nil, apierrors.NewNotFound(operations.Resource(operations.ResourceOperations), name)
	}
	return op.DeepCopy(), nil
}

func (f *fakeStore) ListOperations(_ context.Context, targetUID types.UID, _ string) (*operations.OperationList, error) {
	result := &operations.OperationList{}
	for _, op := range f.operations {
		if targetUID == "" || op.Spec.TargetRef.UID == targetUID {
			result.Items = append(result.Items, *op.DeepCopy())
		}
	}
	sort.Slice(result.Items, func(i, j int) bool { return operationBefore(&result.Items[i], &result.Items[j]) })
	return result, nil
}

func (f *fakeStore) UpdateOperationStatus(
	_ context.Context,
	name string,
	uid types.UID,
	rv string,
	status *operations.OperationStatus,
) (*operations.Operation, error) {
	op, exists := f.operations[name]
	if !exists {
		return nil, apierrors.NewNotFound(operations.Resource(operations.ResourceOperations), name)
	}
	if op.UID != uid || op.ResourceVersion != rv {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceOperations), name, fmt.Errorf("stale object"))
	}
	op.Status = *status.DeepCopy()
	op.ResourceVersion = f.rv()
	return op.DeepCopy(), nil
}

func (f *fakeStore) GetTask(_ context.Context, name, _ string) (*operations.OperationTask, error) {
	task, exists := f.tasks[name]
	if !exists {
		return nil, apierrors.NewNotFound(operations.Resource(operations.ResourceTasks), name)
	}
	return task.DeepCopy(), nil
}

func (f *fakeStore) CreateTask(_ context.Context, task *operations.OperationTask) (*operations.OperationTask, error) {
	if _, exists := f.tasks[task.Name]; exists {
		return nil, apierrors.NewAlreadyExists(operations.Resource(operations.ResourceTasks), task.Name)
	}
	taskCopy := task.DeepCopy()
	taskCopy.UID = types.UID(taskCopy.Name + "-uid")
	taskCopy.ResourceVersion = f.rv()
	taskCopy.CreationTimestamp = metav1.NewTime(testNow.Add(time.Duration(f.nextRV) * time.Second))
	taskCopy.Status.Phase = operations.TaskPending
	f.tasks[taskCopy.Name] = taskCopy
	return taskCopy.DeepCopy(), nil
}

func (f *fakeStore) ListTasksByOperationUID(_ context.Context, operationUID types.UID, _ string) (*operations.OperationTaskList, error) {
	result := &operations.OperationTaskList{}
	for _, task := range f.tasks {
		if task.Spec.OperationRef.UID == operationUID {
			result.Items = append(result.Items, *task.DeepCopy())
		}
	}
	return result, nil
}

func (f *fakeStore) CancelPendingTask(
	_ context.Context,
	name string,
	uid types.UID,
	rv string,
	reason operations.TaskResultReason,
) (*operations.OperationTask, error) {
	task := f.tasks[name]
	if task == nil || task.UID != uid || task.ResourceVersion != rv || task.Status.Phase != operations.TaskPending {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("task changed"))
	}
	task.Status.Phase = operations.TaskCancelled
	task.Status.Result = &operations.TaskResult{Reason: reason}
	task.ResourceVersion = f.rv()
	return task.DeepCopy(), nil
}

func (f *fakeStore) TimeoutRunningTask(_ context.Context, name string, uid types.UID, rv string) (*operations.OperationTask, error) {
	task := f.tasks[name]
	if task == nil || task.UID != uid || task.ResourceVersion != rv || task.Status.Phase != operations.TaskRunning {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("task changed"))
	}
	task.Status.Phase = operations.TaskTimedOut
	task.Status.Result = &operations.TaskResult{Reason: operations.TaskReasonDeadlineExceeded}
	task.ResourceVersion = f.rv()
	return task.DeepCopy(), nil
}

func (f *fakeStore) AcquireLock(_ context.Context, lock *operations.ExecutionLock) (*operations.ExecutionLock, bool, error) {
	if existing, exists := f.locks[lock.Name]; exists {
		return existing.DeepCopy(), false, nil
	}
	lockCopy := lock.DeepCopy()
	lockCopy.UID = types.UID(lockCopy.Name + "-uid")
	lockCopy.ResourceVersion = f.rv()
	f.locks[lockCopy.Name] = lockCopy
	return lockCopy.DeepCopy(), true, nil
}

func (f *fakeStore) GetLock(_ context.Context, name, _ string) (*operations.ExecutionLock, error) {
	lock, exists := f.locks[name]
	if !exists {
		return nil, apierrors.NewNotFound(operations.Resource(operations.ResourceLocks), name)
	}
	return lock.DeepCopy(), nil
}

func (f *fakeStore) ReleaseLock(_ context.Context, name string, lockUID, holderUID types.UID) error {
	lock, exists := f.locks[name]
	if !exists {
		return nil
	}
	if lock.UID != lockUID || lock.Spec.HolderRef.UID != holderUID {
		return apierrors.NewConflict(operations.Resource(operations.ResourceLocks), name, fmt.Errorf("lock changed"))
	}
	delete(f.locks, name)
	return nil
}

func (f *fakeStore) operation(name string) *operations.Operation {
	return f.operations[name].DeepCopy()
}
func (f *fakeStore) task(name string) *operations.OperationTask { return f.tasks[name].DeepCopy() }

func (f *fakeStore) taskSlice() []operations.OperationTask {
	result := make([]operations.OperationTask, 0, len(f.tasks))
	for _, task := range f.tasks {
		result = append(result, *task.DeepCopy())
	}
	return result
}

func (f *fakeStore) onlyTask(t *testing.T) *operations.OperationTask {
	t.Helper()
	if len(f.tasks) != 1 {
		t.Fatalf("tasks=%d, want 1", len(f.tasks))
	}
	for _, task := range f.tasks {
		return task.DeepCopy()
	}
	return nil
}

func (f *fakeStore) setTaskTerminal(name string, phase operations.TaskPhase, outputs map[string]string) {
	task := f.tasks[name]
	started := metav1.NewTime(testNow)
	finished := metav1.NewTime(testNow.Add(time.Second))
	task.Status = operations.OperationTaskStatus{
		Phase:      phase,
		StartedAt:  &started,
		FinishedAt: &finished,
		Result:     &operations.TaskResult{Outputs: outputs},
	}
	task.ResourceVersion = f.rv()
}

func (f *fakeStore) CleanupByTargetUID(_ context.Context, targetUID types.UID) error {
	for name, op := range f.operations {
		if op.Spec.TargetRef.UID == targetUID {
			for taskName, task := range f.tasks {
				if task.Spec.OperationRef.UID == op.UID {
					delete(f.tasks, taskName)
				}
			}
			delete(f.operations, name)
		}
	}
	return nil
}

func TestMapTargetOperationsUsesTargetUIDIndex(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		targetUIDIndex: func(raw any) ([]string, error) {
			op, ok := raw.(*operations.Operation)
			if !ok || op.Spec.TargetRef.UID == "" {
				return nil, nil
			}
			return []string{string(op.Spec.TargetRef.UID)}, nil
		},
	})
	newOp := func(name, uid string) *operations.Operation {
		return &operations.Operation{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       operations.OperationSpec{TargetRef: operations.ObjectReference{Kind: "Cluster", UID: types.UID(uid), Name: "cluster"}},
		}
	}
	self := newOp("op-self", "uid-1")
	sibling := newOp("op-sibling", "uid-1")
	other := newOp("op-other", "uid-2")
	for _, op := range []*operations.Operation{self, sibling, other} {
		if err := indexer.Add(op); err != nil {
			t.Fatal(err)
		}
	}
	requests := mapTargetOperations(indexer)(self)
	names := make([]string, 0, len(requests))
	for _, req := range requests {
		names = append(names, req.Name)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "op-self" || names[1] != "op-sibling" {
		t.Fatalf("mapped requests = %v, want the operation itself and its same-target sibling only", names)
	}
}
