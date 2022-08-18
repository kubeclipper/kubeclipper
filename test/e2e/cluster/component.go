package cluster

import (
	"context"
	"encoding/json"
	"errors"
	v1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	nfsprovisioner "github.com/kubeclipper/kubeclipper/pkg/component/nfs"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = SIGDescribe("[Slow] [Serial] Install component", func() {

	f := framework.NewDefaultFramework("aio")
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup install component test aio cluster", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		clus, err := createClusterBeforeEach(f, initAIOCluster)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.It("install nfs component and ensure component is available", func() {
		ginkgo.By(" start install nfs")
		_, err := f.Client.InstallOrUninstallComponent(context.TODO(), clu.Name, initComponent(false))
		framework.ExpectNoError(err)

		ginkgo.By(" check nfs component condition ")
		err = cluster.WaitForClusterRunning(f.Client, clu.Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		clus, err := f.Client.DescribeCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		if !(clus.Items[0].Addons[0].Name == "nfs-provisioner") {
			framework.ExpectNoError(errors.New("component install failed"))
		}
	})

})

var _ = SIGDescribe("[Slow] [Serial] Uninstall component", func() {

	f := framework.NewDefaultFramework("aio")
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		if err != nil {
			ginkgo.By(" wait cluster delete not found ")
		}
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		clus, err := createClusterBeforeEach(f, initClusterWithPlugin)
		framework.ExpectNoError(err)
		if len(clus.Items) == 0 {
			framework.Failf("unexpected problem, cluster not be nil at this time")
		}
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.It(" uninstall nfs component ", func() {
		ginkgo.By(" start uninstall nfs ")
		clus, err := f.Client.InstallOrUninstallComponent(context.TODO(), clu.Name, initComponent(true))
		framework.ExpectNoError(err)
		if len(clus.Items) == 0 {
			framework.Failf("unexpected problem, cluster not be nil at this time")
		}
		ginkgo.By(" waiting for component uninstalled ")
		err = cluster.WaitForComponentNotFound(f.Client, clu.Name, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})

})

func initNFSComponent() *nfsprovisioner.NFSProvisioner {
	return &nfsprovisioner.NFSProvisioner{
		Namespace:        "kube-system",
		Replicas:         1,
		ServerAddr:       "172.20.151.105",
		SharedPath:       "/tmp/nfs/data",
		StorageClassName: "nfs-sc",
		IsDefault:        false,
		ReclaimPolicy:    "Delete",
		ArchiveOnDelete:  false,
	}
}

func initClusterWithPlugin(clusterName, nodeID string) *corev1.Cluster {
	nfs := initNFSComponent()
	nBytes, _ := json.Marshal(nfs)
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
		Masters: corev1.WorkerNodeList{
			{
				ID: nodeID,
			},
		},
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
		Addons: []corev1.Addon{
			{
				Name:    "nfs-provisioner",
				Version: "v1",
				Config: runtime.RawExtension{
					Raw: nBytes,
				},
			},
		},
	}
}

func initComponent(action bool) *v1.PatchComponents {
	nfs := initNFSComponent()
	nBytes, err := json.Marshal(nfs)
	framework.ExpectNoError(err, "marshal component error")
	return &v1.PatchComponents{
		Uninstall: action,
		Addons: []corev1.Addon{
			{
				Name:    "nfs-provisioner",
				Version: "v1",
				Config: runtime.RawExtension{
					Raw: nBytes,
				},
			},
		},
	}
}
