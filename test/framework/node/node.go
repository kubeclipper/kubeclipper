package node

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

type nodeCondition func(nodes []corev1.Node) (bool, error)

func waitForNodeCondition(c *kc.Client, nodeIP, conditionDesc string, timeout time.Duration, condition nodeCondition) error {
	framework.Logf("Waiting up to %v for node %q to be %q", timeout, nodeIP, conditionDesc)
	start := time.Now()
	err := wait.PollImmediate(framework.Poll, timeout, func() (bool, error) {
		nodes, apiErr := c.ListNodes(context.TODO(), kc.Queries{})
		if apiErr != nil {
			return framework.HandleWaitingAPIError(apiErr, true, "getting node %s", nodeIP)
		}
		ok, err := condition(nodes.Items)
		if err != nil {
			return false, err
		}
		framework.Logf("Node %q not registered, Elapsed: %v", nodeIP, time.Since(start))
		return ok, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) {
		return framework.TimeoutError(fmt.Sprintf("timed out while waiting for node %s to be %s", nodeIP, conditionDesc))
	}
	return framework.MaybeTimeoutError(err, "waiting for node %s to be %s", nodeIP, conditionDesc)
}

func WaitForNodeJoin(c *kc.Client, nodeIP string, timeout time.Duration) (string, error) {
	var nodeID string
	err := waitForNodeCondition(c, nodeIP, "join", timeout, func(nodes []corev1.Node) (bool, error) {
		for _, node := range nodes {
			if node.Status.Ipv4DefaultIP == nodeIP {
				nodeID = node.Name
				return true, nil
			}
		}
		return false, nil
	})
	return nodeID, err
}

func WaitForNodeNotFound(c *kc.Client, nodeID string, timeout time.Duration) error {
	return waitForNodeCondition(c, nodeID, "not found", timeout, func(nodes []corev1.Node) (bool, error) {
		for _, node := range nodes {
			if node.Name == nodeID {
				return false, nil
			}
		}
		return true, nil
	})
}

func IsNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
