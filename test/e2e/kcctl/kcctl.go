package kcctl

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
	"github.com/kubeclipper/kubeclipper/test/framework/node"
)

var _ = SIGDescribe("[Serial]", func() {
	f := framework.NewDefaultFramework("kcctl")
	var nodeID, nodeIP string

	ginkgo.BeforeEach(func() {
		ginkgo.By("Check that there are enough available nodes")
		nodes, err := f.Client.ListNodes(context.TODO(), kc.Queries{
			Pagination:    query.NoPagination(),
			LabelSelector: fmt.Sprintf("!%s", common.LabelNodeDisable),
		})
		framework.ExpectNoError(err)
		nodeID = nodes.Items[0].Name
		nodeIP = nodes.Items[0].Status.Ipv4DefaultIP
	})

	//TODO: spoilt, need fix
	ginkgo.It("[Medium] [Kcctl] [Join] [Drain] should join node to kubeclipper platform and drain it", func() {
		ginkgo.By("drain node")
		err := drainAgentNode(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node not found")
		err = node.WaitForNodeNotFound(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
		framework.Logf("node %s drained\n", nodeIP)

		ginkgo.By("drain node")
		err = drainAgentNode(nodeID)
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
	ginkgo.It("[Medium] [Kcctl] [Drain] [Force] should drain node from kubeclipper platform force", func() {
		ginkgo.By("drain agent node")
		err := drainAgentNodeForce(nodeID)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for node is deleted")
		err = node.WaitForNodeNotFound(f.Client, nodeID, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By("join agent node")
		err = joinAgentNode(nodeIP)
		framework.ExpectNoError(err)

		ginkgo.By("wait for node join")
		err = cluster.WaitForJoinNode(f.Client, nodeIP, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})
})

func joinAgentNode(nodeIP string) error {
	return runCmd(fmt.Sprintf("kcctl join --agent %s", nodeIP))
}

func drainAgentNode(nodeID string) error {
	return runCmd(fmt.Sprintf("kcctl drain --agent %s", nodeID))
}

func drainAgentNodeForce(nodeID string) error {
	return runCmd(fmt.Sprintf("echo y | kcctl drain --agent %s -F", nodeID))
}

func runCmd(cmd string) error {
	ec := cmdutil.NewExecCmd(context.TODO(), "bash", "-c", cmd)
	return ec.Run()
}
