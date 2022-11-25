package cluster

import (
	"context"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
)

var _ = SIGDescribe("[Slow] [Serial] Cri docker registry", func() {
	f := framework.NewDefaultFramework("aio")

	clu := &corev1.Cluster{}
	registry := &corev1.Registry{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("force delete aio cluster")
		queryString := url.Values{}
		queryString.Set(query.ParameterForce, "true")
		err := f.Client.DeleteClusterWithQuery(context.TODO(), clu.Name, queryString)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)

		ginkgo.By("delete registry")
		err = f.Client.DeleteRegistry(context.TODO(), registry.Name)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("create registry")
		registries, err := f.Client.CreateRegistry(context.TODO(), initRegistry())
		framework.ExpectNoError(err)
		registry = registries.Items[0].DeepCopy()

		ginkgo.By("create aio cluster")
		clusters, err := createClusterBeforeEach(f, "cluster-registry-aio", initClusterWithDockerCRI)
		framework.ExpectNoError(err)
		clu = clusters.Items[0].DeepCopy()
	})

	ginkgo.It("create registry and ensure registry is available", func() {
		clu.ContainerRuntime.Registries = []corev1.CRIRegistry{
			{
				RegistryRef: &registry.Name,
			},
		}

		ginkgo.By("update registries to the cluster")
		err := f.Client.UpdateCluster(context.TODO(), clu)
		framework.ExpectNoError(err)

		ginkgo.By("check registries successful")
		err = cluster.WaitForCriRegistry(f.Client, clu.Name, f.Timeouts.CommonTimeout, []string{registry.Host})
		framework.ExpectNoError(err)
	})
})

var _ = SIGDescribe("[Slow] [Serial] Cri containerd registry", func() {
	f := framework.NewDefaultFramework("aio")

	clu := &corev1.Cluster{}
	registry := &corev1.Registry{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("force delete aio cluster")
		queryString := url.Values{}
		queryString.Set(query.ParameterForce, "true")
		err := f.Client.DeleteClusterWithQuery(context.TODO(), clu.Name, queryString)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)

		ginkgo.By("delete registry")
		err = f.Client.DeleteRegistry(context.TODO(), registry.Name)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("create registry")
		registries, err := f.Client.CreateRegistry(context.TODO(), initRegistry())
		framework.ExpectNoError(err)
		registry = registries.Items[0].DeepCopy()

		ginkgo.By("create aio cluster")
		clusters, err := createClusterBeforeEach(f, "cluster-registry-aio", initClusterWithContainerdCRI)
		framework.ExpectNoError(err)
		clu = clusters.Items[0].DeepCopy()
	})

	ginkgo.It("create registry and ensure registry is available", func() {
		clu.ContainerRuntime.Registries = []corev1.CRIRegistry{
			{
				RegistryRef: &registry.Name,
			},
		}

		ginkgo.By("update registries to the cluster")
		err := f.Client.UpdateCluster(context.TODO(), clu)
		framework.ExpectNoError(err)

		ginkgo.By("check registries successful")
		err = cluster.WaitForCriRegistry(f.Client, clu.Name, f.Timeouts.CommonTimeout, []string{registry.Host})
		framework.ExpectNoError(err)
	})
})

func initRegistry() *corev1.Registry {
	return &corev1.Registry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Registry",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubeclipper.io",
		},
		RegistrySpec: corev1.RegistrySpec{
			Scheme:     "https",
			Host:       "registry.kubeclipper.io",
			SkipVerify: true,
		},
	}
}

func initClusterWithDockerCRI(clusterName string, nodeID []string) *corev1.Cluster {
	return &corev1.Cluster{
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
				ID: nodeID[0],
			},
		},
		Workers:           nil,
		KubernetesVersion: "v1.23.6",
		CertSANs:          nil,
		LocalRegistry:     "",
		ContainerRuntime: corev1.ContainerRuntime{
			Type:    corev1.CRIDocker,
			Version: "20.10.20",
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
			Version:       "v3.22.4",
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

func initClusterWithContainerdCRI(clusterName string, nodeID []string) *corev1.Cluster {
	return &corev1.Cluster{
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
				ID: nodeID[0],
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
			Version:       "v3.22.4",
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
