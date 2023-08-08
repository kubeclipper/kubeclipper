package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	apierror "github.com/kubeclipper/kubeclipper/pkg/errors"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

type clusterCondition func(clu *corev1.Cluster) (bool, error)

type backupCondition func(backup *corev1.Backup) (bool, error)

// WaitForClusterCondition waits a cluster to be matched to the given condition.
func WaitForClusterCondition(c *kc.Client, clusterName, conditionDesc string, timeout time.Duration, condition clusterCondition, outLog bool) error {
	framework.Logf("Waiting up to %v for cluster %q to be %q", timeout, clusterName, conditionDesc)
	var (
		lastClusterError error
		lastCluster      *corev1.Cluster
		start            = time.Now()
	)
	err := wait.PollImmediate(framework.Poll, timeout, func() (bool, error) {
		clu, err := c.DescribeCluster(context.TODO(), clusterName)
		lastClusterError = err
		if err != nil || len(clu.Items) == 0 {
			return framework.HandleWaitingAPIError(err, true, "getting cluster %s", clusterName)
		}
		lastCluster = clu.Items[0].DeepCopy()
		framework.Logf("Cluster %q: Phase=%q, Elapsed: %v",
			clusterName, lastCluster.Status.Phase, time.Since(start))
		if outLog {
			q := query.New()
			q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, clusterName)
			opList, err := c.ListOperation(context.TODO(), kc.Queries(*q))
			if err != nil {
				framework.Logf("List Cluster Operation Failed: %v", err)
			}
			for _, op := range opList.Items {
				if op.Labels[common.LabelOperationAction] == "CreateCluster" {
					err := c.PrintLogs(context.TODO(), op)
					if err != nil {
						framework.Logf("Print Step Log Failed: %v", err)
					}
				}
			}
		}

		if done, err := condition(lastCluster); done {
			if err == nil {
				framework.Logf("Cluster %q satisfied condition %q", clusterName, conditionDesc)
			}
			return true, err
		} else if err != nil {
			framework.Logf("Error evaluating cluster condition %s: %v", conditionDesc, err)
		}
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && lastCluster != nil {
		return framework.TimeoutError(fmt.Sprintf("timed out while waiting for cluster %s to be %s", clusterName, conditionDesc),
			lastCluster,
		)
	}
	if lastClusterError != nil {
		// If the last API call was an error.
		err = lastClusterError
	}
	return framework.MaybeTimeoutError(err, "waiting for cluster %s to be %s", clusterName, conditionDesc)
}

func WaitForBackupCondition(c *kc.Client, clusterName, backupName, conditionDesc string, timeout time.Duration, condition backupCondition) error {
	framework.Logf("Waiting up to %v for backup %q to be %q", timeout, backupName, conditionDesc)
	bp := &corev1.Backup{}
	start := time.Now()
	err := wait.PollImmediate(framework.Poll, timeout, func() (bool, error) {
		backups, apiErr := c.ListBackupsWithCluster(context.TODO(), clusterName)
		if apiErr != nil || len(backups.Items) == 0 {
			return framework.HandleWaitingAPIError(apiErr, true, "getting backup %s", backupName)
		}
		bp = backups.Items[0].DeepCopy()
		framework.Logf("Backup %q: Phase=%q, Elapsed: %v", backupName, bp.Status.ClusterBackupStatus, time.Since(start))
		if done, conErr := condition(bp); done {
			if conErr == nil {
				framework.Logf("Backup %q satisfied condition %q", backupName, conditionDesc)
			}
			return true, conErr
		} else if conErr != nil {
			framework.Logf("Error evaluating backup condition %s: %v", conditionDesc, conErr)
		}
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && bp != nil {
		return framework.TimeoutError(fmt.Sprintf("timed out while waiting for backup %s to be %s", backupName, conditionDesc), bp)
	}
	return framework.MaybeTimeoutError(err, "waiting for backup %s to be %s", backupName, conditionDesc)
}

func WaitForClusterRunning(c *kc.Client, clusterName string, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, fmt.Sprintf("cluster %s running", clusterName), timeout, func(clu *corev1.Cluster) (bool, error) {
		return clu.Status.Phase == corev1.ClusterRunning, nil
	}, true)
}

func WaitForClusterHealthy(c *kc.Client, clusterName string, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, fmt.Sprintf("cluster %s healthy", clusterName), timeout, func(clu *corev1.Cluster) (bool, error) {
		for _, item := range clu.Status.ComponentConditions {
			if item.Name == "kubernetes" {
				return item.Status == corev1.ComponentHealthy, nil
			}
		}
		return false, nil
	}, true)
}

// WaitForClusterNotFound returns an error if it takes too long for the pod to fully terminate.
// Unlike `waitForPodTerminatedInNamespace`, the pod's Phase and Reason are ignored. If the pod Get
// api returns IsNotFound then the wait stops and nil is returned. If the Get api returns an error other
// than "not found" then that error is returned and the wait stops.
func WaitForClusterNotFound(c *kc.Client, clusterName string, timeout time.Duration) error {
	var lastCluster *corev1.Cluster
	err := wait.PollImmediate(framework.Poll, timeout, func() (done bool, err error) {
		clu, err := c.DescribeCluster(context.TODO(), clusterName)
		if apierror.IsNotFound(err) {
			// done
			return true, nil
		}
		if err != nil {
			return framework.HandleWaitingAPIError(err, true, "getting cluster %s", clusterName)
		}
		if len(clu.Items) == 0 {
			framework.Logf("unexpected problem, cluster not be nil at this time")
			return false, nil
		}
		lastCluster = clu.Items[0].DeepCopy()
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && lastCluster != nil {
		return framework.TimeoutError(fmt.Sprintf("timeout while waiting for cluster %s to be Not Found", clusterName), lastCluster)
	}
	return framework.MaybeTimeoutError(err, "waiting for cluster %s not found", clusterName)
}

func WaitForComponentNotFound(c *kc.Client, clusterName string, timeout time.Duration) error {
	var lastCluster *corev1.Cluster
	err := wait.PollImmediate(framework.Poll, timeout, func() (done bool, err error) {
		clu, err := c.DescribeCluster(context.TODO(), clusterName)
		if err != nil {
			return framework.HandleWaitingAPIError(err, true, "getting cluster %s", clusterName)
		}
		if len(clu.Items) == 0 {
			framework.Logf("unexpected problem, cluster not be nil at this time")
			return false, nil
		}
		lastCluster = clu.Items[0].DeepCopy()
		if len(lastCluster.Addons) == 0 {
			return true, nil
		}
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && lastCluster != nil {
		return framework.TimeoutError(fmt.Sprintf("timeout while waiting for cluster %s component to be Not Found", clusterName), lastCluster)
	}
	return framework.MaybeTimeoutError(err, "waiting for cluster %s uninstall component not found", clusterName)
}

func WaitForBackupAvailable(c *kc.Client, clusterName, backupName string, timeout time.Duration) error {
	return WaitForBackupCondition(c, clusterName, backupName, fmt.Sprintf("backup %s available", backupName), timeout, func(backup *corev1.Backup) (bool, error) {
		if backup.Status.ClusterBackupStatus == corev1.ClusterBackupAvailable {
			return true, nil
		} else if backup.Status.ClusterBackupStatus == corev1.ClusterBackupError {
			return false, fmt.Errorf("backup %s create failed", backup.Name)
		} else {
			return false, nil
		}
	})
}

func WaitForBackupNotFound(c *kc.Client, clusterName, backupName string, timeout time.Duration) error {
	bp := &corev1.Backup{}
	err := wait.PollImmediate(framework.Poll, timeout, func() (done bool, err error) {
		backups, waitErr := c.ListBackupsWithCluster(context.TODO(), clusterName)
		if waitErr != nil {
			return framework.HandleWaitingAPIError(waitErr, true, "getting backup %s", backupName)
		}
		if len(backups.Items) == 0 {
			return true, nil
		}
		bp = backups.Items[0].DeepCopy()
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && bp != nil {
		return framework.TimeoutError(fmt.Sprintf("timeout while waiting for backup %s to be Not Found", backupName), bp)
	}
	return framework.MaybeTimeoutError(err, "waiting for backup %s not found", backupName)
}

func WaitForRecovery(c *kc.Client, clusterName string, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, "recovery successful", timeout, func(clu *corev1.Cluster) (bool, error) {
		if clu.Status.Phase == corev1.ClusterRunning {
			return true, nil
		} else if clu.Status.Phase == corev1.ClusterRestoreFailed {
			return false, fmt.Errorf("recovery cluster %s failed", clusterName)
		}
		return false, nil
	}, true)
}

func WaitForUpgrade(c *kc.Client, clusterName string, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, "upgrade successful", timeout, func(clu *corev1.Cluster) (bool, error) {
		if clu.Status.Phase == corev1.ClusterRunning {
			return true, nil
		} else if clu.Status.Phase == corev1.ClusterUpdateFailed {
			return false, fmt.Errorf("upgrade cluster %s failed", clusterName)
		}
		return false, nil
	}, true)
}

func WaitForCertInit(c *kc.Client, clusterName string, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, fmt.Sprintf("cluster %s cret init", clusterName), timeout, func(clu *corev1.Cluster) (bool, error) {
		for _, item := range clu.Status.Certifications {
			if item.Name == "ca" {
				return true, nil
			}
		}
		return false, nil
	}, true)
}

func WaitForCertUpdated(c *kc.Client, clusterName string, latestTime metav1.Time, timeout time.Duration) error {
	return WaitForClusterCondition(c, clusterName, fmt.Sprintf("cluster %s cert updated", clusterName), timeout, func(clu *corev1.Cluster) (bool, error) {
		for _, item := range clu.Status.Certifications {
			// update operation will just update non-ca cert,so we need check non-ca certification's expiration time
			if !strings.Contains(item.Name, "ca") && item.ExpirationTime.After(latestTime.Time) {
				framework.Logf("updated cert expiration time: %v", item.ExpirationTime.Format(time.RFC3339))
				return true, nil
			}
		}
		return false, nil
	}, true)
}

func WaitForCriRegistry(c *kc.Client, clusterName string, timeout time.Duration, registries []string) error {
	find := func(registries []corev1.RegistrySpec, target string) bool {
		for _, spec := range registries {
			if spec.Host == target {
				return true
			}
		}
		return false
	}

	return WaitForClusterCondition(c, clusterName, "cri registry successful", timeout, func(clu *corev1.Cluster) (bool, error) {
		if len(registries) == 0 {
			return false, nil
		}
		for _, reg := range registries {
			if !find(clu.Status.Registries, reg) {
				return false, fmt.Errorf("cri failed to add %s registry", reg)
			}
		}
		return true, nil
	}, true)
}

func WaitForJoinNode(c *kc.Client, nodeIP string, timeout time.Duration) error {
	node := &corev1.Node{}
	query := kc.Queries{
		FieldSelector: fmt.Sprintf("status.ipv4DefaultIP=%s", nodeIP),
	}
	err := wait.PollImmediate(framework.Poll, timeout, func() (done bool, err error) {
		nodes, waitErr := c.ListNodes(context.TODO(), query)
		if waitErr != nil {
			return framework.HandleWaitingAPIError(waitErr, true, "getting node %s", nodeIP)
		}
		if len(nodes.Items) == 0 {
			return true, nil
		}
		node = nodes.Items[0].DeepCopy()
		return false, nil
	})
	if err == nil {
		return nil
	}
	if framework.IsTimeout(err) && node != nil {
		return framework.TimeoutError(fmt.Sprintf("timeout while waiting for node %s to be join", nodeIP), node)
	}
	return framework.MaybeTimeoutError(err, "waiting for node %s join", nodeIP)
}
