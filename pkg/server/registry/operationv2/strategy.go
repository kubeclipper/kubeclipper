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
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	validation "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/validation"
)

type operationStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var _ rest.RESTCreateStrategy = operationStrategy{}
var _ rest.RESTUpdateStrategy = operationStrategy{}
var _ rest.RESTDeleteStrategy = operationStrategy{}

func newOperationStrategy(typer runtime.ObjectTyper) operationStrategy {
	return operationStrategy{ObjectTyper: typer, NameGenerator: names.SimpleNameGenerator}
}

func (operationStrategy) NamespaceScoped() bool                                     { return false }
func (operationStrategy) WarningsOnCreate(context.Context, runtime.Object) []string { return nil }
func (operationStrategy) WarningsOnUpdate(context.Context, runtime.Object, runtime.Object) []string {
	return nil
}
func (operationStrategy) AllowCreateOnUpdate() bool      { return false }
func (operationStrategy) AllowUnconditionalUpdate() bool { return false }
func (operationStrategy) Canonicalize(runtime.Object)    {}

func (operationStrategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	op := obj.(*operations.Operation)
	op.Status = operations.OperationStatus{Phase: operations.OperationPending}
	if op.Spec.Timeout.Duration == 0 {
		op.Spec.Timeout.Duration = operations.DefaultOperationTimeout
	}
	if op.Spec.DesiredState == "" {
		op.Spec.DesiredState = operations.OperationDesiredStateActive
	}
	op.TypeMeta = metav1TypeMeta(operations.KindOperation)
}

func (operationStrategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	op := obj.(*operations.Operation)
	oldOp := old.(*operations.Operation)
	// Status is updated through OperationStore; ordinary REST updates may only
	// change the two explicitly controlled spec fields.
	op.Spec.TargetRef = oldOp.Spec.TargetRef
	op.Spec.Action = oldOp.Spec.Action
	op.Spec.Timeout = oldOp.Spec.Timeout
	op.Spec.Steps = oldOp.Spec.Steps
	op.Status = oldOp.Status
	op.TypeMeta = oldOp.TypeMeta
}

func (operationStrategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateOperation(obj.(*operations.Operation))
}

func (operationStrategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	op := obj.(*operations.Operation)
	oldOp := old.(*operations.Operation)
	var errs field.ErrorList
	if !equality.Semantic.DeepEqual(op.Spec.TargetRef, oldOp.Spec.TargetRef) ||
		op.Spec.Action != oldOp.Spec.Action ||
		op.Spec.Timeout != oldOp.Spec.Timeout ||
		!equality.Semantic.DeepEqual(op.Spec.Steps, oldOp.Spec.Steps) {
		errs = append(errs, field.Forbidden(field.NewPath("spec"), "operation plan is immutable"))
	}
	if err := validation.ValidateOperationPhaseTransition(oldOp.Status.Phase, op.Status.Phase); err != nil {
		isRetryReopen := oldOp.Status.Phase != operations.OperationSucceeded && oldOp.Status.Phase.IsTerminal() &&
			op.Status.Phase == operations.OperationPending &&
			op.Spec.DesiredState == operations.OperationDesiredStateActive &&
			op.Spec.RetryGeneration > oldOp.Status.ObservedRetryGeneration &&
			op.Status.ObservedRetryGeneration == op.Spec.RetryGeneration
		if !isRetryReopen {
			errs = append(errs, field.Invalid(field.NewPath("status", "phase"), op.Status.Phase, err.Error()))
		}
	}
	return errs
}

type taskStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var _ rest.RESTCreateStrategy = taskStrategy{}
var _ rest.RESTUpdateStrategy = taskStrategy{}
var _ rest.RESTDeleteStrategy = taskStrategy{}

func newTaskStrategy(typer runtime.ObjectTyper) taskStrategy {
	return taskStrategy{ObjectTyper: typer, NameGenerator: names.SimpleNameGenerator}
}

func (taskStrategy) NamespaceScoped() bool                                     { return false }
func (taskStrategy) WarningsOnCreate(context.Context, runtime.Object) []string { return nil }
func (taskStrategy) WarningsOnUpdate(context.Context, runtime.Object, runtime.Object) []string {
	return nil
}
func (taskStrategy) AllowCreateOnUpdate() bool      { return false }
func (taskStrategy) AllowUnconditionalUpdate() bool { return false }
func (taskStrategy) Canonicalize(runtime.Object)    {}

func (taskStrategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	task := obj.(*operations.OperationTask)
	task.Status = operations.OperationTaskStatus{Phase: operations.TaskPending}
	task.TypeMeta = metav1TypeMeta(operations.KindOperationTask)
}

func (taskStrategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	task := obj.(*operations.OperationTask)
	oldTask := old.(*operations.OperationTask)
	task.Spec = oldTask.Spec
	task.TypeMeta = oldTask.TypeMeta
}

func (taskStrategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateTask(obj.(*operations.OperationTask))
}

