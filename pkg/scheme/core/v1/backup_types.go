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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Backup struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            BackupStatus      `json:"backupStatus,omitempty"`
	ClusterNodes      map[string]string `json:"clusterNodes"`
	// a node selected for executing backup tasks
	PreferredNode   string `json:"preferredNode,omitempty" optional:"true"`
	BackupPointName string `json:"backupPointName"`
}

type BackupStatus struct {
	KubernetesVersion   string `json:"kubernetesVersion"`
	FileName            string `json:"fileName"`
	BackupFileSize      int64  `json:"backupFileSize"`
	BackupFileMD5       string `json:"backupFileMD5"`
	ClusterBackupStatus `json:"status"`
}

// ClusterBackupStatus describes the status of a cluster backup
type ClusterBackupStatus string

const DefaultBackupTimeoutSec = 1200

const (
	// ClusterBackupCreating means the backup is in creating.
	ClusterBackupCreating ClusterBackupStatus = "creating"
	// ClusterBackupAvailable means the backup is available for restoring.
	ClusterBackupAvailable ClusterBackupStatus = "available"
	// ClusterBackupError means the backup is created failed and must not be used.
	ClusterBackupError ClusterBackupStatus = "error"
	// ClusterBackupRestoring means the backup is in using for restoring.
	ClusterBackupRestoring ClusterBackupStatus = "restoring"
)

/* type BackupStatus struct {
	Status ClusterBackupStatus `json:"status"`
} */

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// BackupList contains a list of Backup

type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}
