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

// Package operationv2 is the only persistence boundary used by the v2
// controller and its API subresources.  Keeping CAS and transition checks in
// one place prevents an HTTP handler and a reconcile loop from implementing
// subtly different state machines.
package operationv2

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metainternalvalidation "k8s.io/apimachinery/pkg/apis/meta/internalversion/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	validation "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/validation"
)

type Store interface {
	GetOperation(ctx context.Context, name, resourceVersion string) (*operations.Operation, error)
	CreateOperation(ctx context.Context, op *operations.Operation) (*operations.Operation, error)
	ListOperationsWithOptions(ctx context.Context, options *metav1.ListOptions) (*operations.OperationList, error)
	WatchOperationsWithOptions(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error)
	ListOperations(ctx context.Context, targetUID types.UID, resourceVersion string) (*operations.OperationList, error)
	WatchOperations(ctx context.Context, resourceVersion string) (watch.Interface, error)
	UpdateOperationStatus(
		ctx context.Context,
		name string,
		uid types.UID,
		resourceVersion string,
		status *operations.OperationStatus,
	) (*operations.Operation, error)
	UpdateOperationControl(
		ctx context.Context,
		name string,
		uid types.UID,
		resourceVersion string,
		mutate func(*operations.OperationSpec) error,
	) (*operations.Operation, error)

	GetTask(ctx context.Context, name, resourceVersion string) (*operations.OperationTask, error)
	CreateTask(ctx context.Context, task *operations.OperationTask) (*operations.OperationTask, error)
	ListTasksWithOptions(ctx context.Context, nodeName string, options *metav1.ListOptions) (*operations.OperationTaskList, error)
	WatchTasksWithOptions(ctx context.Context, nodeName string, options *metav1.ListOptions) (watch.Interface, error)
	ListTasksByOperationUID(ctx context.Context, operationUID types.UID, resourceVersion string) (*operations.OperationTaskList, error)
	ListTasksByNode(ctx context.Context, nodeName, resourceVersion string) (*operations.OperationTaskList, error)
	WatchTasks(ctx context.Context, nodeName, resourceVersion string) (watch.Interface, error)
	UpdateTaskStatus(
		ctx context.Context,
		name string,
		uid types.UID,
		resourceVersion string,
		status operations.OperationTaskStatus,
	) (*operations.OperationTask, error)
	CancelPendingTask(
		ctx context.Context,
		name string,
		uid types.UID,
		resourceVersion string,
		reason operations.TaskResultReason,
	) (*operations.OperationTask, error)
	TimeoutRunningTask(ctx context.Context, name string, uid types.UID, resourceVersion string) (*operations.OperationTask, error)

	AcquireLock(ctx context.Context, lock *operations.ExecutionLock) (*operations.ExecutionLock, bool, error)
	GetLock(ctx context.Context, name, resourceVersion string) (*operations.ExecutionLock, error)
	ReleaseLock(ctx context.Context, name string, lockUID, holderUID types.UID) error

	// CleanupByTargetUID is the controlled history purge for cluster deletion:
	// it removes the target's safe-terminal Operations, their Tasks and the
	// target's ExecutionLock, and refuses to touch anything while an Operation
	// or Task is still active.
	CleanupByTargetUID(ctx context.Context, targetUID types.UID) error
}

type StoreOptions struct {
	Operations rest.StandardStorage
	Tasks      rest.StandardStorage
	Locks      rest.StandardStorage
	// OperationsStrong/TasksStrong/LocksStrong serve safety-boundary reads
	// (target ordering, attempt creation, lock acquisition/release checks) from
	// storages that bypass the watch cache: the cache may lag behind etcd and
	// must never back an irreversible conclusion. When nil, the cacher-backed
	// storages above are used instead (test fakes).
	OperationsStrong rest.StandardStorage
	TasksStrong      rest.StandardStorage
	LocksStrong      rest.StandardStorage
	Now              func() time.Time
}

type store struct {
	operations       rest.StandardStorage
	tasks            rest.StandardStorage
	locks            rest.StandardStorage
	operationsStrong rest.StandardStorage
	tasksStrong      rest.StandardStorage
	locksStrong      rest.StandardStorage
	now              func() time.Time
}

var _ Store = (*store)(nil)

