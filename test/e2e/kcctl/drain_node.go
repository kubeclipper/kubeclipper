package kcctl

import (
	"context"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
	fnode "github.com/kubeclipper/kubeclipper/test/framework/node"
)

var _ = SIGDescribe("[Fast] [Serial] Force drain node", func() {
	f := framework.NewDefaultFramework("node")
	nodeID := ""
	nodeIP := ""

	f.AddAfterEach("join agent node", func(f *framework.Framework, failed bool) {
		ginkgo.By("join agent node")
		err := joinAgentNode(nodeIP)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node join")
		err = cluster.WaitForJoinNode(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By("check that there are enough agent nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination: query.NoPagination(),
		})
		framework.ExpectNoError(err)
		if len(nodes.Items) == 0 {
			framework.Failf("Not enough agent nodes to test")
		}
		nodeID = nodes.Items[0].Name
		nodeIP = nodes.Items[0].Status.Ipv4DefaultIP
	})

	ginkgo.It("force drain agent node and ensure the node is deleted", func() {
		ginkgo.By("drain agent node")
		err := drainAgentNodeForce(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for node is deleted")
		err = fnode.WaitForNodeNotFound(f.Client, nodeID, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})
})
