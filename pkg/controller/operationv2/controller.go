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
	"errors"
	"fmt"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

const (
	defaultWaitRequeue = 5 * time.Second
	conflictRequeue    = 200 * time.Millisecond
)

// OperationReconciler is deliberately level driven. Every call derives the
// next action from durable Operation, Task, and Lock objects and performs at
// most one class of mutation before returning.
type OperationReconciler struct {
	Store Store
	Now   func() time.Time
}

// Store is the narrow persistence contract needed by the reconciler. The
// concrete operationv2.Store also serves HTTP handlers, but the controller is
// intentionally unaware of those extra methods.
type Store interface {
	GetOperation(context.Context, string, string) (*operations.Operation, error)
	ListOperations(context.Context, types.UID, string) (*operations.OperationList, error)
	UpdateOperationStatus(context.Context, string, types.UID, string, *operations.OperationStatus) (*operations.Operation, error)
	GetTask(context.Context, string, string) (*operations.OperationTask, error)
	CreateTask(context.Context, *operations.OperationTask) (*operations.OperationTask, error)
	ListTasksByOperationUID(context.Context, types.UID, string) (*operations.OperationTaskList, error)
	CancelPendingTask(context.Context, string, types.UID, string, operations.TaskResultReason) (*operations.OperationTask, error)
	TimeoutRunningTask(context.Context, string, types.UID, string) (*operations.OperationTask, error)
	AcquireLock(context.Context, *operations.ExecutionLock) (*operations.ExecutionLock, bool, error)
	GetLock(context.Context, string, string) (*operations.ExecutionLock, error)
	ReleaseLock(context.Context, string, types.UID, types.UID) error
}