func NewStore(opts StoreOptions) (Store, error) {
	if opts.Operations == nil || opts.Tasks == nil || opts.Locks == nil {
		return nil, fmt.Errorf("all Operation v2 storages are required")
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	s := &store{
		operations: opts.Operations,
		tasks:      opts.Tasks,
		locks:      opts.Locks,
		now:        opts.Now,
	}
	s.operationsStrong = opts.OperationsStrong
	if s.operationsStrong == nil {
		s.operationsStrong = opts.Operations
	}
	s.tasksStrong = opts.TasksStrong
	if s.tasksStrong == nil {
		s.tasksStrong = opts.Tasks
	}
	s.locksStrong = opts.LocksStrong
	if s.locksStrong == nil {
		s.locksStrong = opts.Locks
	}
	return s, nil
}

func withNamespace(ctx context.Context) context.Context {
	return genericapirequest.WithNamespace(ctx, metav1.NamespaceNone)
}

func (s *store) GetOperation(ctx context.Context, name, resourceVersion string) (*operations.Operation, error) {
	obj, err := s.operations.Get(withNamespace(ctx), name, &metav1.GetOptions{ResourceVersion: resourceVersion})
	if err != nil {
		return nil, err
	}
	op, ok := obj.(*operations.Operation)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	return op, nil
}

func (s *store) getOperationStrong(ctx context.Context, name string) (*operations.Operation, error) {
	obj, err := s.operationsStrong.Get(withNamespace(ctx), name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	op, ok := obj.(*operations.Operation)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	return op, nil
}

func (s *store) CreateOperation(ctx context.Context, op *operations.Operation) (*operations.Operation, error) {
	if op == nil {
		return nil, fmt.Errorf("operation is nil")
	}
	obj, err := s.operations.Create(withNamespace(ctx), op, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	result, ok := obj.(*operations.Operation)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	return result, nil
}

// ListOperations is a safety-boundary read (target ordering, latest-operation
// guards) and is served by the quorum storage, bypassing the watch cache.
func (s *store) ListOperations(ctx context.Context, targetUID types.UID, resourceVersion string) (*operations.OperationList, error) {
	options := metav1.ListOptions{ResourceVersion: resourceVersion}
	if targetUID != "" {
		options.FieldSelector = fields.OneTermEqualSelector("spec.targetRef.uid", string(targetUID)).String()
	}
	internal, err := internalListOptions(&options)
	if err != nil {
		return nil, err
	}
	obj, err := s.operationsStrong.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.OperationList)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	return list, nil
}

func (s *store) ListOperationsWithOptions(ctx context.Context, options *metav1.ListOptions) (*operations.OperationList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	internal, err := internalListOptions(options)
	if err != nil {
		return nil, err
	}
	obj, err := s.operations.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.OperationList)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	return list, nil
}

func (s *store) WatchOperations(ctx context.Context, resourceVersion string) (watch.Interface, error) {
	return s.WatchOperationsWithOptions(ctx, &metav1.ListOptions{ResourceVersion: resourceVersion})
}

func (s *store) WatchOperationsWithOptions(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	internal, err := internalListOptions(options)
	if err != nil {
		return nil, err
	}
	return s.operations.Watch(withNamespace(ctx), internal)
}

func (s *store) UpdateOperationStatus(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	status *operations.OperationStatus,
) (*operations.Operation, error) {
	if err := validateStatusMessage(status.Message); err != nil {
		return nil, err
	}
	return s.updateOperation(ctx, name, uid, resourceVersion, func(op *operations.Operation) error {
		if err := validation.ValidateOperationPhaseTransition(op.Status.Phase, status.Phase); err != nil {
			isRetryReopen := op.Status.Phase != operations.OperationSucceeded && op.Status.Phase.IsTerminal() &&
				status.Phase == operations.OperationPending &&
				op.Spec.DesiredState == operations.OperationDesiredStateActive &&
				op.Spec.RetryGeneration > op.Status.ObservedRetryGeneration &&
				status.ObservedRetryGeneration == op.Spec.RetryGeneration
			if !isRetryReopen {
				return err
			}
		}
		op.Status = *status.DeepCopy()
		now := metav1.NewTime(s.now())
		if op.Status.Phase == operations.OperationRunning && op.Status.StartedAt == nil {
			op.Status.StartedAt = &now
		}
		if op.Status.Phase.IsTerminal() && op.Status.FinishedAt == nil {
			op.Status.FinishedAt = &now
		}
		return nil
	})
}

func (s *store) UpdateOperationControl(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	mutate func(*operations.OperationSpec) error,
) (*operations.Operation, error) {
	return s.updateOperation(ctx, name, uid, resourceVersion, func(op *operations.Operation) error {
		if mutate == nil {
			return fmt.Errorf("control mutator is required")
		}
		return mutate(&op.Spec)
	})
}

func (s *store) updateOperation(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	mutate func(*operations.Operation) error,
) (*operations.Operation, error) {
	obj, err := s.operations.Get(withNamespace(ctx), name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	op, ok := obj.(*operations.Operation)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", obj)
	}
	if uid != "" && op.UID != uid {
		return nil, apierrors.NewConflict(
			operations.Resource(operations.ResourceOperations),
			name,
			fmt.Errorf("operation UID does not match"),
		)
	}
	if resourceVersion != "" && op.ResourceVersion != resourceVersion {
		return nil, apierrors.NewConflict(
			operations.Resource(operations.ResourceOperations),
			name,
			fmt.Errorf("resourceVersion does not match"),
		)
	}
	updatedOperation := op.DeepCopy()
	if mutateErr := mutate(updatedOperation); mutateErr != nil {
		return nil, apierrors.NewBadRequest(mutateErr.Error())
	}
	updatedOperation.ResourceVersion = op.ResourceVersion
	updated, _, err := s.operations.Update(
		withNamespace(ctx),
		name,
		rest.DefaultUpdatedObjectInfo(updatedOperation),
		nil,
		nil,
		false,
		&metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, err
	}
	result, ok := updated.(*operations.Operation)
	if !ok {
		return nil, fmt.Errorf("operation storage returned %T", updated)
	}
	return result, nil
}

func (s *store) GetTask(ctx context.Context, name, resourceVersion string) (*operations.OperationTask, error) {
	obj, err := s.tasks.Get(withNamespace(ctx), name, &metav1.GetOptions{ResourceVersion: resourceVersion})
	if err != nil {
		return nil, err
	}
	task, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	return task, nil
}

func (s *store) getTaskStrong(ctx context.Context, name string) (*operations.OperationTask, error) {
	obj, err := s.tasksStrong.Get(withNamespace(ctx), name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	task, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("operation task storage returned %T", obj)
	}
	return task, nil
}

func (s *store) CreateTask(ctx context.Context, task *operations.OperationTask) (*operations.OperationTask, error) {
	obj, err := s.tasks.Create(withNamespace(ctx), task, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	result, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	return result, nil
}

// ListTasksByOperationUID is a safety-boundary read (terminal transition,
// attempt creation, lock-release re-check) and is served by the quorum
// storage, bypassing the watch cache.
func (s *store) ListTasksByOperationUID(
	ctx context.Context,
	operationUID types.UID,
	resourceVersion string,
) (*operations.OperationTaskList, error) {
	options := metav1.ListOptions{
		FieldSelector:   fields.OneTermEqualSelector("spec.operationRef.uid", string(operationUID)).String(),
		ResourceVersion: resourceVersion,
	}
	internal, err := internalListOptions(&options)
	if err != nil {
		return nil, err
	}
	obj, err := s.tasksStrong.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.OperationTaskList)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	return list, nil
}

// ListTasksByNode backs the agent's task discovery and is served by the
// quorum storage, bypassing the watch cache.
func (s *store) ListTasksByNode(ctx context.Context, nodeName, resourceVersion string) (*operations.OperationTaskList, error) {
	options := metav1.ListOptions{ResourceVersion: resourceVersion}
	internal, err := internalListOptions(&options)
	if err != nil {
		return nil, err
	}
	if nodeName != "" {
		internal.FieldSelector = fields.OneTermEqualSelector("spec.nodeRef.name", nodeName)
	}
	obj, err := s.tasksStrong.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.OperationTaskList)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	return list, nil
}

