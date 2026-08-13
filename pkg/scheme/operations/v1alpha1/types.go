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

package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// OperationControlRequest carries optimistic-concurrency preconditions for
// cancel and retry operations.
type OperationControlRequest struct {
	UID             types.UID `json:"uid"`
	ResourceVersion string    `json:"resourceVersion"`
}

const (
	DefaultOperationTimeout = 90 * time.Minute
	MinOperationTimeout     = time.Minute
	MaxOperationTimeout     = 24 * time.Hour

	AgentTerminationGrace  = 30 * time.Second
	ServerTerminationGrace = 2 * time.Minute

	MaxOperationSize   = 512 * 1024
	MaxTaskSize        = 256 * 1024
	MaxSteps           = 256
	MaxTargetsPerStep  = 1000
	MaxStepPayloadSize = 128 * 1024
	MaxRetryLimit      = 3
	MaxMessageSize     = 4 * 1024
	MaxOutputKeySize   = 128
	MaxOutputValueSize = 4 * 1024
	MaxTaskOutputsSize = 16 * 1024
)

type ObjectReference struct {
	Kind string    `json:"kind"`
	Name string    `json:"name"`
	UID  types.UID `json:"uid"`
}

type NodeReference struct {
	Name string    `json:"name"`
	UID  types.UID `json:"uid"`
}

type OperationDesiredState string

const (
	OperationDesiredStateActive    OperationDesiredState = "Active"
	OperationDesiredStateCancelled OperationDesiredState = "Cancelled"
)

type OperationPhase string

const (
	OperationPending   OperationPhase = "Pending"
	OperationRunning   OperationPhase = "Running"
	OperationSucceeded OperationPhase = "Succeeded"
	OperationFailed    OperationPhase = "Failed"
	OperationTimedOut  OperationPhase = "TimedOut"
	OperationCancelled OperationPhase = "Cancelled"
)

func (p OperationPhase) IsTerminal() bool {
	switch p {
	case OperationSucceeded, OperationFailed, OperationTimedOut, OperationCancelled:
		return true
	default:
		return false
	}
}

type OperationReason string

const (
	OperationReasonStepFailed            OperationReason = "StepFailed"
	OperationReasonDeadlineExceeded      OperationReason = "DeadlineExceeded"
	OperationReasonCancelledByRequest    OperationReason = "CancelledByRequest"
	OperationReasonInvalidExecutionFacts OperationReason = "InvalidExecutionFacts"
)

type OperationSpec struct {
	TargetRef       ObjectReference       `json:"targetRef"`
	Action          string                `json:"action"`
	DesiredState    OperationDesiredState `json:"desiredState"`
	RetryGeneration int64                 `json:"retryGeneration,omitempty"`
	Timeout         metav1.Duration       `json:"timeout"`
	Steps           []OperationStep       `json:"steps"`
}

type OperationStep struct {
	ID         string               `json:"id"`
	Targets    []NodeReference      `json:"targets"`
	Executor   string               `json:"executor"`
	Payload    runtime.RawExtension `json:"payload"`
	Inputs     []StepInput          `json:"inputs,omitempty"`
	RetryLimit int32                `json:"retryLimit,omitempty"`
}

type StepInput struct {
	Field       string    `json:"field"`
	FromStepID  string    `json:"fromStepID"`
	FromNodeUID types.UID `json:"fromNodeUID"`
	OutputKey   string    `json:"outputKey"`
}

type OperationStatus struct {
	Phase                   OperationPhase  `json:"phase"`
	ObservedRetryGeneration int64           `json:"observedRetryGeneration,omitempty"`
	Reason                  OperationReason `json:"reason,omitempty"`
	Message                 string          `json:"message,omitempty"`
	Deadline                *metav1.Time    `json:"deadline,omitempty"`
	StartedAt               *metav1.Time    `json:"startedAt,omitempty"`
	FinishedAt              *metav1.Time    `json:"finishedAt,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +genclient:onlyVerbs=create,get,list,watch
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Operation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OperationSpec   `json:"spec"`
	Status            OperationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OperationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operation `json:"items"`
}

type TaskPhase string

const (
	TaskPending   TaskPhase = "Pending"
	TaskRunning   TaskPhase = "Running"
	TaskSucceeded TaskPhase = "Succeeded"
	TaskFailed    TaskPhase = "Failed"
	TaskTimedOut  TaskPhase = "TimedOut"
	TaskCancelled TaskPhase = "Cancelled"
)

func (p TaskPhase) IsTerminal() bool {
	switch p {
	case TaskSucceeded, TaskFailed, TaskTimedOut, TaskCancelled:
		return true
	default:
		return false
	}
}

type TaskResultReason string

const (
	TaskReasonExecutionFailed                      TaskResultReason = "ExecutionFailed"
	TaskReasonDeadlineExceeded                     TaskResultReason = "DeadlineExceeded"
	TaskReasonOperationCancelled                   TaskResultReason = "OperationCancelled"
	TaskReasonSiblingFailed                        TaskResultReason = "SiblingFailed"
	TaskReasonOperationDeadlineExceededBeforeStart TaskResultReason = "OperationDeadlineExceededBeforeStart"
)

type OperationTaskSpec struct {
	OperationRef    ObjectReference      `json:"operationRef"`
	StepID          string               `json:"stepID"`
	NodeRef         NodeReference        `json:"nodeRef"`
	RetryGeneration int64                `json:"retryGeneration"`
	Attempt         int32                `json:"attempt"`
	Executor        string               `json:"executor"`
	Payload         runtime.RawExtension `json:"payload"`
	Deadline        metav1.Time          `json:"deadline"`
}

type TaskResult struct {
	Reason  TaskResultReason  `json:"reason,omitempty"`
	Message string            `json:"message,omitempty"`
	Outputs map[string]string `json:"outputs,omitempty"`
}

type OperationTaskStatus struct {
	Phase      TaskPhase    `json:"phase"`
	Result     *TaskResult  `json:"result,omitempty"`
	StartedAt  *metav1.Time `json:"startedAt,omitempty"`
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:onlyVerbs=get,list,watch,updateStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OperationTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OperationTaskSpec   `json:"spec"`
	Status            OperationTaskStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OperationTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperationTask `json:"items"`
}

type ExecutionLockSpec struct {
	TargetRef ObjectReference `json:"targetRef"`
	HolderRef ObjectReference `json:"holderRef"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExecutionLock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ExecutionLockSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExecutionLockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExecutionLock `json:"items"`
}
