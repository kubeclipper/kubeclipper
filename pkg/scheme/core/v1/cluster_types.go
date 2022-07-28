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
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ClusterFinalizer   = "finalizer.cluster.kubeclipper.io"
	OperationFinalizer = "finalizer.operation.kubeclipper.io"
	BackupFinalizer    = "finalizer.backup.kubeclipper.io"
)

type WorkerNode struct {
	ID     string            `json:"id"`
	Labels map[string]string `json:"labels,omitempty"`
	Taints []Taint           `json:"taints,omitempty"`
}

type WorkerNodeList []WorkerNode

func (l WorkerNodeList) GetNodeIDs() (nodes []string) {
	for _, node := range l {
		nodes = append(nodes, node.ID)
	}
	return
}

// Intersection of original worker nodes list and the provided one.
func (l WorkerNodeList) Intersect(nodes ...WorkerNode) WorkerNodeList {
	out := make(WorkerNodeList, 0)
	nodesMap := map[string]struct{}{}
	for _, n := range l {
		nodesMap[n.ID] = struct{}{}
	}
	for _, n := range nodes {
		if _, ok := nodesMap[n.ID]; ok {
			out = append(out, n)
		}
	}
	return out
}

// Relative complement of original worker nodes list in the provided one.
func (l WorkerNodeList) Complement(nodes ...WorkerNode) WorkerNodeList {
	out := make(WorkerNodeList, 0)
	nodesMap := map[string]struct{}{}
	for _, n := range nodes {
		nodesMap[n.ID] = struct{}{}
	}
	for _, n := range l {
		if _, ok := nodesMap[n.ID]; !ok {
			out = append(out, n)
		}
	}
	return out
}

