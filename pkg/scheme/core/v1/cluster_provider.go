package v1

import (
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const (
	ClusterKubeadm string = "kubeadm"
	// TODO: other cluster type

	FedTypeAgent = "agent"
	FedTypePod   = "pod"
)

type ProviderSpec struct {
	// kubeadm/other
	Name string `json:"name"`
	// pod-agent/node-agent
	FedType string `json:"fedType"`
	// node region default value: "default"
	NodeRegion string      `json:"nodeRegion,omitempty"`
	SSHConfig  SSH         `json:"sshConfig"`
	Kubeadm    KubeadmSpec `json:"kubeadm,omitempty"`
	// TODO: other cluster type
}

type SSH struct {
	User              string         `json:"user" yaml:"user,omitempty"`
	Password          string         `json:"password,omitempty" yaml:"password,omitempty"`
	PkDataEncode      string         `json:"pkDataEncode,omitempty" yaml:"pkDataEncode,omitempty"`
	ConnectionTimeout *time.Duration `json:"connectionTimeout,omitempty" yaml:"connectionTimeout,omitempty"`
}

func (s *SSH) Convert() *sshutils.SSH {
	return &sshutils.SSH{
		User:         s.User,
		Password:     s.Password,
		PkDataEncode: s.PkDataEncode,
	}
}

type KubeadmSpec struct {
	APIServer  string `json:"apiServer,omitempty"`
	KubeConfig string `json:"kubeConfig"`
}

type ClusterConfiguration struct {
	// Etcd holds configuration for etcd.
	Etcd KubeEtcd `json:"etcd,omitempty"`

	// Networking holds configuration for the networking topology of the cluster.
	Networking Network `json:"networking,omitempty"`

	// KubernetesVersion is the target version of the control plane.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// APIServer contains extra settings for the API server control plane component
	APIServer APIServer `json:"apiServer,omitempty"`

	// ImageRepository sets the container registry to pull images from.
	ImageRepository string `json:"imageRepository,omitempty"`

	// The cluster name
	ClusterName string `json:"clusterName,omitempty"`
}

type KubeEtcd struct {
	// Local provides configuration knobs for configuring the local etcd instance
	// Local and External are mutually exclusive
	Local *LocalEtcd `json:"local,omitempty"`
}

type LocalEtcd struct {
	// DataDir is the directory etcd will place its data.
	// Defaults to "/var/lib/etcd".
	DataDir string `json:"dataDir"`
}

type Network struct {
	// ServiceSubnet is the subnet used by k8s services. Defaults to "10.96.0.0/12".
	ServiceSubnet string `json:"serviceSubnet,omitempty"`
	// PodSubnet is the subnet used by pods.
	PodSubnet string `json:"podSubnet,omitempty"`
	// DNSDomain is the dns domain used by k8s services. Defaults to "cluster.local".
	DNSDomain string `json:"dnsDomain,omitempty"`
}

type APIServer struct {
	// CertSANs sets extra Subject Alternative Names for the API Server signing cert.
	CertSANs []string `json:"certSANs,omitempty"`
}
