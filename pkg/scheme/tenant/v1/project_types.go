// Package v1 implements tenant v1 resource's informer.
package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Project define
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec   `json:"spec,omitempty"`
	Status            ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	Manager string   `json:"manager,omitempty"`
	Nodes   []string `json:"nodes,omitempty"`
}

// ProjectStatus defines the status of Project
type ProjectStatus struct {
	Count ProjectCount `json:"count"`
}

// ProjectCount defines the count of ProjectStatus
type ProjectCount struct {
	Cluster int64 `json:"cluster"`
	Node    int64 `json:"node"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
