/*
 *
 *  * Copyright 2021 KubeClipper Authors.
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

package v1

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Operation struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Steps             []Step          `json:"steps,omitempty"`
	Status            OperationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// OperationList contains a list of User
type OperationList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operation `json:"items"`
}

func (op *Operation) GetStep(stepID string) (step Step, ok bool) {
	for _, s := range op.Steps {
		if s.ID == stepID {
			return s, true
		}
	}
	return
}

// default operation timeout is 90 min

const DefaultOperationTimeoutSecs = "5400"

type OperationStatusType string

const (
	OperationStatusPending     OperationStatusType = "pending"
	OperationStatusRunning     OperationStatusType = "running"
	OperationStatusFailed      OperationStatusType = "failed"
	OperationStatusTermination OperationStatusType = "termination"
	OperationStatusUnknown     OperationStatusType = "unknown"
	OperationStatusSuccessful  OperationStatusType = "successful"
)

type OperationStatus struct {
	Status     OperationStatusType  `json:"status,omitempty"`
	Conditions []OperationCondition `json:"conditions,omitempty"`
}

type StepAction string

var ErrInvalidAction = errors.New("invalid step action")

const (
	ActionInstall   StepAction = "install"
	ActionUninstall StepAction = "uninstall"
	ActionUpgrade   StepAction = "upgrade"
)

const (
	OperationCreateCluster                = "CreateCluster"
	OperationDeleteCluster                = "DeleteCluster"
	OperationUpgradeCluster               = "UpgradeCluster"
	OperationAddNodes                     = "AddNodes"
	OperationRemoveNodes                  = "RemoveNodes"
	OperationBackupCluster                = "BackupCluster"
	OperationDeleteBackup                 = "DeleteBackup"
	OperationRecoverCluster               = "RecoveryCluster"
	OperationInstallComponents            = "InstallComponents"
	OperationUninstallComponents          = "UninstallComponents"
	OperationUpdateCertification          = "UpdateCertifications"
	OperationUpdateAPIServerCertification = "UpdateAPIServerCertifications"
)

// Step TODO: add commands struct instead of string
type Step struct {
	ID                string          `json:"id,omitempty"`
	Name              string          `json:"name,omitempty"`
	Nodes             []StepNode      `json:"nodes,omitempty"`
	Action            StepAction      `json:"action,omitempty"`
	Timeout           metav1.Duration `json:"timeout,omitempty"`
	ErrIgnore         bool            `json:"errIgnore"`
	Commands          []Command       `json:"commands,omitempty"`
	BeforeRunCommands []Command       `json:"beforeRunCommands,omitempty"`
	AfterRunCommands  []Command       `json:"afterRunCommands,omitempty"`
	RetryTimes        int32           `json:"retryTimes,omitempty"`
	AutomaticRetry    bool            `json:"automaticRetry"`
}

type StepNode struct {
	ID       string `json:"id,omitempty"`
	IPv4     string `json:"ipv4,omitempty"`
	NodeIPv4 string `json:"nodeIPv4,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type CommandType string

const (
	CommandShell          CommandType = "shell"
	CommandTemplateRender CommandType = "templateRender"
	CommandCustom         CommandType = "custom"
)

type TemplateCommand struct {
	Identity string `json:"identity,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

type Command struct {
	Type          CommandType      `json:"type"`
	ShellCommand  []string         `json:"shellCommand,omitempty"`
	Identity      string           `json:"identity,omitempty"`
	CustomCommand []byte           `json:"customCommand,omitempty"`
	Template      *TemplateCommand `json:"template,omitempty"`
}

// OperationCondition contains condition information for a node.
type OperationCondition struct {
	StepID string       `json:"stepID,omitempty"`
	Status []StepStatus `json:"status,omitempty"`
}

type StepStatusType string

const (
	StepStatusSuccessful StepStatusType = "successful"
	StepStatusFailed     StepStatusType = "failed"
)

type StepStatus struct {
	StartAt metav1.Time    `json:"startAt,omitempty"`
	EndAt   metav1.Time    `json:"endAt,omitempty"`
	Node    string         `json:"node,omitempty"`
	Status  StepStatusType `json:"status,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message  string `json:"message,omitempty"`
	Response []byte `json:"response,omitempty"`
}

type PendingOperation struct {
	OperationID            string `json:"operationID"`
	OperationType          string `json:"operationType"`
	OperationSponsor       string `json:"operationSponsor"`
	Timeout                string `json:"timeout"`
	ClusterResourceVersion string `json:"clusterResourceVersion"`
	ExtraData              []byte `json:"extraData,omitempty"`
}
