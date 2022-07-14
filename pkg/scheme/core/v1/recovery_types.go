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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Recovery struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	UseBackupName     string `json:"useBackupName"`
	// a node selected for executing recovery tasks
	Description string `json:"description,omitempty" optional:"true"`
}

// ClusterBackupStatus describes the status of a cluster backup
//type ClusterRecoveryStatus string

const DefaultRecoveryTimeoutSec = 1200

const (
//// ClusterRecoveryCreating means the recovery is in creating.
//ClusterRecoveryCreating ClusterRecoveryStatus = "creating"
//// ClusterRecoveryCreating means the recovery is available for restoring.
//ClusterRecoveryAvailable ClusterRecoveryStatus = "available"
//// ClusterRecoveryError means the recovery is created failed and must not be used.
//ClusterRecoveryError ClusterRecoveryStatus = "error"
//// ClusterRecoveryRestoring means the recovery is in using for restoring.
//ClusterRecoveryRestoring ClusterRecoveryStatus = "restoring"
)

/* type RecoveryStatus struct {
	Status ClusterRecoveryStatus `json:"status"`
} */

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// RecoveryList contains a list of Recovery

type RecoveryList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recovery `json:"items"`
}
