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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

type stepFacts struct {
	Step       *operations.OperationStep
	Tasks      []*operations.OperationTask
	ByNode     map[types.UID][]*operations.OperationTask
	Incomplete []operations.NodeReference
}

func validateAndCurrentStep(op *operations.Operation, tasks []operations.OperationTask) (*stepFacts, bool, error) {
	if op == nil {
		return nil, false, fmt.Errorf("operation is nil")
	}
	steps := make(map[string]int, len(op.Spec.Steps))
	targets := make(map[string]map[types.UID]operations.NodeReference, len(op.Spec.Steps))
	for index := range op.Spec.Steps {
		step := &op.Spec.Steps[index]
		if _, exists := steps[step.ID]; exists {
			return nil, false, fmt.Errorf("duplicate step %q", step.ID)
		}
		steps[step.ID] = index
		targets[step.ID] = make(map[types.UID]operations.NodeReference, len(step.Targets))
		for _, target := range step.Targets {
			if _, exists := targets[step.ID][target.UID]; exists {
				return nil, false, fmt.Errorf("duplicate node UID %q in step %q", target.UID, step.ID)
			}
			targets[step.ID][target.UID] = target
		}
	}

	byStep := make(map[string][]*operations.OperationTask, len(op.Spec.Steps))
	for index := range tasks {
		task := &tasks[index]
		if task.Spec.OperationRef.Name != op.Name || task.Spec.OperationRef.UID != op.UID {
			return nil, false, fmt.Errorf("task %q refers to another operation", task.Name)
		}
		stepIndex, exists := steps[task.Spec.StepID]
		if !exists {
			return nil, false, fmt.Errorf("task %q refers to unknown step %q", task.Name, task.Spec.StepID)
		}
		target, exists := targets[task.Spec.StepID][task.Spec.NodeRef.UID]
		if !exists || target.Name != task.Spec.NodeRef.Name {
			return nil, false, fmt.Errorf("task %q refers to an unknown node in step %q", task.Name, task.Spec.StepID)
		}
		if task.Spec.RetryGeneration > op.Spec.RetryGeneration {
			return nil, false, fmt.Errorf("task %q has future retry generation %d", task.Name, task.Spec.RetryGeneration)
		}
		if task.Spec.Executor != op.Spec.Steps[stepIndex].Executor {
			return nil, false, fmt.Errorf("task %q executor differs from its immutable plan", task.Name)
		}
		byStep[task.Spec.StepID] = append(byStep[task.Spec.StepID], task)
	}

	for stepIndex := range op.Spec.Steps {
		step := &op.Spec.Steps[stepIndex]
		facts := &stepFacts{
			Step:   step,
			Tasks:  byStep[step.ID],
			ByNode: make(map[types.UID][]*operations.OperationTask, len(step.Targets)),
		}
		for _, task := range facts.Tasks {
			facts.ByNode[task.Spec.NodeRef.UID] = append(facts.ByNode[task.Spec.NodeRef.UID], task)
		}
		for nodeUID := range facts.ByNode {
			sort.SliceStable(facts.ByNode[nodeUID], func(i, j int) bool {
				left, right := facts.ByNode[nodeUID][i], facts.ByNode[nodeUID][j]
				if left.Spec.Attempt != right.Spec.Attempt {
					return left.Spec.Attempt < right.Spec.Attempt
				}
				return left.CreationTimestamp.Before(&right.CreationTimestamp)
			})
		}
		for _, target := range step.Targets {
			if !nodeSucceeded(facts.ByNode[target.UID]) {
				facts.Incomplete = append(facts.Incomplete, target)
			}
		}
		if len(facts.Incomplete) != 0 {
			for later := stepIndex + 1; later < len(op.Spec.Steps); later++ {
				if len(byStep[op.Spec.Steps[later].ID]) != 0 {
					return nil, false, fmt.Errorf("tasks for step %q exist before step %q completed", op.Spec.Steps[later].ID, step.ID)
				}
			}
			return facts, false, nil
		}
	}
	return nil, true, nil
}

func nodeSucceeded(tasks []*operations.OperationTask) bool {
	for _, task := range tasks {
		if task.Status.Phase == operations.TaskSucceeded {
			return true
		}
	}
	return false
}

func latestTask(tasks []*operations.OperationTask) *operations.OperationTask {
	if len(tasks) == 0 {
		return nil
	}
	latest := tasks[0]
	for _, task := range tasks[1:] {
		if task.Spec.Attempt > latest.Spec.Attempt ||
			(task.Spec.Attempt == latest.Spec.Attempt && latest.CreationTimestamp.Before(&task.CreationTimestamp)) {
			latest = task
		}
	}
	return latest
}

func activeTasks(tasks []operations.OperationTask) (pending, running []*operations.OperationTask) {
	for index := range tasks {
		task := &tasks[index]
		switch task.Status.Phase {
		case operations.TaskPending:
			pending = append(pending, task)
		case operations.TaskRunning:
			running = append(running, task)
		}
	}
	return pending, running
}

func latestFailure(facts *stepFacts) bool {
	for _, target := range facts.Incomplete {
		task := latestTask(facts.ByNode[target.UID])
		if task == nil {
			continue
		}
		switch task.Status.Phase {
		case operations.TaskFailed, operations.TaskTimedOut:
			return true
		}
	}
	return false
}

