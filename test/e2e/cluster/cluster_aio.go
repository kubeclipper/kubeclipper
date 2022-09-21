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

package cluster

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Slow] [Serial] AIO", func() {
	f := framework.NewDefaultFramework("aio")

	var nodeID []string
	clusterName := "e2e-aio"

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
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
			LabelSelector: fmt.Sprintf("!%s", common.LabelNodeRole),
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) == 0 {
			framework.Failf("Not enough nodes to test")
		}
		for _, node := range nodes.Items {
			nodeID = append(nodeID, node.Name)
		}
	})

	ginkgo.It("should create a AIO minimal kubernetes cluster and ensure cluster is running.", func() {
		ginkgo.By("create aio cluster")

		clus, err := f.Client.CreateCluster(context.TODO(), initAIOCluster(clusterName, nodeID))
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
	})
})

func initAIOCluster(clusterName string, nodeID []string) *corev1.Cluster {
	// TODO: make version be parameter
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

type initfunc func(clusterName string, nodeID []string) *corev1.Cluster

func createClusterBeforeEach(f *framework.Framework, clusterName string, initial initfunc) (*kc.ClustersList, error) {
	var nodeID []string
	ginkgo.By("Check that there are enough available nodes")
	nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
		Pagination:    query.NoPagination(),
		LabelSelector: "!kubeclipper.io/nodeRole",
	})
	framework.ExpectNoError(err)
	if len(nodes.Items) == 0 {
		framework.Failf("Not enough nodes to test")
		return nil, err
	}
	for _, node := range nodes.Items {
		nodeID = append(nodeID, node.Name)
	}

	ginkgo.By("create aio cluster")
	clus, err := f.Client.CreateCluster(context.TODO(), initial(clusterName, nodeID))
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
