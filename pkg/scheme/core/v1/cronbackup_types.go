/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type CronBackup struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CronBackupSpec `json:"spec"`
	// +optional
	Status CronBackupStatus `json:"status,omitempty"`
}

// CronBackupSpec defines the desired state of CronBackup
type CronBackupSpec struct {
	// the cluster to which the cronBackup belongs
	ClusterName string `json:"clusterName,omitempty"`
	// the schedule in cron format
	Schedule string `json:"schedule,omitempty"`
	// maximum number of reserved backups
	MaxBackupNum int `json:"maxBackupNum,omitempty"`
	// specific run time
	RunAt *metav1.Time `json:"runAt,omitempty"`
}

// CronBackupStatus defines the status of cronBackup
type CronBackupStatus struct {
	// next create backup time
	NextScheduleTime *metav1.Time `json:"NextScheduleTime,omitempty"`
	// last create backup time
	LastScheduleTime *metav1.Time `json:"LastScheduleTime,omitempty"`
	// last successfully create backup time
	LastSuccessfulTime *metav1.Time `json:"LastSuccessfulTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CronBackupList contains a list of CronBackup

type CronBackupList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronBackup `json:"items"`
}