func nextAttempt(
	step *operations.OperationStep,
	tasks []*operations.OperationTask,
	retryGeneration int64,
) (int32, bool) {
	if nodeSucceeded(tasks) || hasActiveTask(tasks) {
		return 0, false
	}
	if len(tasks) == 0 {
		return 0, true
	}
	baseGeneration, next := taskGenerationAndNextAttempt(tasks)
	if retryGeneration != baseGeneration {
		return retryGenerationAttempt(tasks, retryGeneration, baseGeneration, next)
	}
	return next, executedAttempts(tasks, baseGeneration) < 1+step.RetryLimit
}

func hasActiveTask(tasks []*operations.OperationTask) bool {
	for _, task := range tasks {
		if task.Status.Phase == operations.TaskPending || task.Status.Phase == operations.TaskRunning {
			return true
		}
	}
	return false
}

func taskGenerationAndNextAttempt(tasks []*operations.OperationTask) (baseGeneration int64, next int32) {
	baseGeneration = tasks[0].Spec.RetryGeneration
	for _, task := range tasks {
		if task.Spec.RetryGeneration < baseGeneration {
			baseGeneration = task.Spec.RetryGeneration
		}
		if task.Spec.Attempt >= next {
			next = task.Spec.Attempt + 1
		}
	}
	return baseGeneration, next
}

func retryGenerationAttempt(tasks []*operations.OperationTask, retryGeneration, baseGeneration int64, next int32) (int32, bool) {
	if retryGeneration < baseGeneration {
		return 0, false
	}
	for _, task := range tasks {
		if task.Spec.RetryGeneration == retryGeneration {
			return 0, false
		}
	}
	return next, true
}

func executedAttempts(tasks []*operations.OperationTask, generation int64) int32 {
	var executed int32
	for _, task := range tasks {
		if task.Spec.RetryGeneration != generation {
			continue
		}
		if task.Status.StartedAt != nil || task.Status.Phase == operations.TaskRunning ||
			task.Status.Phase == operations.TaskSucceeded || task.Status.Phase == operations.TaskFailed ||
			task.Status.Phase == operations.TaskTimedOut {
			executed++
		}
	}
	return executed
}

func materializePayload(step *operations.OperationStep, tasks []operations.OperationTask) (runtime.RawExtension, error) {
	if len(step.Inputs) == 0 {
		return *step.Payload.DeepCopy(), nil
	}
	payload := make(map[string]json.RawMessage)
	if len(step.Payload.Raw) != 0 {
		if err := json.Unmarshal(step.Payload.Raw, &payload); err != nil {
			return runtime.RawExtension{}, fmt.Errorf("decode payload for step %q: %w", step.ID, err)
		}
		if payload == nil {
			return runtime.RawExtension{}, fmt.Errorf("payload for step %q must be a JSON object", step.ID)
		}
	}
	for _, input := range step.Inputs {
		var matches []*operations.OperationTask
		for index := range tasks {
			task := &tasks[index]
			if task.Spec.StepID == input.FromStepID && task.Spec.NodeRef.UID == input.FromNodeUID &&
				task.Status.Phase == operations.TaskSucceeded {
				matches = append(matches, task)
			}
		}
		if len(matches) != 1 || matches[0].Status.Result == nil {
			return runtime.RawExtension{}, fmt.Errorf(
				"input %q for step %q has %d successful source tasks",
				input.Field,
				step.ID,
				len(matches),
			)
		}
		value, exists := matches[0].Status.Result.Outputs[input.OutputKey]
		if !exists {
			return runtime.RawExtension{}, fmt.Errorf("output %q is missing for input %q in step %q", input.OutputKey, input.Field, step.ID)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return runtime.RawExtension{}, err
		}
		payload[input.Field] = encoded
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return runtime.RawExtension{}, err
	}
	return runtime.RawExtension{Raw: raw}, nil
}

func operationBefore(left, right *operations.Operation) bool {
	if left.CreationTimestamp.Equal(&right.CreationTimestamp) {
		leftRevision, leftErr := strconv.ParseUint(left.ResourceVersion, 10, 64)
		rightRevision, rightErr := strconv.ParseUint(right.ResourceVersion, 10, 64)
		if leftErr == nil && rightErr == nil && leftRevision != rightRevision {
			return leftRevision < rightRevision
		}
		return string(left.UID) < string(right.UID)
	}
	return left.CreationTimestamp.Before(&right.CreationTimestamp)
}

func operationNeedsExecution(op *operations.Operation) bool {
	return !op.Status.Phase.IsTerminal() || op.Spec.RetryGeneration > op.Status.ObservedRetryGeneration
}

func isEarliestRunnable(op *operations.Operation, operationsList []operations.Operation) bool {
	for index := range operationsList {
		candidate := &operationsList[index]
		if candidate.UID == op.UID || !operationNeedsExecution(candidate) {
			continue
		}
		if operationBefore(candidate, op) {
			return false
		}
	}
	return true
}

func isLatestOperation(op *operations.Operation, operationsList []operations.Operation) bool {
	for index := range operationsList {
		candidate := &operationsList[index]
		if candidate.UID != op.UID && operationBefore(op, candidate) {
			return false
		}
	}
	return true
}

func taskSpecEqual(left, right *operations.OperationTaskSpec) bool {
	return apiequality.Semantic.DeepEqual(left, right)
}

func deadlineExpired(op *operations.Operation, now time.Time) bool {
	return op.Status.Deadline != nil && !now.Before(op.Status.Deadline.Time)
}
