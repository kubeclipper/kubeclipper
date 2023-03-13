package kubeadm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Wrapper struct {
	ProviderName string `json:"providerName"`
	APIEndpoint  string `json:"apiEndpoint,omitempty"`
	KubeConfig   string `json:"kubeConfig"`
	Region       string `json:"region"`
	ClusterName  string `json:"clusterName"`
	KubeCli      *kubernetes.Clientset
}

func NewKubeClient(config runtime.RawExtension) (*kubernetes.Clientset, error) {
	c, err := ParseConf(config)
	if err != nil {
		return nil, err
	}
	conf, err := base64.StdEncoding.DecodeString(c.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig decode failed: %v", err)
	}

	cliConfig, err := clientcmd.NewClientConfigFromBytes(conf)
	if err != nil {
		return nil, fmt.Errorf("client config(bytes) create failed: %v", err)
	}
	kubeConfig, err := cliConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("client config failed: %v", err)
	}

	if c.APIEndpoint != "" {
		kubeConfig.APIPath = c.APIEndpoint
	}

	cliSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("client set init failed: %v. current kube-api-endpoint is %s", err, c.APIEndpoint)
	}
	return cliSet, nil
}

func ParseConf(config runtime.RawExtension) (*Config, error) {
	data, err := config.MarshalJSON()
	if err != nil {
		return nil, err
	}
	kubeadm := new(Config)
	if err = json.Unmarshal(data, kubeadm); err != nil {
		return nil, err
	}
	return kubeadm, nil
}

func (w *Wrapper) ClusterInfo() (*v1.Cluster, error) {
	c := &v1.Cluster{}
	c.Kind = "cluster"
	c.APIVersion = "core.kubeclipper.io/v1"
	c.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			common.LabelTopologyRegion:      w.Region,
			common.LabelClusterProviderType: ProviderKubeadm,
			common.LabelClusterProviderName: w.ProviderName,
		},
		Annotations: map[string]string{
			common.AnnotationDescription: "kubeadm cluster import",
		},
	}

	c.Status = v1.ClusterStatus{
		Phase: v1.ClusterRunning,
	}

	kubeconfig, err := base64.StdEncoding.DecodeString(w.KubeConfig)
	if err != nil {
		return nil, err
	}
	c.KubeConfig = kubeconfig

	return c, w.completeClusterInfo(c)
}

func (w *Wrapper) completeClusterInfo(c *v1.Cluster) error {
	kubeadmConf, err := w.KubeCli.CoreV1().ConfigMaps("kube-system").
		Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = w.completeKubeadm(kubeadmConf, c)
	if err != nil {
		return err
	}
	nodeList, err := w.KubeCli.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	w.completeNodes(nodeList, c)

	return nil
}

func (w *Wrapper) completeKubeadm(kubeadmConf *corev1.ConfigMap, c *v1.Cluster) error {
	clusterData := kubeadmConf.Data["ClusterConfiguration"]
	clusterConf := &ClusterConfiguration{}
	err := yaml.Unmarshal([]byte(clusterData), clusterConf)
	if err != nil {
		return err
	}

	proxyConf, err := w.KubeCli.CoreV1().ConfigMaps("kube-system").
		Get(context.TODO(), "kube-proxy", metav1.GetOptions{})
	if err != nil {
		return err
	}
	proxyMode := "iptables"
	if strings.Contains(proxyConf.Data["config.conf"], "mode: ipvs") {
		proxyMode = "ipvs"
	}
	serviceSubnet := strings.Split(clusterConf.Networking.ServiceSubnet, ",")
	podSubnet := strings.Split(clusterConf.Networking.PodSubnet, ",")
	ipFamily := v1.IPFamilyIPv4
	if len(serviceSubnet) >= 2 && len(podSubnet) >= 2 {
		ipFamily = v1.IPFamilyDualStack
	}

	c.Name = w.ClusterName
	c.Annotations[common.AnnotationActualName] = clusterConf.ClusterName
	c.KubernetesVersion = clusterConf.KubernetesVersion
	c.CertSANs = clusterConf.APIServer.CertSANs
	c.KubeProxy = v1.KubeProxy{}
	if clusterConf.Etcd.Local == nil {
		c.Etcd.DataDir = "External"
	} else {
		c.Etcd.DataDir = clusterConf.Etcd.Local.DataDir
	}
	c.Kubelet.RootDir = ""
	c.Networking.IPFamily = ipFamily
	c.Networking.Services = v1.NetworkRanges{CIDRBlocks: serviceSubnet}
	c.Networking.Pods = v1.NetworkRanges{CIDRBlocks: podSubnet}
	c.Networking.DNSDomain = clusterConf.Networking.DNSDomain
	c.Networking.ProxyMode = proxyMode
	c.Networking.WorkerNodeVip = ""
	c.CNI = v1.CNI{}
	c.Addons = []v1.Addon{}

	return nil
}

