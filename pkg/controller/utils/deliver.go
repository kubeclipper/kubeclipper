package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
)

func ClusterServiceAccount(delivery service.CmdDelivery, nodeID string, action v1.StepAction) error {
	switch action {
	case v1.ActionInstall:
		action = "create"
	case v1.ActionUninstall:
		action = "delete"
	default:
		return fmt.Errorf("no support action(%v)", action)
	}
	sa := []string{"kubectl", string(action), "sa", "kc-server", "-n", "kube-system"}
	bind := []string{"kubectl", string(action), "clusterrolebinding", "kc-server", "--clusterrole=cluster-admin", "--serviceaccount=kube-system:kc-server"}

	logger.Debugf("The id of the execute command node is %s", nodeID)

	res, err := delivery.DeliverCmd(context.TODO(), nodeID, sa, 3*time.Second)
	if err != nil {
		return fmt.Errorf("%s ServiceAccount failed: %v. result: %s", action, err, string(res))
	}
	res, err = delivery.DeliverCmd(context.TODO(), nodeID, bind, 3*time.Second)
	if err != nil {
		return fmt.Errorf("%s clusterrolebinding failed: %v. result: %s", action, err, string(res))
	}

	return nil
}