func (r *OperationReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	if r.Store == nil {
		return reconcile.Result{}, fmt.Errorf("operation v2 store is required")
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}

	op, err := r.Store.GetOperation(ctx, req.Name, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	taskList, err := r.Store.ListTasksByOperationUID(ctx, op.UID, "")
	if err != nil {
		return reconcile.Result{}, err
	}

	retryRequested := op.Spec.RetryGeneration > op.Status.ObservedRetryGeneration
	if op.Status.Phase.IsTerminal() && !retryRequested {
		return r.releaseTerminalLock(ctx, op, taskList.Items)
	}

	targetOperations, err := r.Store.ListOperations(ctx, op.Spec.TargetRef.UID, "")
	if err != nil {
		return reconcile.Result{}, err
	}
	if retryRequested && !isLatestOperation(op, targetOperations.Items) {
		return r.skipRetry(ctx, op)
	}
	if !isEarliestRunnable(op, targetOperations.Items) {
		return reconcile.Result{RequeueAfter: defaultWaitRequeue}, nil
	}

	// A never-started canceled operation has no side effects and does not need
	// to acquire the target lock merely to record cancellation.
	if op.Status.Phase == operations.OperationPending &&
		op.Spec.DesiredState == operations.OperationDesiredStateCancelled &&
		len(taskList.Items) == 0 {
		return r.finish(
			ctx,
			op,
			nil,
			operations.OperationCancelled,
			operations.OperationReasonCancelledByRequest,
			"operation canceled before execution",
		)
	}

	lock, owned, err := r.acquireLock(ctx, op)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !owned {
		return reconcile.Result{RequeueAfter: defaultWaitRequeue}, nil
	}
	if retryRequested {
		// The latest-operation check and lock Create affect different etcd
		// keys. Recheck after acquiring the lock so a concurrently persisted
		// newer Operation cannot be missed by a stale first list.
		latest, listErr := r.Store.ListOperations(ctx, op.Spec.TargetRef.UID, "")
		if listErr != nil {
			return reconcile.Result{}, listErr
		}
		if !isLatestOperation(op, latest.Items) {
			if releaseErr := r.Store.ReleaseLock(ctx, lock.Name, lock.UID, op.UID); releaseErr != nil {
				return reconcile.Result{}, releaseErr
			}
			return r.skipRetry(ctx, op)
		}
	}

	if retryRequested {
		// A retry opens a new execution generation without rewriting any old
		// Task. Persist Pending first; the next reconcile acquires/adopts the
		// same lock and starts the new deadline. This also makes the otherwise
		// terminal phase transition explicit and recoverable.
		status := operations.OperationStatus{
			Phase:                   operations.OperationPending,
			ObservedRetryGeneration: op.Spec.RetryGeneration,
		}
		_, err = r.Store.UpdateOperationStatus(ctx, op.Name, op.UID, op.ResourceVersion, &status)
		return resultForConflict(err)
	}
	if op.Status.Phase == operations.OperationPending {
		startedAt := now()
		status := operations.OperationStatus{
			Phase:                   operations.OperationRunning,
			ObservedRetryGeneration: op.Spec.RetryGeneration,
			StartedAt:               timePointer(startedAt),
			Deadline:                timePointer(startedAt.Add(op.Spec.Timeout.Duration)),
		}
		_, err = r.Store.UpdateOperationStatus(ctx, op.Name, op.UID, op.ResourceVersion, &status)
		return resultForConflict(err)
	}
	if op.Status.Phase != operations.OperationRunning {
		return reconcile.Result{}, fmt.Errorf("operation %q has non-runnable phase %q", op.Name, op.Status.Phase)
	}
	if op.Status.Deadline == nil {
		return r.failInvalidFacts(ctx, op, lock, taskList.Items, fmt.Errorf("running operation has no deadline"))
	}

	if deadlineExpired(op, now()) {
		return r.reconcileDeadline(ctx, op, lock, taskList.Items, now())
	}

	facts, complete, factsErr := validateAndCurrentStep(op, taskList.Items)
	if factsErr != nil {
		return r.failInvalidFacts(ctx, op, lock, taskList.Items, factsErr)
	}
	if complete {
		return r.finish(ctx, op, lock, operations.OperationSucceeded, "", "")
	}

	if op.Spec.DesiredState == operations.OperationDesiredStateCancelled {
		return r.reconcileCancellation(ctx, op, lock, taskList.Items)
	}

	if latestFailure(facts) {
		changed, cancelErr := r.cancelPending(ctx, facts.Tasks, operations.TaskReasonSiblingFailed)
		if cancelErr != nil {
			return resultForConflict(cancelErr)
		}
		if changed {
			return reconcile.Result{Requeue: true}, nil
		}
	}

	pending, running := activeTaskPointers(facts.Tasks)
	if len(pending)+len(running) != 0 {
		return reconcile.Result{RequeueAfter: untilDeadline(op, now())}, nil
	}

	created, eligible, err := r.createAttempts(ctx, op, facts, taskList.Items)
	if err != nil {
		var invalidFacts *invalidExecutionFactsError
		if errors.As(err, &invalidFacts) {
			return r.failInvalidFacts(ctx, op, lock, taskList.Items, invalidFacts)
		}
		return reconcile.Result{}, err
	}
	if created {
		return reconcile.Result{Requeue: true}, nil
	}
	if !eligible {
		return r.finish(ctx, op, lock, operations.OperationFailed, operations.OperationReasonStepFailed,
			fmt.Sprintf("step %q exhausted its retry limit", facts.Step.ID))
	}
	return reconcile.Result{RequeueAfter: defaultWaitRequeue}, nil
}

func (r *OperationReconciler) skipRetry(ctx context.Context, op *operations.Operation) (reconcile.Result, error) {
	status := op.Status
	status.ObservedRetryGeneration = op.Spec.RetryGeneration
	_, err := r.Store.UpdateOperationStatus(ctx, op.Name, op.UID, op.ResourceVersion, &status)
	return resultForConflict(err)
}

func (r *OperationReconciler) acquireLock(ctx context.Context, op *operations.Operation) (*operations.ExecutionLock, bool, error) {
	name := LockName(op.Spec.TargetRef.Kind, op.Spec.TargetRef.UID)
	wanted := &operations.ExecutionLock{
		TypeMeta:   metav1.TypeMeta{APIVersion: operations.SchemeGroupVersion.String(), Kind: operations.KindExecutionLock},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: operations.ExecutionLockSpec{
			TargetRef: op.Spec.TargetRef,
			HolderRef: operations.ObjectReference{Kind: operations.KindOperation, Name: op.Name, UID: op.UID},
		},
	}
	lock, created, err := r.Store.AcquireLock(ctx, wanted)
	if err != nil {
		return nil, false, err
	}
	if created || lockBelongsToOperation(lock, op) {
		return lock, true, nil
	}
	return lock, false, nil
}

func lockBelongsToOperation(lock *operations.ExecutionLock, op *operations.Operation) bool {
	return lock.Spec.HolderRef.Name == op.Name && lock.Spec.HolderRef.UID == op.UID &&
		apiequality.Semantic.DeepEqual(lock.Spec.TargetRef, op.Spec.TargetRef)
}

func (r *OperationReconciler) releaseTerminalLock(
	ctx context.Context,
	op *operations.Operation,
	tasks []operations.OperationTask,
) (reconcile.Result, error) {
	pending, running := activeTasks(tasks)
	if len(pending)+len(running) != 0 {
		return reconcile.Result{}, fmt.Errorf("terminal operation %q still has active tasks", op.Name)
	}
	name := LockName(op.Spec.TargetRef.Kind, op.Spec.TargetRef.UID)
	lock, err := r.Store.GetLock(ctx, name, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if lock.Spec.HolderRef.UID != op.UID {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{}, r.Store.ReleaseLock(ctx, lock.Name, lock.UID, op.UID)
}

func (r *OperationReconciler) reconcileCancellation(
	ctx context.Context,
	op *operations.Operation,
	lock *operations.ExecutionLock,
	tasks []operations.OperationTask,
) (reconcile.Result, error) {
	changed, err := r.cancelPending(ctx, taskPointers(tasks), operations.TaskReasonOperationCancelled)
	if err != nil {
		return resultForConflict(err)
	}
	if changed {
		return reconcile.Result{Requeue: true}, nil
	}
	_, running := activeTasks(tasks)
	if len(running) != 0 {
		return reconcile.Result{RequeueAfter: untilDeadline(op, r.now())}, nil
	}
	facts, complete, err := validateAndCurrentStep(op, tasks)
	if err != nil {
		return r.failInvalidFacts(ctx, op, lock, tasks, err)
	}
	_ = facts
	if complete {
		return r.finish(ctx, op, lock, operations.OperationSucceeded, "", "")
	}
	return r.finish(
		ctx,
		op,
		lock,
		operations.OperationCancelled,
		operations.OperationReasonCancelledByRequest,
		"operation canceled; running tasks finished",
	)
}

func (r *OperationReconciler) reconcileDeadline(
	ctx context.Context,
	op *operations.Operation,
	lock *operations.ExecutionLock,
	tasks []operations.OperationTask,
	now time.Time,
) (reconcile.Result, error) {
	changed, err := r.cancelPending(ctx, taskPointers(tasks), operations.TaskReasonOperationDeadlineExceededBeforeStart)
	if err != nil {
		return resultForConflict(err)
	}
	if changed {
		return reconcile.Result{Requeue: true}, nil
	}
	_, running := activeTasks(tasks)
	graceEnd := op.Status.Deadline.Add(operations.ServerTerminationGrace)
	if len(running) != 0 && now.Before(graceEnd) {
		return reconcile.Result{RequeueAfter: boundedWait(graceEnd.Sub(now))}, nil
	}
	if len(running) != 0 {
		for _, task := range running {
			if _, timeoutErr := r.Store.TimeoutRunningTask(ctx, task.Name, task.UID, task.ResourceVersion); timeoutErr != nil {
				return resultForConflict(timeoutErr)
			}
		}
		return reconcile.Result{Requeue: true}, nil
	}
	facts, complete, err := validateAndCurrentStep(op, tasks)
	if err != nil {
		return r.failInvalidFacts(ctx, op, lock, tasks, err)
	}
	_ = facts
	if complete {
		return r.finish(ctx, op, lock, operations.OperationSucceeded, "", "")
	}
	if op.Spec.DesiredState == operations.OperationDesiredStateCancelled {
		return r.finish(ctx, op, lock, operations.OperationCancelled, operations.OperationReasonCancelledByRequest, "operation canceled")
	}
	return r.finish(ctx, op, lock, operations.OperationTimedOut, operations.OperationReasonDeadlineExceeded, "operation deadline exceeded")
}

func (r *OperationReconciler) cancelPending(
	ctx context.Context,
	tasks []*operations.OperationTask,
	reason operations.TaskResultReason,
) (bool, error) {
	changed := false
	for _, task := range tasks {
		if task.Status.Phase != operations.TaskPending {
			continue
		}
		if _, err := r.Store.CancelPendingTask(ctx, task.Name, task.UID, task.ResourceVersion, reason); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

func (r *OperationReconciler) createAttempts(
	ctx context.Context,
	op *operations.Operation,
	facts *stepFacts,
	allTasks []operations.OperationTask,
) (created, allEligible bool, err error) {
	type candidate struct {
		node    operations.NodeReference
		attempt int32
	}
	candidates := make([]candidate, 0, len(facts.Incomplete))
	for _, node := range facts.Incomplete {
		attempt, eligible := nextAttempt(facts.Step, facts.ByNode[node.UID], op.Spec.RetryGeneration)
		if !eligible {
			return false, false, nil
		}
		candidates = append(candidates, candidate{node: node, attempt: attempt})
	}
	payload, err := materializePayload(facts.Step, allTasks)
	if err != nil {
		return false, false, &invalidExecutionFactsError{cause: err}
	}
	if op.Status.Deadline == nil {
		return false, false, &invalidExecutionFactsError{cause: fmt.Errorf("operation %q has no deadline", op.Name)}
	}
	for _, item := range candidates {
		task := &operations.OperationTask{
			TypeMeta:   metav1.TypeMeta{APIVersion: operations.SchemeGroupVersion.String(), Kind: operations.KindOperationTask},
			ObjectMeta: metav1.ObjectMeta{Name: TaskName(op.UID, op.Spec.RetryGeneration, facts.Step.ID, item.node.UID, item.attempt)},
			Spec: operations.OperationTaskSpec{
				OperationRef:    operations.ObjectReference{Kind: operations.KindOperation, Name: op.Name, UID: op.UID},
				StepID:          facts.Step.ID,
				NodeRef:         item.node,
				RetryGeneration: op.Spec.RetryGeneration,
				Attempt:         item.attempt,
				Executor:        facts.Step.Executor,
				Payload:         payload,
				Deadline:        *op.Status.Deadline.DeepCopy(),
			},
		}
		if _, createErr := r.Store.CreateTask(ctx, task); createErr != nil {
			if !apierrors.IsAlreadyExists(createErr) {
				return created, true, createErr
			}
			existing, getErr := r.Store.GetTask(ctx, task.Name, "")
			if getErr != nil {
				return created, true, getErr
			}
			if !taskSpecEqual(&existing.Spec, &task.Spec) {
				return created, true, &invalidExecutionFactsError{
					cause: fmt.Errorf("deterministic task %q already exists with different spec", task.Name),
				}
			}
			continue
		}
		created = true
	}
	return created, true, nil
}

type invalidExecutionFactsError struct {
	cause error
}

func (e *invalidExecutionFactsError) Error() string { return e.cause.Error() }
func (e *invalidExecutionFactsError) Unwrap() error { return e.cause }

func (r *OperationReconciler) failInvalidFacts(
	ctx context.Context,
	op *operations.Operation,
	lock *operations.ExecutionLock,
	tasks []operations.OperationTask,
	cause error,
) (reconcile.Result, error) {
	pending, running := activeTasks(tasks)
	if len(running) != 0 {
		return reconcile.Result{RequeueAfter: defaultWaitRequeue}, fmt.Errorf("invalid execution facts while tasks are running: %w", cause)
	}
	if len(pending) != 0 {
		changed, err := r.cancelPending(ctx, pending, operations.TaskReasonSiblingFailed)
		if err != nil {
			return resultForConflict(err)
		}
		if changed {
			return reconcile.Result{Requeue: true}, nil
		}
	}
	return r.finish(ctx, op, lock, operations.OperationFailed, operations.OperationReasonInvalidExecutionFacts, cause.Error())
}

func (r *OperationReconciler) finish(
	ctx context.Context,
	op *operations.Operation,
	lock *operations.ExecutionLock,
	phase operations.OperationPhase,
	reason operations.OperationReason,
	message string,
) (reconcile.Result, error) {
	// Re-list immediately before the irreversible Operation transition. Active
	// Task status can race a prior list, while terminal Task facts cannot change.
	tasks, err := r.Store.ListTasksByOperationUID(ctx, op.UID, "")
	if err != nil {
		return reconcile.Result{}, err
	}
	pending, running := activeTasks(tasks.Items)
	if len(pending)+len(running) != 0 {
		return reconcile.Result{RequeueAfter: defaultWaitRequeue}, nil
	}
	if phase == operations.OperationSucceeded {
		_, complete, factsErr := validateAndCurrentStep(op, tasks.Items)
		if factsErr != nil {
			return reconcile.Result{}, factsErr
		}
		if !complete {
			return reconcile.Result{}, fmt.Errorf("refusing to mark incomplete operation %q succeeded", op.Name)
		}
	}
	status := op.Status
	status.Phase = phase
	status.Reason = reason
	status.Message = message
	status.ObservedRetryGeneration = op.Spec.RetryGeneration
	status.FinishedAt = timePointer(r.now())
	updated, err := r.Store.UpdateOperationStatus(ctx, op.Name, op.UID, op.ResourceVersion, &status)
	if err != nil {
		return resultForConflict(err)
	}
	if lock != nil {
		if err := r.Store.ReleaseLock(ctx, lock.Name, lock.UID, updated.UID); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *OperationReconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func activeTaskPointers(tasks []*operations.OperationTask) (pending, running []*operations.OperationTask) {
	for _, task := range tasks {
		switch task.Status.Phase {
		case operations.TaskPending:
			pending = append(pending, task)
		case operations.TaskRunning:
			running = append(running, task)
		}
	}
	return pending, running
}

func taskPointers(tasks []operations.OperationTask) []*operations.OperationTask {
	result := make([]*operations.OperationTask, 0, len(tasks))
	for index := range tasks {
		result = append(result, &tasks[index])
	}
	return result
}

func timePointer(value time.Time) *metav1.Time {
	result := metav1.NewTime(value)
	return &result
}

func untilDeadline(op *operations.Operation, now time.Time) time.Duration {
	if op.Status.Deadline == nil {
		return defaultWaitRequeue
	}
	return boundedWait(op.Status.Deadline.Sub(now))
}

func boundedWait(wait time.Duration) time.Duration {
	if wait <= 0 {
		return conflictRequeue
	}
	if wait > defaultWaitRequeue {
		return defaultWaitRequeue
	}
	return wait
}

func resultForConflict(err error) (reconcile.Result, error) {
	if err == nil {
		return reconcile.Result{Requeue: true}, nil
	}
	if apierrors.IsConflict(err) {
		return reconcile.Result{RequeueAfter: conflictRequeue}, nil
	}
	return reconcile.Result{}, err
}