func (s *store) ListTasksWithOptions(
	ctx context.Context,
	nodeName string,
	options *metav1.ListOptions,
) (*operations.OperationTaskList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	internal, err := internalListOptions(options)
	if err != nil {
		return nil, err
	}
	if nodeName != "" {
		internal.FieldSelector = fields.OneTermEqualSelector("spec.nodeRef.name", nodeName)
	}
	obj, err := s.tasks.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.OperationTaskList)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	return list, nil
}

func (s *store) WatchTasks(ctx context.Context, nodeName, resourceVersion string) (watch.Interface, error) {
	return s.WatchTasksWithOptions(ctx, nodeName, &metav1.ListOptions{ResourceVersion: resourceVersion})
}

func (s *store) WatchTasksWithOptions(ctx context.Context, nodeName string, options *metav1.ListOptions) (watch.Interface, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	internal, err := internalListOptions(options)
	if err != nil {
		return nil, err
	}
	if nodeName != "" {
		internal.FieldSelector = fields.OneTermEqualSelector("spec.nodeRef.name", nodeName)
	}
	return s.tasks.Watch(withNamespace(ctx), internal)
}

func (s *store) UpdateTaskStatus(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	status operations.OperationTaskStatus,
) (*operations.OperationTask, error) {
	if err := validateTaskStatus(status); err != nil {
		return nil, err
	}
	obj, err := s.tasks.Get(withNamespace(ctx), name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	task, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	if uid != "" && task.UID != uid {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("task UID does not match"))
	}
	if resourceVersion != "" && task.ResourceVersion != resourceVersion {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("resourceVersion does not match"))
	}
	if transitionErr := validateAgentTaskTransition(task, status); transitionErr != nil {
		return nil, transitionErr
	}
	updatedTask := task.DeepCopy()
	updatedTask.Status = *status.DeepCopy()
	now := metav1.NewTime(s.now())
	if updatedTask.Status.Phase == operations.TaskRunning && updatedTask.Status.StartedAt == nil {
		updatedTask.Status.StartedAt = &now
	}
	if updatedTask.Status.Phase.IsTerminal() && updatedTask.Status.FinishedAt == nil {
		updatedTask.Status.FinishedAt = &now
	}
	updatedTask.ResourceVersion = task.ResourceVersion
	updated, _, err := s.tasks.Update(
		withNamespace(ctx),
		name,
		rest.DefaultUpdatedObjectInfo(updatedTask),
		nil,
		nil,
		false,
		&metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, err
	}
	result, ok := updated.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", updated)
	}
	return result, nil
}

