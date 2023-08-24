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
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=false

type Cluster struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// move offline to metadata annotation
	// Offline           bool   `json:"offline" optional:"true"`
	LocalRegistry     string         `json:"localRegistry,omitempty" optional:"true"`
	Masters           WorkerNodeList `json:"masters"`
	Workers           WorkerNodeList `json:"workers" optional:"true"`
	KubernetesVersion string         `json:"kubernetesVersion" enum:"v1.20.13"`
	// when generate cert,use GetAllCertSANs
	CertSANs          []string           `json:"certSANs,omitempty" optional:"true"`
	ExternalCaCert    string             `json:"externalCaCert,omitempty" optional:"true"`
	ExternalCaKey     string             `json:"externalCaKey,omitempty" optional:"true"`
	KubeProxy         KubeProxy          `json:"kubeProxy,omitempty" optional:"true"`
	Etcd              Etcd               `json:"etcd,omitempty" optional:"true"`
	Kubelet           Kubelet            `json:"kubelet,omitempty" optional:"true"`
	Networking        Networking         `json:"networking"`
	ContainerRuntime  ContainerRuntime   `json:"containerRuntime"`
	CNI               CNI                `json:"cni"`
	KubeConfig        []byte             `json:"kubeConfig,omitempty"`
	Addons            []Addon            `json:"addons" optional:"true"`
	Description       string             `json:"description,omitempty" optional:"true"`
	Status            ClusterStatus      `json:"status,omitempty" optional:"true"`
	PendingOperations []PendingOperation `json:"pendingOperations,omitempty" optional:"true"`
	FeatureGates      map[string]bool    `json:"featureGates,omitempty"`
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

// ClusterVersionsStatus contains information regarding the current and desired versions
// of the cluster control plane and worker nodes.
type ClusterVersionsStatus struct {
	// ControlPlane is the currently active cluster version. This can lag behind the apiserver
	// version if an update is currently rolling out.
	ControlPlane string `json:"controlPlane"`
	// Apiserver is the currently desired version of the kube-apiserver. During
	// upgrades across multiple minor versions (e.g. from 1.20 to 1.23), this will gradually
	// be increased by the update-controller until the desired cluster version (spec.version)
	// is reached.
	Apiserver string `json:"apiserver"`
	// ControllerManager is the currently desired version of the kube-controller-manager. This
	// field behaves the same as the apiserver field.
	ControllerManager string `json:"controllerManager"`
	// Scheduler is the currently desired version of the kube-scheduler. This field behaves the
	// same as the apiserver field.
	Scheduler string `json:"scheduler"`
}

type ClusterPhase string

// These are the valid phases of a project.
const (
	ClusterInstalling      ClusterPhase = "Installing"
	ClusterInstallFailed   ClusterPhase = "InstallFailed"
	ClusterRunning         ClusterPhase = "Running"
	ClusterUpdating        ClusterPhase = "Updating"
	ClusterUpdateFailed    ClusterPhase = "UpdateFailed"
	ClusterUpgrading       ClusterPhase = "Upgrading"
	ClusterUpgradeFailed   ClusterPhase = "UpgradeFailed"
	ClusterBackingUp       ClusterPhase = "BackingUp"
	ClusterRestoring       ClusterPhase = "Restoring"
	ClusterRestoreFailed   ClusterPhase = "RestoreFailed"
	ClusterTerminating     ClusterPhase = "Terminating"
	ClusterTerminateFailed ClusterPhase = "TerminateFailed"
)

const (
	ClusterFinalizer       = "finalizer.cluster.kubeclipper.io"
	CloudProviderFinalizer = "finalizer.cloudprovider.kubeclipper.io"
	OperationFinalizer     = "finalizer.operation.kubeclipper.io"
	BackupFinalizer        = "finalizer.backup.kubeclipper.io"
)

