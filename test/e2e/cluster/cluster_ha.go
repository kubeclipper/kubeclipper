package cluster

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/onsi/ginkgo"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Slow] [Serial] HA", func() {
	f := framework.NewDefaultFramework("ha")

	clusterName := "e2e-ha"
	nodeList := corev1.WorkerNodeList{}

	f.AddAfterEach("cleanup ha", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete ha cluster")
		err := f.Client.DeleteCluster(context.TODO(), clusterName)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clusterName, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: "!kubeclipper.io/nodeRole",
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) < 3 {
			framework.Failf("Not enough nodes to test")
		}
		nodeList = corev1.WorkerNodeList{
			{
				ID: nodes.Items[0].Name,
			},
			{
				ID: nodes.Items[1].Name,
			},
			{
				ID: nodes.Items[2].Name,
			},
		}
	})

	ginkgo.It("should create a HA minimal kubernetes cluster and ensure cluster is running.", func() {
		time.Sleep(5 * time.Second)
		ginkgo.By("create ha cluster")
		clus, err := f.Client.CreateCluster(context.TODO(), initHACluster(clusterName, nodeList))
		framework.ExpectNoError(err)
		if len(clus.Items) == 0 {
			framework.Failf("unexpected problem, cluster not be nil at this time")
		}

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clusterName, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cluster is healthy")
		err = cluster.WaitForClusterHealthy(f.Client, clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)

		ginkgo.By("ensure operation status is successful")

	})
})

func initHACluster(clusterName string, nodeList corev1.WorkerNodeList) *corev1.Cluster {
	return &corev1.Cluster{
		Provider: corev1.ProviderSpec{
			Name: corev1.ClusterKubeadm,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Annotations: map[string]string{
				common.AnnotationOffline: "true",
			},
		},
		Masters:           nodeList,
		Workers:           nil,
		KubernetesVersion: "v1.23.6",
		CertSANs:          nil,
		LocalRegistry:     "",
		ContainerRuntime: corev1.ContainerRuntime{
			Type:    corev1.CRIContainerd,
			Version: "1.6.4",
		},
		Networking: corev1.Networking{
			IPFamily:      corev1.IPFamilyIPv4,
			Services:      corev1.NetworkRanges{CIDRBlocks: []string{"10.96.0.0/16"}},
			Pods:          corev1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/24"}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},

		KubeProxy: corev1.KubeProxy{},
		Etcd:      corev1.Etcd{},
		CNI: corev1.CNI{
			LocalRegistry: "",
			Type:          "calico",
			Version:       "v3.21.2",
			Calico: &corev1.Calico{
				IPv4AutoDetection: "first-found",
				IPv6AutoDetection: "first-found",
				Mode:              "Overlay-Vxlan-All",
				IPManger:          true,
				MTU:               1440,
			},
		},
	}
}
