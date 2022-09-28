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
	"k8s.io/apimachinery/pkg/runtime"
)

// CloudProvider
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false
type CloudProvider struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Type              string `json:"type,omitempty"`
	Region            string `json:"region,omitempty"`
	// SSH config for connect to nodes.
	SSH    SSH                  `json:"ssh,omitempty"`
	Config runtime.RawExtension `json:"config,omitempty"`
	Status CloudProviderStatus  `json:"status"`
}

type CloudProviderStatus struct {
	Conditions []CloudProviderCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// CloudProviderCondition contains condition information for a cloudProvider.
type CloudProviderCondition struct {
	// Type of node condition.
	Type CloudProviderConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	//  Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type CloudProviderConditionType string

const (
	CloudProviderReady       CloudProviderConditionType = "Ready"
	CloudProviderProgressing CloudProviderConditionType = "Progressing"
)

type SSH struct {
	User               string `json:"user,omitempty"`
	Password           string `json:"password,omitempty"`
	PrivateKey         string `json:"privateKey,omitempty"`
	PrivateKeyPassword string `json:"privateKeyPassword,omitempty"`
	Port               int    `json:"port,omitempty"`
}

// CloudProviderList is a resource containing a list of CloudProvider objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CloudProviderList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta

	// Items is the list of CloudProvider.
	Items []CloudProvider
}

const (
	CloudProviderCreated      = "Created"
	CloudProviderSyncing      = "Syncing"
	CloudProviderSyncSucceed  = "SyncSucceed"
	CloudProviderSyncFailed   = "SyncFailed"
	CloudProviderRemoving     = "Removing"
	CloudProviderRemoveFailed = "RemoveFailed"
)