func (taskStrategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	task := obj.(*operations.OperationTask)
	oldTask := old.(*operations.OperationTask)
	var errs field.ErrorList
	if !equality.Semantic.DeepEqual(task.Spec, oldTask.Spec) {
		errs = append(errs, field.Forbidden(field.NewPath("spec"), "task spec is immutable"))
	}
	if err := validation.ValidateTaskPhaseTransition(oldTask.Status.Phase, task.Status.Phase); err != nil {
		errs = append(errs, field.Invalid(field.NewPath("status", "phase"), task.Status.Phase, err.Error()))
	}
	if err := validation.ValidateTaskResult(task.Status.Result); err != nil {
		errs = append(errs, field.Invalid(field.NewPath("status", "result"), task.Status.Result, err.Error()))
	}
	return errs
}

type lockStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var _ rest.RESTCreateStrategy = lockStrategy{}
var _ rest.RESTUpdateStrategy = lockStrategy{}
var _ rest.RESTDeleteStrategy = lockStrategy{}

func newLockStrategy(typer runtime.ObjectTyper) lockStrategy {
	return lockStrategy{ObjectTyper: typer, NameGenerator: names.SimpleNameGenerator}
}

func (lockStrategy) NamespaceScoped() bool                                     { return false }
func (lockStrategy) WarningsOnCreate(context.Context, runtime.Object) []string { return nil }
func (lockStrategy) WarningsOnUpdate(context.Context, runtime.Object, runtime.Object) []string {
	return nil
}
func (lockStrategy) AllowCreateOnUpdate() bool      { return false }
func (lockStrategy) AllowUnconditionalUpdate() bool { return false }
func (lockStrategy) Canonicalize(runtime.Object)    {}
func (lockStrategy) PrepareForCreate(_ context.Context, obj runtime.Object) {
	obj.(*operations.ExecutionLock).TypeMeta = metav1TypeMeta(operations.KindExecutionLock)
}
func (lockStrategy) PrepareForUpdate(_ context.Context, obj, old runtime.Object) {
	lock := obj.(*operations.ExecutionLock)
	oldLock := old.(*operations.ExecutionLock)
	lock.Spec = oldLock.Spec
	lock.TypeMeta = oldLock.TypeMeta
}
func (lockStrategy) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	lock := obj.(*operations.ExecutionLock)
	var errs field.ErrorList
	if lock.Name == "" {
		errs = append(errs, field.Required(field.NewPath("metadata", "name"), "lock name is required"))
	}
	errs = append(errs, validateLockReference(lock.Spec.TargetRef, field.NewPath("spec", "targetRef"))...)
	errs = append(errs, validateLockReference(lock.Spec.HolderRef, field.NewPath("spec", "holderRef"))...)
	if lock.Spec.HolderRef.Kind != operations.KindOperation {
		errs = append(
			errs,
			field.Invalid(field.NewPath("spec", "holderRef", "kind"), lock.Spec.HolderRef.Kind, "holder must be an Operation"),
		)
	}
	return errs
}
func (lockStrategy) ValidateUpdate(_ context.Context, obj, old runtime.Object) field.ErrorList {
	if !equality.Semantic.DeepEqual(obj.(*operations.ExecutionLock).Spec, old.(*operations.ExecutionLock).Spec) {
		return field.ErrorList{field.Forbidden(field.NewPath("spec"), "execution lock is immutable")}
	}
	return nil
}

func validateLockReference(ref operations.ObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if ref.Kind == "" {
		errs = append(errs, field.Required(path.Child("kind"), "kind is required"))
	}
	if ref.Name == "" {
		errs = append(errs, field.Required(path.Child("name"), "name is required"))
	}
	if ref.UID == "" {
		errs = append(errs, field.Required(path.Child("uid"), "uid is required"))
	}
	return errs
}

func metav1TypeMeta(kind string) metav1.TypeMeta {
	return metav1.TypeMeta{APIVersion: operations.SchemeGroupVersion.String(), Kind: kind}
}

func attrsForOperation(obj runtime.Object) (labelSet labels.Set, fieldSet fields.Set, err error) {
	op, ok := obj.(*operations.Operation)
	if !ok {
		return nil, nil, fmt.Errorf("expected Operation, got %T", obj)
	}
	set := generic.ObjectMetaFieldsSet(&op.ObjectMeta, false)
	set["spec.targetRef.uid"] = string(op.Spec.TargetRef.UID)
	return op.Labels, set, nil
}

func attrsForTask(obj runtime.Object) (labelSet labels.Set, fieldSet fields.Set, err error) {
	task, ok := obj.(*operations.OperationTask)
	if !ok {
		return nil, nil, fmt.Errorf("expected OperationTask, got %T", obj)
	}
	set := generic.ObjectMetaFieldsSet(&task.ObjectMeta, false)
	set["spec.nodeRef.name"] = task.Spec.NodeRef.Name
	set["spec.operationRef.uid"] = string(task.Spec.OperationRef.UID)
	return task.Labels, set, nil
}

func attrsForLock(obj runtime.Object) (labelSet labels.Set, fieldSet fields.Set, err error) {
	lock, ok := obj.(*operations.ExecutionLock)
	if !ok {
		return nil, nil, fmt.Errorf("expected ExecutionLock, got %T", obj)
	}
	set := generic.ObjectMetaFieldsSet(&lock.ObjectMeta, false)
	set["spec.targetRef.uid"] = string(lock.Spec.TargetRef.UID)
	return lock.Labels, set, nil
}
