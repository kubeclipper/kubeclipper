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
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/test/e2e/project"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Slow] [Serial] Update Cert", func() {
	f := framework.NewDefaultFramework("aio")

	var nodeID []string
	clusterName := "e2e-aio"
	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		err := f.Client.DeleteClusterWithQuery(context.TODO(), clusterName, nil)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clusterName, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)

		ginkgo.By("delete e2e project")
		err = project.DeleteProject(f)
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

		for _, item := range nodes.Items {
			nodeID = append(nodeID, item.Name)
		}

		ginkgo.By("create e2e project")
		err = project.CreateProject(f)
		framework.ExpectNoError(err)

	})

	ginkgo.It("should create a aio minimal kubernetes cluster and update cert when ensure cluster is running.", func() {
		ginkgo.By("create aio cluster")

		clus, err := f.Client.CreateCluster(context.TODO(), initAIOCluster(clusterName, nodeID))
		framework.ExpectNoError(err)

		if len(clus.Items) == 0 {
			framework.Failf("unexpected problem, cluster not be nil at this time")
		}

		ginkgo.By("check cluster status is running")
		clusterName = fmt.Sprintf("%s-%s", project.DefaultE2EProject, clusterName)
		err = cluster.WaitForClusterRunning(f.Client, clusterName, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cluster is healthy")
		err = cluster.WaitForClusterHealthy(f.Client, clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cert init")
		err = cluster.WaitForCertInit(f.Client, clusterName, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By("get cert expiration time")
		t, err := GetCertExpirationTime(f, clusterName)
		framework.ExpectNoError(err)

		framework.Logf("current cert expiration time: %v", t.Format(time.RFC3339))
		ginkgo.By("update cert")
		err = UpdateCert(f, clusterName)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cert update")
		err = cluster.WaitForCertUpdated(f.Client, clusterName, *t, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})
})

func UpdateCert(f *framework.Framework, clusterName string) error {
	_, err := f.Client.UpdateCert(context.TODO(), clusterName)
	return err
}

func GetCertExpirationTime(f *framework.Framework, clusterName string) (*metav1.Time, error) {
	describeCluster, err := f.Client.DescribeCluster(context.TODO(), clusterName)
	if err != nil {
		return nil, err
	}
	if len(describeCluster.Items) == 0 {
		return nil, fmt.Errorf("unexpected problem, cluster %s not found", clusterName)
	}
	for _, certification := range describeCluster.Items[0].Status.Certifications {
		if !strings.Contains(certification.Name, "ca") {
			return &certification.ExpirationTime, nil
		}
	}
	return nil, fmt.Errorf("unexpected problem, cluster %s ca expiration time not exist", clusterName)
}
