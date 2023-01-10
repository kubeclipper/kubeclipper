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
	"time"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	apierror "github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

type Setter func(cluster *corev1.Cluster)

func SetContainerdRuntime() Setter {
	return func(c *corev1.Cluster) {
		c.ContainerRuntime = corev1.ContainerRuntime{
			Type:    corev1.CRIContainerd,
			Version: "1.6.4",
		}
	}
}

func SetDockerRuntime() Setter {
	return func(c *corev1.Cluster) {
		c.ContainerRuntime = corev1.ContainerRuntime{
			Type:    corev1.CRIDocker,
			Version: "20.10.20",
		}
	}
}

func SetClusterName(name string) Setter {
	return func(c *corev1.Cluster) {
		c.Name = name
	}
}

func SetClusterVersion(version string) Setter {
	return func(c *corev1.Cluster) {
		c.KubernetesVersion = version
	}
}

func SetClusterBackupPoint(bp string) Setter {
	return func(c *corev1.Cluster) {
		if c.Labels == nil {
			c.Labels = make(map[string]string, 0)
		}
		c.Labels[common.LabelBackupPoint] = bp
	}
}

func SetClusterNodes(masters []string, workers []string) Setter {
	return func(c *corev1.Cluster) {
		if len(masters) > 0 {
			c.Masters = corev1.WorkerNodeList{}
		}
		if len(workers) > 0 {
			c.Workers = corev1.WorkerNodeList{}
		}
		for _, id := range masters {
			c.Masters = append(c.Masters, corev1.WorkerNode{
				ID: id,
			})
		}
		for _, id := range workers {
			c.Workers = append(c.Workers, corev1.WorkerNode{
				ID: id,
			})
		}
	}
}

func InitClusterWithSetter(c *corev1.Cluster, setter []Setter) {
	for _, f := range setter {
		f(c)
	}
}

func initCluster() *corev1.Cluster {
	return &corev1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       corev1.KindCluster,
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "e2e-cluster",
			Annotations: map[string]string{
				common.AnnotationOffline: "true",
			},
		},
		Masters:           corev1.WorkerNodeList{},
		Workers:           corev1.WorkerNodeList{},
		KubernetesVersion: "v1.23.6",
		CertSANs:          nil,
		LocalRegistry:     "",
		ContainerRuntime: corev1.ContainerRuntime{
			Type:    corev1.CRIContainerd,
			Version: "1.6.4",
		},
		Networking: corev1.Networking{
			IPFamily:      corev1.IPFamilyIPv4,
			Services:      corev1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterServiceSubnet}},
			Pods:          corev1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterPodSubnet}},
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

func beforeEachCheckNodeEnough(f *framework.Framework, min int) []string {
	ginkgo.By("Check that there are enough available nodes")
	nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
		Pagination:    query.NoPagination(),
		LabelSelector: "!kubeclipper.io/nodeRole",
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(nodes.Items) >= min, true, fmt.Sprintf("node count must more than %d", min))
	var ids []string
	for _, node := range nodes.Items {
		ids = append(ids, node.Name)
	}
	return ids
}

func beforeEachCreateCluster(f *framework.Framework, c *corev1.Cluster) func() {
	return func() {
		clus, err := f.Client.CreateCluster(context.TODO(), c)
		framework.ExpectNoError(err)
		framework.ExpectNotEqual(len(clus.Items), 0, "unexpected problem, cluster not be nil at this time")

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clus.Items[0].Name, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cluster is healthy")
		err = cluster.WaitForClusterHealthy(f.Client, clus.Items[0].Name, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)
	}
}

func afterEachDeleteCluster(f *framework.Framework, clusterName *string) func() {
	return func() {
		ginkgo.By("delete cluster")
		err := retryOperation(func() error {
			delErr := f.Client.DeleteCluster(context.TODO(), *clusterName)
			if apierror.IsNotFound(delErr) {
				// cluster not exist
				return nil
			}
			return delErr
		}, 2)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, *clusterName, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	}
}

type fn func() error

func retryOperation(f fn, times int) error {
	var err error
	for i := 0; i < times; i++ {
		if err = f(); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return err
}
