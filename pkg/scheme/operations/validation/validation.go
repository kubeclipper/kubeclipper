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

// Package validation contains the small, shared admission contract for the
// Operation Engine v2.  It is deliberately independent of HTTP and storage so
// the API server, controller and tests use the same rules.
package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func ValidateOperation(op *operations.Operation) field.ErrorList {
	allErrs := field.ErrorList{}
	if op == nil {
		return append(allErrs, field.Required(field.NewPath("operation"), "must not be nil"))
	}
	if op.Name == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("metadata", "name"), "stable name is required"))
	}
	if op.GenerateName != "" {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("metadata", "generateName"), "generateName is not supported"))
	}
	allErrs = append(allErrs, validateReference(op.Spec.TargetRef, field.NewPath("spec", "targetRef"))...)
	if op.Spec.Action == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "action"), "action is required"))
	}
	if op.Spec.DesiredState != operations.OperationDesiredStateActive {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "desiredState"), op.Spec.DesiredState, "new operations must be Active"))
	}
	if op.Spec.RetryGeneration != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "retryGeneration"), op.Spec.RetryGeneration, "new operations must start at generation 0"))
	}
	if op.Spec.Timeout.Duration < operations.MinOperationTimeout || op.Spec.Timeout.Duration > operations.MaxOperationTimeout {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "timeout"), op.Spec.Timeout.Duration, fmt.Sprintf("must be between %s and %s", operations.MinOperationTimeout, operations.MaxOperationTimeout)))
	}
	if len(op.Spec.Steps) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "steps"), "at least one step is required"))
	}
	if len(op.Spec.Steps) > operations.MaxSteps {
		allErrs = append(allErrs, field.TooMany(field.NewPath("spec", "steps"), len(op.Spec.Steps), operations.MaxSteps))
	}
	seenSteps := make(map[string]int, len(op.Spec.Steps))
	for i := range op.Spec.Steps {
		step := &op.Spec.Steps[i]
		path := field.NewPath("spec", "steps").Index(i)
		if step.ID == "" {
			allErrs = append(allErrs, field.Required(path.Child("id"), "step id is required"))
		} else if previous, ok := seenSteps[step.ID]; ok {
			allErrs = append(allErrs, field.Duplicate(path.Child("id"), fmt.Sprintf("also used at index %d", previous)))
		} else {
			seenSteps[step.ID] = i
		}
		if step.Executor == "" {
			allErrs = append(allErrs, field.Required(path.Child("executor"), "executor is required"))
		}
		if len(step.Payload.Raw) > operations.MaxStepPayloadSize {
			allErrs = append(allErrs, field.TooLong(path.Child("payload"), len(step.Payload.Raw), operations.MaxStepPayloadSize))
		}
		if len(step.Targets) == 0 {
			allErrs = append(allErrs, field.Required(path.Child("targets"), "at least one target is required"))
		}
		if len(step.Targets) > operations.MaxTargetsPerStep {
			allErrs = append(allErrs, field.TooMany(path.Child("targets"), len(step.Targets), operations.MaxTargetsPerStep))
		}
		seenNodes := make(map[string]struct{}, len(step.Targets))
		for j := range step.Targets {
			targetPath := path.Child("targets").Index(j)
			allErrs = append(allErrs, validateNodeReference(step.Targets[j], targetPath)...)
			key := string(step.Targets[j].UID)
			if key != "" {
				if _, ok := seenNodes[key]; ok {
					allErrs = append(allErrs, field.Duplicate(targetPath, "node UID is duplicated in this step"))
				}
				seenNodes[key] = struct{}{}
			}
		}
		if step.RetryLimit < 0 || step.RetryLimit > operations.MaxRetryLimit {
			allErrs = append(allErrs, field.Invalid(path.Child("retryLimit"), step.RetryLimit, fmt.Sprintf("must be between 0 and %d", operations.MaxRetryLimit)))
		}
		for j := range step.Inputs {
			input := step.Inputs[j]
			inputPath := path.Child("inputs").Index(j)
			if input.Field == "" || input.FromStepID == "" || input.FromNodeUID == "" || input.OutputKey == "" {
				allErrs = append(allErrs, field.Required(inputPath, "field, fromStepID, fromNodeUID and outputKey are required"))
			}
			fromIndex, ok := seenSteps[input.FromStepID]
			if !ok || fromIndex >= i {
				allErrs = append(allErrs, field.Invalid(inputPath.Child("fromStepID"), input.FromStepID, "must reference an earlier step"))
			}
		}
	}
	if size, err := json.Marshal(op); err == nil && len(size) > operations.MaxOperationSize {
		allErrs = append(allErrs, field.TooLong(field.NewPath("operation"), len(size), operations.MaxOperationSize))
	}
	return allErrs
}

