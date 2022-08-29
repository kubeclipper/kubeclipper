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

package node

import (
	"context"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	cluster2 "github.com/kubeclipper/kubeclipper/test/e2e/cluster"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = cluster2.SIGDescribe("[Fast] [Serial] Disable node", func() {
	f := framework.NewDefaultFramework("node")
	nodeID := ""

	f.AddAfterEach("enable the disable node", func(f *framework.Framework, failed bool) {
		ginkgo.By("enable node")
		err := f.Client.EnableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: fmt.Sprintf("!%s", common.LabelNodeDisable),
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) == 0 {
			framework.Failf("Not enough enabled nodes to test")
		}
		nodeID = nodes.Items[0].Name
	})

	ginkgo.It("disable a node and ensure the node is disabled", func() {
		ginkgo.By("disable node")
		err := f.Client.DisableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("check node is disabled")
		nodeList, err := f.Client.DescribeNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
		if _, ok := nodeList.Items[0].Labels[common.LabelNodeDisable]; !ok {
			framework.Failf("Fail to disable node")
		} else {
			ginkgo.By("node is disabled")
		}
	})
})

var _ = cluster2.SIGDescribe("[Fast] [Serial] Enable node", func() {
	f := framework.NewDefaultFramework("node")
	nodeID := ""

	f.AddAfterEach("disable the enabled node", func(f *framework.Framework, failed bool) {
		ginkgo.By("disable node")
		err := f.Client.DisableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: fmt.Sprintf("%s", common.LabelNodeDisable),
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) == 0 {
			framework.Failf("Not enough disabled nodes to test")
		}
		nodeID = nodes.Items[0].Name

		if _, ok := nodes.Items[0].Labels[common.LabelNodeDisable]; !ok {
			ginkgo.By("Make sure the node is disabled")
			err := f.Client.DisableNode(context.TODO(), nodeID)
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("enable a node and ensure the node is enabled", func() {
		ginkgo.By("enable node")
		err := f.Client.EnableNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("check node is enabled")
		nodeList, err := f.Client.DescribeNode(context.TODO(), nodeID)
		framework.ExpectNoError(err)
		if _, ok := nodeList.Items[0].Labels[common.LabelNodeDisable]; !ok {
			ginkgo.By("node is enabled")
		} else {
			framework.Failf("Fail to enabled node")
		}
	})
})