func (w *Wrapper) completeNodes(nodeList *corev1.NodeList, c *v1.Cluster) {
	c.Masters = v1.WorkerNodeList{}
	c.Workers = v1.WorkerNodeList{}
	for _, node := range nodeList.Items {
		var taints []v1.Taint
		nodeIP := ""
		for _, taint := range node.Spec.Taints {
			taints = append(taints, v1.Taint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: v1.TaintEffect(taint.Effect),
			})
		}
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodeIP = addr.Address
				break
			}
		}
		// example: "containerd://1.6.4"
		criInfo := strings.Split(node.Status.NodeInfo.ContainerRuntimeVersion, "://")
		n := v1.WorkerNode{
			// Since the kc-agent service has not yet started,
			// the node ID cannot be obtained, so this field is temporarily put into the IP instead,
			// and the IP of this field will be replaced with the ID during the execution of the subsequent process
			ID:     nodeIP,
			Labels: node.ObjectMeta.Labels,
			Taints: taints,
			ContainerRuntime: v1.ContainerRuntime{
				Type:    criInfo[0],
				Version: criInfo[1],
			},
		}

		if _, ok := node.ObjectMeta.Labels["node-role.kubernetes.io/control-plane"]; ok {
			c.Masters = append(c.Masters, n)
			continue
		}
		if _, ok := node.ObjectMeta.Labels["node-role.kubernetes.io/master"]; ok {
			c.Masters = append(c.Masters, n)
			continue
		}
		c.Workers = append(c.Workers, n)
	}
	c.ContainerRuntime = c.Masters[0].ContainerRuntime
}

type ClusterConfiguration struct {
	Kind string `yaml:"kind,omitempty"`

	// Etcd holds configuration for etcd.
	Etcd KubeEtcd `yaml:"etcd,omitempty"`

	// Networking holds configuration for the networking topology of the cluster.
	Networking Network `yaml:"networking,omitempty"`

	// KubernetesVersion is the target version of the control plane.
	KubernetesVersion string `yaml:"kubernetesVersion,omitempty"`

	// APIServer contains extra settings for the API server control plane component
	APIServer APIServer `yaml:"apiServer,omitempty"`

	// ImageRepository sets the container registry to pull images from.
	ImageRepository string `yaml:"imageRepository,omitempty"`

	// The cluster name
	ClusterName string `yaml:"clusterName,omitempty"`
}

type KubeEtcd struct {
	// Local provides configuration knobs for configuring the local etcd instance
	// Local and External are mutually exclusive
	Local *LocalEtcd `yaml:"local,omitempty"`
}

type LocalEtcd struct {
	// DataDir is the directory etcd will place its data.
	// Defaults to "/var/lib/etcd".
	DataDir string `yaml:"dataDir"`
}

type Network struct {
	// ServiceSubnet is the subnet used by k8s services. Defaults to "10.96.0.0/12".
	ServiceSubnet string `yaml:"serviceSubnet,omitempty"`
	// PodSubnet is the subnet used by pods.
	PodSubnet string `yaml:"podSubnet,omitempty"`
	// DNSDomain is the dns domain used by k8s services. Defaults to "cluster.local".
	DNSDomain string `yaml:"dnsDomain,omitempty"`
}

type APIServer struct {
	// CertSANs sets extra Subject Alternative Names for the API Server signing cert.
	CertSANs []string `yaml:"certSANs,omitempty"`
}
