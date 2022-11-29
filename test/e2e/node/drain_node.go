package node

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
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
		err := drainAgentNode(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for node is deleted")
		err = cluster.WaitForNodeNotFound(f.Client, nodeID, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})
})

func drainAgentNode(nodeID string) error {
	return runCmd(fmt.Sprintf("echo y | kcctl drain --agent %s -F", nodeID))
}

func joinAgentNode(nodeIP string) error {
	return runCmd(fmt.Sprintf("kcctl join --agent %s", nodeIP))
}

func runCmd(cmd string) error {
	ec := cmdutil.NewExecCmd(context.TODO(), "bash", "-c", cmd)
	return ec.Run()
}
