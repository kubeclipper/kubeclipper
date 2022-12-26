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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false
type Node struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ProxyIpv4CIDR     string `json:"proxyIpv4CIDR" description:"proxy ip address of node, only use when bastion not able to reach client ip but client can reach bastion ip"`
	// Most recently observed status of the node.
	// Populated by the system.
	// Read-only.
	// +optional
	Status NodeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// NodeList contains a list of Node
type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Node `json:"items"`
}

// ResourceName is the name identifying various resources in a ResourceList.
type ResourceName string

// Resource names must be not more than 63 characters, consisting of upper- or lower-case alphanumeric characters,
// with the -, _, and . characters allowed anywhere, except the first or last character.
// The default convention, matching that for annotations, is to use lower-case names, with dashes, rather than
// camel case, separating compound words.
// Fully-qualified resource typenames are constructed from a DNS-style subdomain, followed by a slash `/` and a name.
const (
	// CPU, in cores. (500m = .5 cores)
	ResourceCPU ResourceName = "cpu"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceMemory ResourceName = "memory"
	// Volume size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	ResourceStorage ResourceName = "storage"
	// Local ephemeral storage, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// The resource name for ResourceEphemeralStorage is alpha and it can change across releases.
	ResourceEphemeralStorage ResourceName = "ephemeral-storage"
)

// ResourceList is a set of (resource name, quantity) pairs.
type ResourceList map[ResourceName]resource.Quantity

type NodeConditionType string

// These are valid conditions of node. Currently, we don't have enough information to decide
// node condition. In the future, we will add more. The proposed set of conditions are:
// NodeReachable, NodeLive, NodeReady, NodeSchedulable, NodeRunnable.
const (
	NodeReady              NodeConditionType = "Ready"
	NodeMemoryPressure     NodeConditionType = "MemoryPressure"
	NodeDiskPressure       NodeConditionType = "DiskPressure"
	NodePIDPressure        NodeConditionType = "PIDPressure"
	NodeNetworkUnavailable NodeConditionType = "NetworkUnavailable"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// NodeCondition contains condition information for a node.
type NodeCondition struct {
	// Type of node condition.
	Type NodeConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type NodeAddressType string

// These are valid address type of node.
const (
	NodeHostName    NodeAddressType = "Hostname"
	NodeIPV4Addr    NodeAddressType = "IPV4"
	NodeIPV6Addr    NodeAddressType = "IPV6"
	NodeExternalIP  NodeAddressType = "ExternalIP"
	NodeInternalIP  NodeAddressType = "InternalIP"
	NodeExternalDNS NodeAddressType = "ExternalDNS"
	NodeInternalDNS NodeAddressType = "InternalDNS"
)

// NodeAddress contains information for the node's address.
type NodeAddress struct {
	// Node address type, one of Hostname, ExternalIP or InternalIP.
	Type NodeAddressType `json:"type"`
	// The node address.
	Address string `json:"address"`
}

// NodeSystemInfo is a set of ids/uuids to uniquely identify the node.
type NodeSystemInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"` // ex: freebsd, linux
	Arch            string `json:"arch"`
	Platform        string `json:"platform"`        // ex: ubuntu, linuxmint
	PlatformFamily  string `json:"platformFamily"`  // ex: debian, rhel
	PlatformVersion string `json:"platformVersion"` // version of the complete OS
	KernelVersion   string `json:"kernelVersion"`   // version of the OS kernel (if available)
	KernelArch      string `json:"kernelArch"`      // native cpu architecture queried at runtime, as returned by `uname -m` or empty string in case of error
	HostID          string `json:"hostId"`          // MachineId
}

type UniqueVolumeName string

// AttachedVolume describes a volume attached to a node
type AttachedVolume struct {
	// Name of the attached volume
	Name UniqueVolumeName `json:"name"`

	// DevicePath represents the device path where the volume should be available
	DevicePath string `json:"devicePath"`
}

type NodeStatus struct {
	// Note: It is usually used for routing between kc nodes and belongs to the operation and maintenance management network.
	Ipv4DefaultIP string `json:"ipv4DefaultIP" description:"node ipv4 default gateway interface ip"`
	Ipv4DefaultGw string `json:"ipv4DefaultGw" description:"node ipv4 default gateway ip"`
	// Note: It is commonly used for routing between nodes in a kubernetes cluster.
	// If the `--node-ip-detect` parameter is not specified at deployment time, it is the same as `Ipv4DefaultIP`;
	// otherwise, it depends on the ip detect method.
	NodeIpv4DefaultIP string `json:"nodeIpv4DefaultIP"`
	// Capacity represents the total resources of a node.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#capacity
	// +optional
	Capacity ResourceList `json:"capacity,omitempty"`
	// Allocatable represents the resources of a node that are available for scheduling.
	// Defaults to Capacity.
	// +optional
	Allocatable ResourceList `json:"allocatable,omitempty"`
	// Conditions is an array of current observed node conditions.
	// More info: https://kubernetes.io/docs/concepts/nodes/node/#condition
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []NodeCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// List of addresses reachable to the node.
	// Queried from cloud provider, if available.
	// More info: https://kubernetes.io/docs/concepts/nodes/node/#addresses
	// Note: This field is declared as mergeable, but the merge key is not sufficiently
	// unique, which can cause data corruption when it is merged. Callers should instead
	// use a full-replacement patch. See http://pr.k8s.io/79391 for an example.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Addresses []NodeAddress `json:"addresses,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// Set of ids/uuids to uniquely identify the node.
	// More info: https://kubernetes.io/docs/concepts/nodes/node/#info
	// +optional
	NodeInfo NodeSystemInfo `json:"nodeInfo,omitempty"`
	// List of volumes that are attached to the node.
	// +optional
	VolumesAttached      []AttachedVolume `json:"volumesAttached,omitempty"`
	ContainerRuntimeInfo ContainerRuntime `json:"containerRuntime"`
}