func validateTaskStatus(status operations.OperationTaskStatus) error {
	if err := validation.ValidateTaskResult(status.Result); err != nil {
		return err
	}
	if status.Phase == operations.TaskSucceeded && status.Result != nil && status.Result.Reason != "" {
		return apierrors.NewBadRequest("Succeeded task must not have a result reason")
	}
	if status.Phase != operations.TaskSucceeded && status.Result != nil && len(status.Result.Outputs) > 0 {
		return apierrors.NewBadRequest("outputs are only allowed on Succeeded tasks")
	}
	return nil
}

func validateAgentTaskTransition(task *operations.OperationTask, status operations.OperationTaskStatus) error {
	if err := validation.ValidateTaskPhaseTransition(task.Status.Phase, status.Phase); err != nil {
		return apierrors.NewConflict(operations.Resource(operations.ResourceTasks), task.Name, err)
	}
	if status.Phase == operations.TaskCancelled ||
		(task.Status.Phase == operations.TaskPending && status.Phase != operations.TaskPending && status.Phase != operations.TaskRunning) {
		return apierrors.NewConflict(
			operations.Resource(operations.ResourceTasks), task.Name,
			fmt.Errorf("agent cannot transition %s task to %s", task.Status.Phase, status.Phase),
		)
	}
	return nil
}

func (s *store) CancelPendingTask(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	reason operations.TaskResultReason,
) (*operations.OperationTask, error) {
	return s.updateTaskTerminal(ctx, name, uid, resourceVersion, operations.TaskPending, operations.TaskCancelled, reason)
}

func (s *store) TimeoutRunningTask(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
) (*operations.OperationTask, error) {
	return s.updateTaskTerminal(
		ctx,
		name,
		uid,
		resourceVersion,
		operations.TaskRunning,
		operations.TaskTimedOut,
		operations.TaskReasonDeadlineExceeded,
	)
}

func (s *store) updateTaskTerminal(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	expected, phase operations.TaskPhase,
	reason operations.TaskResultReason,
) (*operations.OperationTask, error) {
	return s.updateTask(ctx, name, uid, resourceVersion, func(task *operations.OperationTask) error {
		if task.Status.Phase != expected {
			return apierrors.NewConflict(
				operations.Resource(operations.ResourceTasks),
				name,
				fmt.Errorf("task is %s, expected %s", task.Status.Phase, expected),
			)
		}
		task.Status.Phase = phase
		task.Status.Result = &operations.TaskResult{Reason: reason}
		return nil
	})
}