func (c Cluster) GetAllNodes() sets.String {
	s := sets.NewString(c.Kubeadm.Masters.GetNodeIDs()...)
	s.Insert(c.Kubeadm.Workers.GetNodeIDs()...)
	return s
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Cluster struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	ClusterType ClusterType   `json:"type"`
	Kubeadm     *Kubeadm      `json:"kubeadm,omitempty"`
	Status      ClusterStatus `json:"status,omitempty" optional:"true"`
	KubeConfig  []byte        `json:"kubeconfig,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ClusterList contains a list of Cluster

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

type ClusterType string

const (
	ClusterKubeadm ClusterType = "kubeadm"
	// TODO: other cluster type
)

func (c ClusterType) String() string {
	return string(c)
}

func (c Cluster) Complete() {
	switch c.ClusterType {
	case ClusterKubeadm:
		if c.Kubeadm.Offline {
			c.Kubeadm.KubeComponents.CNI.LocalRegistry = c.Kubeadm.LocalRegistry
		}
		matchCniVersion(c.Kubeadm.KubernetesVersion, &c.Kubeadm.KubeComponents.CNI)
	}
}

var k8sMatchCniVersion = map[string]map[string]string{
	"118": {"calico": "v3.11.2"},
	"119": {"calico": "v3.11.2"},
	"120": {"calico": "v3.21.2"},
	"121": {"calico": "v3.21.2"},
	"122": {"calico": "v3.21.2"},
	"123": {"calico": "v3.21.2"},
}

func matchCniVersion(k8sVersion string, cni *CNI) {
	if k8sVersion == "" || cni == nil {
		return
	}
	k8sVersion = strings.ReplaceAll(k8sVersion, "v", "")
	k8sVersion = strings.ReplaceAll(k8sVersion, ".", "")

	k8sVersion = strings.Join(strings.Split(k8sVersion, "")[0:3], "")
	switch cni.Type {
	case "calico":
		cni.Calico.Version = k8sMatchCniVersion[k8sVersion][cni.Type]
	}
}

type Kubeadm struct {
	Description       string           `json:"description,omitempty" optional:"true"`
	Masters           WorkerNodeList   `json:"masters"`
	Workers           WorkerNodeList   `json:"workers" optional:"true"`
	KubernetesVersion string           `json:"kubernetesVersion" enum:"v1.20.13"`
	CertSANs          []string         `json:"certSANs,omitempty" optional:"true"`
	LocalRegistry     string           `json:"localRegistry,omitempty" optional:"true"`
	ContainerRuntime  ContainerRuntime `json:"containerRuntime"`
	Networking        Networking       `json:"networking"`
	KubeComponents    KubeComponents   `json:"kubeComponents"`
	Components        []Component      `json:"components" optional:"true"`
	WorkerNodeVip     string           `json:"workerNodeVip" optional:"true"`
	Offline           bool             `json:"offline" optional:"true"`
}

type ClusterStatusType string

const (
	ClusterStatusRunning       ClusterStatusType = "Running"
	ClusterStatusInstalling    ClusterStatusType = "Installing"
	ClusterStatusInstallFailed ClusterStatusType = "InstallFailed"
	ClusterStatusUpdating      ClusterStatusType = "Updating"
	ClusterStatusUpdateFailed  ClusterStatusType = "UpdateFailed"
	ClusterStatusUpgrading     ClusterStatusType = "Upgrading"
	ClusterStatusUpgradeFailed ClusterStatusType = "UpgradeFailed"
	ClusterStatusBackingUp     ClusterStatusType = "BackingUp"
	ClusterStatusRestoring     ClusterStatusType = "Restoring"
	ClusterStatusRestoreFailed ClusterStatusType = "RestoreFailed"
	ClusterStatusDeleting      ClusterStatusType = "Deleting"
	ClusterStatusDeleteFailed  ClusterStatusType = "DeleteFailed"
)

type Taint struct {
	// Required. The taint key to be applied to a node.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// The taint value corresponding to the taint key.
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
	// Required. The effect of the taint on pods
	// that do not tolerate the taint.
	// Valid effects are NoSchedule, PreferNoSchedule and NoExecute.
	Effect TaintEffect `json:"effect" protobuf:"bytes,3,opt,name=effect,casttype=TaintEffect"`
}

type TaintEffect string

const (
	TaintEffectNoSchedule       TaintEffect = "NoSchedule"
	TaintEffectPreferNoSchedule TaintEffect = "PreferNoSchedule"
	TaintEffectNoExecute        TaintEffect = "NoExecute"
)

type ClusterStatus struct {
	Status         ClusterStatusType `json:"status,omitempty"`
	Certifications []Certification   `json:"certifications,omitempty"`
	// cluster component health status
	ComponentConditions []ComponentConditions `json:"componentConditions,omitempty"`
	Conditions          []ClusterCondition    `json:"conditions,omitempty"`
}

type ComponentStatus string

const (
	ComponentHealthy     ComponentStatus = "Healthy"
	ComponentUnhealthy   ComponentStatus = "Unhealthy"
	ComponentUnKnown     ComponentStatus = "Unknown"
	ComponentUnsupported ComponentStatus = "Unsupported"
)

type Certification struct {
	Name           string `json:"name,omitempty"`
	CAName         string `json:"caName"`
	ExpirationTime string `json:"expirationTime,omitempty"`
}

type ComponentConditions struct {
	Name     string          `json:"name"`
	Category string          `json:"category"`
	Status   ComponentStatus `json:"status"`
}

type ClusterCondition struct {
	Operation       string              `json:"operation,omitempty"`
	OperationID     string              `json:"operationID,omitempty"`
	StartAt         metav1.Time         `json:"startAt,omitempty"`
	OperationStatus OperationStatusType `json:"operationStatus,omitempty"`
}

type Component struct {
	Name    string               `json:"name"`
	Version string               `json:"version"`
	Config  runtime.RawExtension `json:"config"`
}

type Networking struct {
	ServiceSubnet string `json:"serviceSubnet"`
	PodSubnet     string `json:"podSubnet"`
	DNSDomain     string `json:"dnsDomain"`
}

type Kubelet struct {
	RootDir string `json:"rootDir" yaml:"rootDir"`
}

type CNI struct {
	LocalRegistry string `json:"localRegistry" optional:"true"`
	Type          string `json:"type" enum:"calico"`
	PodIPv4CIDR   string `json:"podIPv4CIDR"`
	PodIPv6CIDR   string `json:"podIPv6CIDR"`
	MTU           int    `json:"mtu"`
	Calico        Calico `json:"calico" optional:"true"`
}

type Calico struct {
	IPv4AutoDetection string `json:"IPv4AutoDetection" enum:"first-found|can-reach=DESTINATION|interface=INTERFACE-REGEX|skip-interface=INTERFACE-REGEX"`
	IPv6AutoDetection string `json:"IPv6AutoDetection" enum:"first-found|can-reach=DESTINATION|interface=INTERFACE-REGEX|skip-interface=INTERFACE-REGEX"`
	Mode              string `json:"mode" enum:"BGP|Overlay-IPIP-All|Overlay-IPIP-Cross-Subnet|Overlay-Vxlan-All|Overlay-Vxlan-Cross-Subnet|overlay"`
	DualStack         bool   `json:"dualStack" optional:"true"`
	IPManger          bool   `json:"IPManger" optional:"true"`
	Version           string `json:"version" enum:"v3.11.2"`
}

type Etcd struct {
	DataDir string `json:"dataDir,omitempty" optional:"true"`
}

type KubeProxy struct {
	IPvs bool `json:"ipvs,omitempty" optional:"true"`
}

type KubeComponents struct {
	KubeProxy KubeProxy `json:"kubeProxy,omitempty" optional:"true"`
	Etcd      Etcd      `json:"etcd,omitempty" optional:"true"`
	Kubelet   Kubelet   `json:"kubelet,omitempty" yaml:"kubelet"`
	CNI       CNI       `json:"cni"`
}

type CRIType string

func (c CRIType) String() string {
	return string(c)
}

const (
	CRIDocker     CRIType = "docker"
	CRIContainerd CRIType = "containerd"
)

type ContainerRuntime struct {
	Type       CRIType    `json:"containerRuntimeType" enum:"docker|containerd"`
	Docker     Docker     `json:"docker,omitempty"`
	Containerd Containerd `json:"containerd,omitempty"`
}

type Docker struct {
	Version          string   `json:"version,omitempty" enum:"19.03.12"`
	DataRootDir      string   `json:"rootDir,omitempty"`
	InsecureRegistry []string `json:"insecureRegistry,omitempty"`
}

type Containerd struct {
	Version          string   `json:"version,omitempty" enum:"1.4.4"`
	DataRootDir      string   `json:"rootDir,omitempty"`
	InsecureRegistry []string `json:"insecureRegistry,omitempty"`
}
