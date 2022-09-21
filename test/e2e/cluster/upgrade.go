package cluster

import (
	"context"
	"errors"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	apiv1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

const version = "v1.23.9"

var _ = SIGDescribe("[Slow] [Serial] Online cluster upgrade", func() {
	f := framework.NewDefaultFramework("aio")
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		clus, err := createClusterBeforeEach(f, "cluster-aio", initOnlineAIOCluster)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.It("upgrade online cluster and ensure cluster is upgraded", func() {
		ginkgo.By("upgrade cluster")
		err := f.Client.UpgradeCluster(context.TODO(), clu.Name, initUpgradeCluster(false, version))
		framework.ExpectNoError(err)

		ginkgo.By("wait cluster upgrade")
		err = cluster.WaitForUpgrade(f.Client, clu.Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster is upgraded")
		clus, err := f.Client.DescribeCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		if clus.Items[0].KubernetesVersion != version {
			framework.ExpectNoError(errors.New("upgrade online cluster failed"))
		}
	})
})

var _ = SIGDescribe("[Slow] [Serial] Offline cluster upgrade", func() {
	f := framework.NewDefaultFramework("aio")
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("create aio cluster")
		clus, err := createClusterBeforeEach(f, "cluster-aio", initAIOCluster)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()

		ginkgo.By("check resource exist")
		meta, err := f.Client.GetComponentMeta(context.TODO())
		framework.ExpectNoError(err)

		if !isResourceExist(meta) {
			framework.Failf("resource not found")
		}
	})

	ginkgo.It("upgrade offline cluster and ensure cluster is upgraded", func() {
		ginkgo.By("upgrade cluster")
		err := f.Client.UpgradeCluster(context.TODO(), clu.Name, initUpgradeCluster(true, version))
		framework.ExpectNoError(err)

		ginkgo.By("wait cluster upgrade")
		err = cluster.WaitForUpgrade(f.Client, clu.Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster is upgraded")
		clus, err := f.Client.DescribeCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)

		if clus.Items[0].KubernetesVersion != version {
			framework.ExpectNoError(errors.New("upgrade offline cluster failed"))
		}
	})
})

func initUpgradeCluster(offLine bool, version string) *apiv1.ClusterUpgrade {
	return &apiv1.ClusterUpgrade{
		Offline: offLine,
		Version: version,
	}
}

func initOnlineAIOCluster(clusterName string, nodeID []string) *corev1.Cluster {
	return &corev1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Masters: corev1.WorkerNodeList{
			{
				ID: nodeID[0],
			},
		},
		Workers:           nil,
		KubernetesVersion: "v1.23.6",
		CertSANs:          nil,
		LocalRegistry:     "172.20.150.138:5000",
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
			LocalRegistry: "172.20.150.138:5000",
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

func isResourceExist(meta *kc.ComponentMeta) bool {
	for _, addon := range meta.Addons {
		if addon.Name == "k8s" && addon.Arch == "amd64" && addon.Version == version {
			return true
		}
	}
	return false
}
