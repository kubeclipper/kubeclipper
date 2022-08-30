package clustermanage

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

type KubeadmSpec v1.KubeadmSpec

func (in *KubeadmSpec) CreateClusterManage(spec v1.ProviderSpec) ClusterManage {
	kubeadm := KubeadmSpec(spec.Kubeadm)
	return &kubeadm
}

func (in *KubeadmSpec) ClusterType() string {
	return ClusterKubeadm
}

func (in *KubeadmSpec) ClusterInfo(name string, provider v1.ProviderSpec) (*v1.Cluster, map[string]string, error) {
	c := new(v1.Cluster)
	c.Provider = provider
	c.Name = name

	conf, err := base64.StdEncoding.DecodeString(in.KubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("kubeconfig decode failed: %v", err)
	}
	cliConfig, err := clientcmd.NewClientConfigFromBytes(conf)
	if err != nil {
		return nil, nil, err
	}
	kubeConfig, err := cliConfig.ClientConfig()
	if err != nil {
		return nil, nil, err
	}

	masterURL := ""
	if in.APIServer != "" {
		masterURL = fmt.Sprintf("https://%s:6443", in.APIServer)
		kubeConfig.APIPath = masterURL
	}

	cliSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("client set init failed: %v", err)
	}

	c.KubeConfig = []byte(kubeConfig.String())

	ipMap, err := in.completeClusterInfo(c, cliSet)

	return c, ipMap, err
}

func (in *KubeadmSpec) completeClusterInfo(c *v1.Cluster, clientSet *kubernetes.Clientset) (map[string]string, error) {
	kubeadmConf, err := clientSet.CoreV1().ConfigMaps("kube-system").
		Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = completeKubeadm(clientSet, kubeadmConf, c)
	if err != nil {
		return nil, err
	}
	nodeList, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return completeNodes(nodeList, c), nil
}

func completeKubeadm(clientSet *kubernetes.Clientset, kubeadmConf *corev1.ConfigMap, c *v1.Cluster) error {
	clusterData := kubeadmConf.Data["ClusterConfiguration"]
	clusterConf := &v1.ClusterConfiguration{}
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

	c.Kind = "cluster"
	c.APIVersion = "core.kubeclipper.io/v1"
	c.Name = fmt.Sprintf("%s-%s", c.Name, clusterConf.ClusterName)
	c.Labels = make(map[string]string)
	c.Labels[common.LabelTopologyRegion] = c.Provider.NodeRegion
	c.Provider.Name = v1.ClusterKubeadm
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
	c.Annotations = make(map[string]string)
	c.Annotations[common.AnnotationDescription] = "kubeadm cluster import"

	return nil
}

func completeNodes(nodeList *corev1.NodeList, c *v1.Cluster) (nodeMap map[string]string) {
	nodeMap = make(map[string]string)
	set := sets.NewString()
	c.Masters = v1.WorkerNodeList{}
	c.Workers = v1.WorkerNodeList{}
	for _, node := range nodeList.Items {
		var taints []v1.Taint
		nodeID := uuid.New().String()
		for _, taint := range node.Spec.Taints {
			taints = append(taints, v1.Taint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: v1.TaintEffect(taint.Effect),
			})
		}
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodeMap[addr.Address] = nodeID
				break
			}
		}
		// example: "containerd://1.6.4"
		criInfo := strings.Split(node.Status.NodeInfo.ContainerRuntimeVersion, "://")
		set.Insert(criInfo[0])
		n := v1.WorkerNode{
			ID:     nodeID,
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

	return
}