type ClusterStatus struct {
	// Phase is a description of the current cluster status, summarizing the various conditions,
	// possible active updates etc. This field is for informational purpose only and no logic
	// should be tied to the phase.
	// +optional
	Phase ClusterPhase `json:"phase,omitempty"`
	// Status ClusterStatusType `json:"status,omitempty"`
	// Versions contains information regarding the current and desired versions
	// of the cluster control plane and worker nodes.
	// +optional
	Versions ClusterVersionsStatus `json:"versions,omitempty"`
	// cluster component health status
	ComponentConditions []ComponentConditions `json:"componentConditions,omitempty"`

	Certifications []Certification `json:"certifications,omitempty"`
	// Registries all CRI registry
	Registries []RegistrySpec `json:"registries,omitempty"`
	// ControlPlane Health
	ControlPlaneHealth []ControlPlaneHealth `json:"controlPlaneHealth,omitempty"`
}

type ControlPlaneHealth struct {
	ID       string          `json:"id,omitempty"`
	Hostname string          `json:"hostname,omitempty"`
	Address  string          `json:"address,omitempty"`
	Status   ComponentStatus `json:"status,omitempty"`
}

func (c *Cluster) Offline() bool {
	if _, ok := c.Annotations[common.AnnotationOffline]; ok {
		return true
	}
	return false
}

// GetAllCertSANs if api server set externalIP,use it as certSans
func (c *Cluster) GetAllCertSANs() []string {
	list := c.CertSANs
	ip, ok := c.Labels[common.LabelExternalIP]
	if ok {
		list = append(list, ip)
	}
	domain, ok := c.Labels[common.LabelExternalDomain]
	if ok {
		list = append(list, domain)
	}
	return list
}

func (c *Cluster) Complete() {
	if c.Networking.ProxyMode == "" {
		c.Networking.ProxyMode = "ipvs"
	}
	if c.Etcd.DataDir == "" {
		c.Etcd.DataDir = "/var/lib/etcd"
	}
	if c.Kubelet.RootDir == "" {
		c.Kubelet.RootDir = "/var/lib/kubelet"
	}
	c.CNI.LocalRegistry = c.LocalRegistry
	c.CNI.CriType = c.ContainerRuntime.Type
	c.CNI.Offline = c.Offline()
	if common.IsKubeVersionGreater(c.KubernetesVersion, 126) {
		c.CNI.Namespace = "calico-system"
	} else {
		c.CNI.Namespace = "kube-system"
	}
}

type Certification struct {
	Name           string      `json:"name,omitempty"`
	CAName         string      `json:"caName"`
	ExpirationTime metav1.Time `json:"expirationTime,omitempty"`
}

type ComponentStatus string

const (
	ComponentHealthy     ComponentStatus = "Healthy"
	ComponentUnhealthy   ComponentStatus = "Unhealthy"
	ComponentUnKnown     ComponentStatus = "Unknown"
	ComponentUnsupported ComponentStatus = "Unsupported"
)

type ComponentConditions struct {
	Name     string          `json:"name"`
	Category string          `json:"category"`
	Status   ComponentStatus `json:"status"`
}

type Addon struct {
	Name    string               `json:"name"`
	Version string               `json:"version"`
	Config  runtime.RawExtension `json:"config"`
}

type IPFamily string

const (
	// IPFamilyIPv4 represents IPv4-only address family.
	IPFamilyIPv4 IPFamily = "IPv4"
	// IPFamilyDualStack represents dual-stack address family with IPv4 as the primary address family.
	IPFamilyDualStack IPFamily = "IPv4+IPv6"
)

// NetworkRanges represents ranges of network addresses.
type NetworkRanges struct {
	CIDRBlocks []string `json:"cidrBlocks"`
}