func (s *store) updateTask(
	ctx context.Context,
	name string,
	uid types.UID,
	resourceVersion string,
	mutate func(*operations.OperationTask) error,
) (*operations.OperationTask, error) {
	obj, err := s.tasks.Get(withNamespace(ctx), name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	task, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", obj)
	}
	if uid != "" && task.UID != uid {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("task UID does not match"))
	}
	if resourceVersion != "" && task.ResourceVersion != resourceVersion {
		return nil, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), name, fmt.Errorf("resourceVersion does not match"))
	}
	updatedTask := task.DeepCopy()
	if mutateErr := mutate(updatedTask); mutateErr != nil {
		return nil, mutateErr
	}
	if updatedTask.Status.Phase.IsTerminal() && updatedTask.Status.FinishedAt == nil {
		now := metav1.NewTime(s.now())
		updatedTask.Status.FinishedAt = &now
	}
	updatedTask.ResourceVersion = task.ResourceVersion
	updated, _, err := s.tasks.Update(
		withNamespace(ctx),
		name,
		rest.DefaultUpdatedObjectInfo(updatedTask),
		nil,
		nil,
		false,
		&metav1.UpdateOptions{},
	)
	if err != nil {
		return nil, err
	}
	result, ok := updated.(*operations.OperationTask)
	if !ok {
		return nil, fmt.Errorf("task storage returned %T", updated)
	}
	return result, nil
}

func (s *store) AcquireLock(ctx context.Context, lock *operations.ExecutionLock) (*operations.ExecutionLock, bool, error) {
	obj, err := s.locks.Create(withNamespace(ctx), lock, nil, &metav1.CreateOptions{})
	if err == nil {
		result, ok := obj.(*operations.ExecutionLock)
		if !ok {
			return nil, false, fmt.Errorf("lock storage returned %T", obj)
		}
		return result, true, nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return nil, false, err
	}
	// Create is the atomic ownership decision. The follow-up read must be
	// consistent: RV=0 may return a deleted predecessor from the watch cache
	// and make a former holder look current after the lock was recreated.
	existing, getErr := s.GetLock(ctx, lock.Name, "")
	if getErr != nil {
		// A predecessor may release the lock between Create's AlreadyExists
		// result and this read. Treat that narrow race as contention; the
		// controller will retry acquisition from durable state.
		if apierrors.IsNotFound(getErr) {
			return nil, false, nil
		}
		return nil, false, getErr
	}
	return existing, false, nil
}

func (s *store) GetLock(ctx context.Context, name, resourceVersion string) (*operations.ExecutionLock, error) {
	// Lock ownership is an irreversible safety boundary: AcquireLock uses this
	// read after an AlreadyExists result and ReleaseLock uses it before Delete.
	// A cacher can return a deleted predecessor after a lock is recreated, so
	// this must read the quorum storage.
	obj, err := s.locksStrong.Get(withNamespace(ctx), name, &metav1.GetOptions{ResourceVersion: resourceVersion})
	if err != nil {
		return nil, err
	}
	lock, ok := obj.(*operations.ExecutionLock)
	if !ok {
		return nil, fmt.Errorf("lock storage returned %T", obj)
	}
	return lock, nil
}

