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

package kcctl

import (
	"context"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/e2e/cluster"
	"github.com/kubeclipper/kubeclipper/test/framework/node"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Medium] [Serial] Join node", func() {
	f := framework.NewDefaultFramework("node")
	var nodeIP, nodeID string

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
		nodeIP = nodes.Items[0].Status.Ipv4DefaultIP
		framework.Logf("target node ip:[%s] id:[%s]\n", nodeIP, nodeID)
	})

	ginkgo.It("join node", func() {
		ginkgo.By("drain node")
		err := drainAgentNode(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node not found")
		err = node.WaitForNodeNotFound(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
		framework.Logf("node %s drained\n", nodeIP)

		ginkgo.By("join node")
		err = joinAgentNode(nodeIP)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node join")
		nodeID, err = node.WaitForNodeJoin(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
		framework.Logf("node %s registered，id is:%s\n", nodeIP, nodeID)
	})
})

var _ = cluster.SIGDescribe("[Medium] [Serial] Drain node", func() {
	f := framework.NewDefaultFramework("node")
	var nodeIP, nodeID string

	f.AddAfterEach("join node", func(f *framework.Framework, failed bool) {
		ginkgo.By("join node")
		err := joinAgentNode(nodeIP)
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
		nodeIP = nodes.Items[0].Status.Ipv4DefaultIP
		framework.Logf("target node ip:[%s] id:[%s]\n", nodeIP, nodeID)
	})

	ginkgo.It("drain node", func() {
		ginkgo.By("drain node")
		err := drainAgentNode(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node not found")
		err = node.WaitForNodeNotFound(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
		framework.Logf("node %s drained\n", nodeIP)
	})
})
