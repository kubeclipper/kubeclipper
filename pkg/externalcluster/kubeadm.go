package externalcluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func init() {
	registerProvider(&KubeadmSpec{})
}

type KubeadmSpec struct {
	GeneralOption
	APIServer     string `json:"apiServer,omitempty"`
	KubeConfig    string `json:"kubeConfig"`
	ClusterName   string `json:"clusterName"`
	ConfigMapName string `json:"configMapName"`
}

func (in *KubeadmSpec) CreateClusterManage(cm *v1.ConfigMap) (ClusterManage, error) {
	kubeadm := &KubeadmSpec{}
	kubeadm.ConfigMapName = cm.Name
	if v, ok := cm.Data["apiServer"]; ok {
		kubeadm.APIServer = v
	}
	if v, ok := cm.Data["kubeConfig"]; ok {
		kubeadm.KubeConfig = v
	}

	kubeadm.ClusterName = cm.Name
	if kubeadm.ConfigMapName == "" {
		return nil, fmt.Errorf("cluster import configmap name cannot be empty")
	}
	if kubeadm.KubeConfig == "" {
		return nil, fmt.Errorf("cluster kubeconfig cannot be empty")
	}

	err := kubeadm.ReadEntity(cm)
	return kubeadm, err
}

func (in *KubeadmSpec) ClusterType() string {
	return ClusterKubeadm
}

func (in *KubeadmSpec) ClusterInfo() (*v1.Cluster, error) {
	c := &v1.Cluster{}
	c.Kind = "Cluster"
	c.APIVersion = "core.kubeclipper.io/v1"
	c.Labels = map[string]string{
		common.LabelTopologyRegion: in.NodeRegion}
	c.Annotations = map[string]string{
		common.AnnotationConfigMap:   in.ConfigMapName,
		common.AnnotationDescription: "kubeadm cluster import"}

	conf, err := base64.StdEncoding.DecodeString(in.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig decode failed: %v", err)
	}
	cliConfig, err := clientcmd.NewClientConfigFromBytes(conf)
	if err != nil {
		return nil, err
	}
	kubeConfig, err := cliConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	masterURL := ""
	if in.APIServer != "" {
		masterURL = fmt.Sprintf("https://%s:6443", in.APIServer)
		kubeConfig.APIPath = masterURL
	}

	cliSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("client set init failed: %v", err)
	}

	c.KubeConfig = []byte(kubeConfig.String())

	return c, in.completeClusterInfo(c, cliSet)
}

func (in *KubeadmSpec) completeClusterInfo(c *v1.Cluster, clientSet *kubernetes.Clientset) error {
	kubeadmConf, err := clientSet.CoreV1().ConfigMaps("kube-system").
		Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = in.completeKubeadm(clientSet, kubeadmConf, c)
	if err != nil {
		return err
	}
	nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	completeNodes(nodeList, c)

	return nil
}

func (in *KubeadmSpec) completeKubeadm(clientSet *kubernetes.Clientset, kubeadmConf *corev1.ConfigMap, c *v1.Cluster) error {
	clusterData := kubeadmConf.Data["ClusterConfiguration"]
	clusterConf := &ClusterConfiguration{}
	err := yaml.Unmarshal([]byte(clusterData), clusterConf)
	if err != nil {
		return err
	}

	proxyConf, err := clientSet.CoreV1().ConfigMaps("kube-system").
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

	c.Name = fmt.Sprintf("%s-%s", in.ClusterName, clusterConf.ClusterName)
	c.KubernetesVersion = clusterConf.KubernetesVersion
	c.CertSANs = clusterConf.APIServer.CertSANs
	c.LocalRegistry = clusterConf.ImageRepository
	c.KubeProxy = v1.KubeProxy{}
	c.Etcd.DataDir = clusterConf.Etcd.Local.DataDir
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

func completeNodes(nodeList *corev1.NodeList, c *v1.Cluster) {
	set := sets.NewString()
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
		set.Insert(criInfo[0])
		n := v1.WorkerNode{
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
	c.ContainerRuntime.Type = strings.Join(set.List(), ",")
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