type Networking struct {
	// Optional: IP family used for cluster networking. Supported values are "IPv4" or "IPv4+IPv6".
	// Can be omitted / empty if pods and services network ranges are specified.
	// In that case it defaults according to the IP families of the provided network ranges.
	// If neither ipFamily nor pods & services network ranges are specified, defaults to "IPv4".
	// +optional
	IPFamily IPFamily `json:"ipFamily,omitempty"`
	// The network ranges from which service VIPs are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Services NetworkRanges `json:"services"`

	// The network ranges from which POD networks are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Pods NetworkRanges `json:"pods"`
	// Domain name for services.
	DNSDomain string `json:"dnsDomain"`
	// ProxyMode defines the kube-proxy mode ("ipvs" / "iptables" / "ebpf").
	// Defaults to "ipvs". "ebpf" disables kube-proxy and requires CNI support.
	ProxyMode     string `json:"proxyMode"`
	WorkerNodeVip string `json:"workerNodeVip" optional:"true"`
}

var (
	AllowedCNI = sets.NewString("calico")
)

type CNI struct {
	LocalRegistry string `json:"localRegistry" optional:"true"`
	// TODO: Cluster multiple cni plugins are not supported at this time
	Type      string  `json:"type" enum:"calico"`
	Version   string  `json:"version"`
	CriType   string  `json:"criType"`
	Offline   bool    `json:"offline"`
	Namespace string  `json:"namespace"`
	Calico    *Calico `json:"calico" optional:"true"`
}

type Calico struct {
	IPv4AutoDetection string `json:"IPv4AutoDetection" enum:"first-found|can-reach=DESTINATION|interface=INTERFACE-REGEX|skip-interface=INTERFACE-REGEX"`
	IPv6AutoDetection string `json:"IPv6AutoDetection" enum:"first-found|can-reach=DESTINATION|interface=INTERFACE-REGEX|skip-interface=INTERFACE-REGEX"`
	Mode              string `json:"mode" enum:"BGP|Overlay-IPIP-All|Overlay-IPIP-Cross-Subnet|Overlay-Vxlan-All|Overlay-Vxlan-Cross-Subnet|overlay"`
	IPManger          bool   `json:"IPManger" optional:"true"`
	MTU               int    `json:"mtu"`
}

type Etcd struct {
	DataDir string `json:"dataDir,omitempty" optional:"true"`
}

type Kubelet struct {
	RootDir    string `json:"rootDir" yaml:"rootDir"`
	NodeIP     string `json:"nodeIP" yaml:"nodeIP"`
	IPAsName   bool   `json:"ipAsName" yaml:"ipAsName"`
	ResolvConf string `json:"resolvConf" yaml:"resolvConf"`
}

type KubeProxy struct {
}

// container runtime define

var (
	AllowedCRIType = sets.NewString(CRIDocker, CRIContainerd)
)

type CRIType string

const (
	CRIDocker     = "docker"
	CRIContainerd = "containerd"
)

type ContainerRuntime struct {
	Type        string `json:"type" enum:"docker|containerd"`
	Version     string `json:"version,omitempty" enum:"1.4.4"`
	DataRootDir string `json:"rootDir,omitempty"`
	// Deprecated use Registries  insteadof
	InsecureRegistry []string `json:"insecureRegistry,omitempty"`
	// When updating
	Registries []CRIRegistry `json:"registries,omitempty"`
}

type CRIRegistry struct {
	InsecureRegistry string  `json:"insecureRegistry,omitempty"`
	RegistryRef      *string `json:"registryRef,omitempty"`
}

// taint define

type TaintEffect string

const (
	TaintEffectNoSchedule       TaintEffect = "NoSchedule"
	TaintEffectPreferNoSchedule TaintEffect = "PreferNoSchedule"
	TaintEffectNoExecute        TaintEffect = "NoExecute"
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

// WorkerNode define
type WorkerNode struct {
	ID               string            `json:"id"`
	Labels           map[string]string `json:"labels,omitempty"`
	Taints           []Taint           `json:"taints,omitempty"`
	ContainerRuntime ContainerRuntime  `json:"containerRuntime,omitempty"`
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

func (c *Cluster) GetAllNodes() sets.Set[string] {
	s := sets.New(c.Masters.GetNodeIDs()...)
	s.Insert(c.Workers.GetNodeIDs()...)
	return s
}
