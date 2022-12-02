package cluster

import (
	"context"
	"fmt"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

func DescribeCluster(c *kc.Client, clusterName string) (*corev1.Cluster, error) {
	clusterList, err := c.DescribeCluster(context.TODO(), clusterName)
	if err != nil {
		return nil, fmt.Errorf("cliet DescribeCluster: %w", err)
	}
	if len(clusterList.Items) == 0 {
		return nil, fmt.Errorf("cluster %s not exist", clusterName)
	}
	return &clusterList.Items[0], nil
}

func ExistComponent(c *kc.Client, clusterName, component string) (bool, error) {
	cluster, err := DescribeCluster(c, clusterName)
	if err != nil {
		return false, err
	}
	for _, addon := range cluster.Addons {
		if addon.Name == component {
			return true, nil
		}
	}
	return false, nil
}