func ValidateTask(task *operations.OperationTask) field.ErrorList {
	allErrs := field.ErrorList{}
	if task == nil {
		return append(allErrs, field.Required(field.NewPath("task"), "must not be nil"))
	}
	if task.Name == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("metadata", "name"), "deterministic name is required"))
	}
	if task.GenerateName != "" {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("metadata", "generateName"), "generateName is not supported"))
	}
	allErrs = append(allErrs, validateReference(task.Spec.OperationRef, field.NewPath("spec", "operationRef"))...)
	allErrs = append(allErrs, validateNodeReference(task.Spec.NodeRef, field.NewPath("spec", "nodeRef"))...)
	if task.Spec.StepID == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "stepID"), "step id is required"))
	}
	if task.Spec.Executor == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "executor"), "executor is required"))
	}
	if task.Spec.RetryGeneration < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "retryGeneration"), task.Spec.RetryGeneration, "must not be negative"))
	}
	if task.Spec.Attempt < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "attempt"), task.Spec.Attempt, "must not be negative"))
	}
	if len(task.Spec.Payload.Raw) > operations.MaxStepPayloadSize {
		allErrs = append(allErrs, field.TooLong(field.NewPath("spec", "payload"), len(task.Spec.Payload.Raw), operations.MaxStepPayloadSize))
	}
	if size, err := json.Marshal(task); err == nil && len(size) > operations.MaxTaskSize {
		allErrs = append(allErrs, field.TooLong(field.NewPath("task"), len(size), operations.MaxTaskSize))
	}
	return allErrs
}

func ValidateTaskResult(result *operations.TaskResult) error {
	if result == nil {
		return nil
	}
	if len(result.Message) > operations.MaxMessageSize {
		return fmt.Errorf("task result message exceeds %d bytes", operations.MaxMessageSize)
	}
	var total int
	for key, value := range result.Outputs {
		if len(key) == 0 || len(key) > operations.MaxOutputKeySize {
			return fmt.Errorf("task output key must be between 1 and %d bytes", operations.MaxOutputKeySize)
		}
		if len(value) > operations.MaxOutputValueSize {
			return fmt.Errorf("task output %q exceeds %d bytes", key, operations.MaxOutputValueSize)
		}
		total += len(key) + len(value)
	}
	if total > operations.MaxTaskOutputsSize {
		return fmt.Errorf("task outputs exceed %d bytes", operations.MaxTaskOutputsSize)
	}
	return nil
}

func ValidateTaskPhaseTransition(oldPhase, newPhase operations.TaskPhase) error {
	if oldPhase.IsTerminal() {
		if oldPhase != newPhase {
			return fmt.Errorf("terminal task phase %q is immutable", oldPhase)
		}
		return nil
	}
	switch oldPhase {
	case "":
		if newPhase != operations.TaskPending {
			return fmt.Errorf("new task must start Pending")
		}
	case operations.TaskPending:
		if newPhase != operations.TaskPending && newPhase != operations.TaskRunning && newPhase != operations.TaskCancelled {
			return fmt.Errorf("Pending task cannot transition to %q", newPhase)
		}
	case operations.TaskRunning:
		if newPhase != operations.TaskRunning && !newPhase.IsTerminal() {
			return fmt.Errorf("Running task cannot transition to %q", newPhase)
		}
	default:
		return fmt.Errorf("unknown old task phase %q", oldPhase)
	}
	return nil
}

func ValidateOperationPhaseTransition(oldPhase, newPhase operations.OperationPhase) error {
	if oldPhase.IsTerminal() {
		if oldPhase != newPhase {
			return fmt.Errorf("terminal operation phase %q is immutable", oldPhase)
		}
		return nil
	}
	switch oldPhase {
	case "":
		if newPhase != operations.OperationPending {
			return fmt.Errorf("new operation must start Pending")
		}
	case operations.OperationPending:
		if newPhase != operations.OperationPending && newPhase != operations.OperationRunning && newPhase != operations.OperationCancelled {
			return fmt.Errorf("Pending operation cannot transition to %q", newPhase)
		}
	case operations.OperationRunning:
		if newPhase != operations.OperationRunning && !newPhase.IsTerminal() {
			return fmt.Errorf("Running operation cannot transition to %q", newPhase)
		}
	default:
		return fmt.Errorf("unknown old operation phase %q", oldPhase)
	}
	return nil
}

func validateReference(ref operations.ObjectReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if strings.TrimSpace(ref.Kind) == "" {
		errs = append(errs, field.Required(path.Child("kind"), "kind is required"))
	}
	if strings.TrimSpace(ref.Name) == "" {
		errs = append(errs, field.Required(path.Child("name"), "name is required"))
	}
	if ref.UID == "" {
		errs = append(errs, field.Required(path.Child("uid"), "uid is required"))
	}
	return errs
}

func validateNodeReference(ref operations.NodeReference, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	if strings.TrimSpace(ref.Name) == "" {
		errs = append(errs, field.Required(path.Child("name"), "name is required"))
	}
	if ref.UID == "" {
		errs = append(errs, field.Required(path.Child("uid"), "uid is required"))
	}
	return errs
}

func InvalidOperation(name string, errs field.ErrorList) error {
	return apierrors.NewInvalid(operations.Kind(operations.KindOperation), name, errs)
}

func InvalidTask(name string, errs field.ErrorList) error {
	return apierrors.NewInvalid(operations.Kind(operations.KindOperationTask), name, errs)
}

var _ runtime.Object = (*operations.Operation)(nil)