func (s *store) ReleaseLock(ctx context.Context, name string, lockUID, holderUID types.UID) error {
	lock, err := s.GetLock(ctx, name, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if lock.UID != lockUID || lock.Spec.HolderRef.UID != holderUID {
		return apierrors.NewConflict(operations.Resource(operations.ResourceLocks), name, fmt.Errorf("lock holder or UID changed"))
	}
	uid := lock.UID
	_, _, err = s.locks.Delete(withNamespace(ctx), name, func(_ context.Context, _ runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func validateStatusMessage(message string) error {
	if len(message) > operations.MaxMessageSize {
		return fmt.Errorf("status message exceeds %d bytes", operations.MaxMessageSize)
	}
	return nil
}

func internalListOptions(options *metav1.ListOptions) (*metainternalversion.ListOptions, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	labelSelector, err := labels.Parse(options.LabelSelector)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid labelSelector: %v", err))
	}
	fieldSelector, err := fields.ParseSelector(options.FieldSelector)
	if err != nil {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid fieldSelector: %v", err))
	}
	internal := &metainternalversion.ListOptions{
		LabelSelector:        labelSelector,
		FieldSelector:        fieldSelector,
		Watch:                options.Watch,
		AllowWatchBookmarks:  options.AllowWatchBookmarks,
		ResourceVersion:      options.ResourceVersion,
		ResourceVersionMatch: options.ResourceVersionMatch,
		TimeoutSeconds:       options.TimeoutSeconds,
		Limit:                options.Limit,
		Continue:             options.Continue,
	}
	if errs := metainternalvalidation.ValidateListOptions(internal); len(errs) != 0 {
		return nil, apierrors.NewBadRequest(errs.ToAggregate().Error())
	}
	return internal, nil
}

// CleanupByTargetUID implements the controlled history purge required for
// cluster deletion. There is no generic GC for Operation/Task/Lock objects, so
// this is the only path that removes them. Enumeration uses list reads (a
// stale view can only leave objects behind, never delete live ones), while
// every terminal-state check re-reads the object from the strong storage, so
// an irreversible delete is never backed by a lagging cache. The method
// tolerates already-deleted objects and is safe to retry.
func (s *store) CleanupByTargetUID(ctx context.Context, targetUID types.UID) error {
	if targetUID == "" {
		return fmt.Errorf("target UID is required")
	}
	ops, err := s.ListOperations(ctx, targetUID, "")
	if err != nil {
		return err
	}
	if err := s.verifyTerminalHistory(ctx, ops.Items); err != nil {
		return err
	}
	if err := s.purgeHistory(ctx, ops.Items); err != nil {
		return err
	}
	return s.deleteStaleLocks(ctx, targetUID)
}

// verifyTerminalHistory re-reads every Operation and Task of the target with a
// quorum Get and refuses while anything is still active.
func (s *store) verifyTerminalHistory(ctx context.Context, ops []operations.Operation) error {
	for i := range ops {
		op, err := s.getOperationStrong(ctx, ops[i].Name)
		if err != nil {
			return err
		}
		if !op.Status.Phase.IsTerminal() {
			return fmt.Errorf("operation %s is not terminal", op.Name)
		}
		tasks, err := s.ListTasksByOperationUID(ctx, op.UID, "")
		if err != nil {
			return err
		}
		for j := range tasks.Items {
			task, err := s.getTaskStrong(ctx, tasks.Items[j].Name)
			if err != nil {
				return err
			}
			if !task.Status.Phase.IsTerminal() {
				return fmt.Errorf("task %s is not terminal", task.Name)
			}
		}
	}
	return nil
}

// purgeHistory deletes the tasks of every Operation first, then the Operations
// themselves.
func (s *store) purgeHistory(ctx context.Context, ops []operations.Operation) error {
	for i := range ops {
		tasks, err := s.ListTasksByOperationUID(ctx, ops[i].UID, "")
		if err != nil {
			return err
		}
		for j := range tasks.Items {
			if err := s.deleteObject(ctx, s.tasks, tasks.Items[j].Name); err != nil {
				return err
			}
		}
		if err := s.deleteObject(ctx, s.operations, ops[i].Name); err != nil {
			return err
		}
	}
	return nil
}

// deleteStaleLocks removes ExecutionLock objects of a deleted target: a crash
// between terminal-status write and lock release can leave the lock behind.
func (s *store) deleteStaleLocks(ctx context.Context, targetUID types.UID) error {
	locks, err := s.listLocks(ctx)
	if err != nil {
		return err
	}
	for i := range locks.Items {
		if locks.Items[i].Spec.TargetRef.UID == targetUID {
			if err := s.deleteObject(ctx, s.locks, locks.Items[i].Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteObject deletes by name and tolerates an already-deleted object, which
// is what makes repeated cleanup calls safe.
func (*store) deleteObject(ctx context.Context, storage rest.StandardStorage, name string) error {
	_, _, err := storage.Delete(withNamespace(ctx), name, nil, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *store) listLocks(ctx context.Context) (*operations.ExecutionLockList, error) {
	internal, err := internalListOptions(&metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	obj, err := s.locks.List(withNamespace(ctx), internal)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*operations.ExecutionLockList)
	if !ok {
		return nil, fmt.Errorf("lock storage returned %T", obj)
	}
	return list, nil
}
