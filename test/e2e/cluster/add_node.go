package cluster

import (
	"context"
	"errors"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Slow] [Serial] Add a node to the cluster", func() {
	f := framework.NewDefaultFramework("aio")
	nodeID := ""
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup add node test aio cluster", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		clus, err := createClusterBeforeEach(f, "cluster-aio", initAIOCluster)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: "!kubeclipper.io/nodeRole",
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) == 0 {
			framework.Failf("Not enough nodes to test")
		}
		nodeID = nodes.Items[0].Name
	})

	ginkgo.It("add a node to the cluster and ensure the node is added", func() {
		ginkgo.By("add node")
		_, err := f.Client.AddOrRemoveNode(context.TODO(), initPatchNode(apiv1.NodesOperationAdd, nodeID), clu.Name)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clu.Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("check node is added")
		clus, err := f.Client.DescribeCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		if clus.Items[0].Workers[0].ID != nodeID {
			framework.ExpectNoError(errors.New("add node to cluster failed"))
		}
	})
})

var _ = SIGDescribe("[Slow] [Serial] Remove a node from the cluster", func() {
	f := framework.NewDefaultFramework("ha")
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup remove node test ha cluster", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete ha cluster")
		err := f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("create a cluster with a worker node")
		clus, err := createClusterWithWorkerNodeBeforeEach(f, "cluster-ha", initClusterWithWorkNode)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.It("remove a node from the cluster and ensure the node is removed", func() {
		ginkgo.By("remove node")
		_, err := f.Client.AddOrRemoveNode(context.TODO(), initPatchNode(apiv1.NodesOperationRemove, clu.Workers[0].ID), clu.Name)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clu.Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("check node is removed")
		clus, err := f.Client.DescribeCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		if len(clus.Items[0].Workers) != 0 {
			framework.ExpectNoError(errors.New("remove node failed"))
		}
	})
})

func initClusterWithWorkNode(clusterName string, nodeID []string) *corev1.Cluster {
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
				ID: nodeID[0],
			},
		},
		Workers: corev1.WorkerNodeList{
			{
				ID: nodeID[1],
			},
		},
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

func initPatchNode(operation, node string) *apiv1.PatchNodes {
	return &apiv1.PatchNodes{
		Operation: apiv1.NodesPatchOperation(operation),
		Role:      "worker",
		Nodes: corev1.WorkerNodeList{
			{
				ID: node,
			},
		},
	}
}

func createClusterWithWorkerNodeBeforeEach(f *framework.Framework, clusterName string, initial initfunc) (*kc.ClustersList, error) {
	var nodeIDs []string
	ginkgo.By("Check that there are enough available nodes")
	nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
		Pagination:    query.NoPagination(),
		LabelSelector: "!kubeclipper.io/nodeRole",
	})
	framework.ExpectNoError(err)
	if len(nodes.Items) < 2 {
		framework.Failf("Not enough nodes to test")
		return nil, err
	}
	for _, node := range nodes.Items {
		nodeIDs = append(nodeIDs, node.Name)
	}

	ginkgo.By("create cluster")
	clus, err := f.Client.CreateCluster(context.TODO(), initial(clusterName, nodeIDs))
	framework.ExpectNoError(err)
	if len(clus.Items) == 0 {
		framework.Failf("unexpected problem, cluster not be nil at this time")
		return nil, err
	}

	ginkgo.By("check cluster status is running")
	err = cluster.WaitForClusterRunning(f.Client, clus.Items[0].Name, f.Timeouts.ClusterInstall)
	framework.ExpectNoError(err)

	ginkgo.By("wait for cluster is healthy")
	err = cluster.WaitForClusterHealthy(f.Client, clus.Items[0].Name, f.Timeouts.ClusterInstallShort)
	framework.ExpectNoError(err)
	return clus, nil
}
