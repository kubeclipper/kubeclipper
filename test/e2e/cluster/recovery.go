package cluster

import (
	"context"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = SIGDescribe("[Slow] [Serial] Recovery", func() {
	f := framework.NewDefaultFramework("aio")

	bpPoint := &corev1.BackupPoint{}
	bp := &corev1.Backup{}
	clu := &corev1.Cluster{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By(" delete backup first ")
		err := f.Client.DeleteBackup(context.TODO(), clu.Name, bp.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for backup to be deleted")
		err = cluster.WaitForBackupNotFound(f.Client, clu.Name, bp.Name, f.Timeouts.ComponentInstall)

		ginkgo.By("delete aio cluster")
		err = f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)

		ginkgo.By(" delete backup-point ")
		err = f.Client.DeleteBackupPoint(context.TODO(), bpPoint.Name)
	})

	ginkgo.BeforeEach(func() {
		//创建一个备份点
		ginkgo.By(" create backup point ")
		backupPoints, err := f.Client.CreateBackupPoint(context.TODO(), initBackUpPoint())
		framework.ExpectNoError(err)
		if len(backupPoints.Items) == 0 {
			framework.Failf("unexpected problem, backup not be nil at this time")
		}
		bpPoint = backupPoints.Items[0].DeepCopy()

		//创建一个集群
		clus, err := createClusterBeforeEach(f, initClusterWithBackupPoint)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()

		//获取备份节点
		ginkgo.By(" get cluster node for backup ")
		nodes, err := f.Client.DescribeNode(context.TODO(), clu.Masters[0].ID)
		framework.ExpectNoError(err)
		//创建备份
		ginkgo.By(" create backup ")
		backups, err := f.Client.CreateBackup(context.TODO(), clu.Name, initBackup(&nodes.Items[0]))
		framework.ExpectNoError(err)
		bp = backups.Items[0].DeepCopy()
		// 检查是否备份成功
		ginkgo.By(" check if the backup was available ")
		err = cluster.WaitForBackupAvailable(f.Client, clu.Name, backupName, f.Timeouts.ComponentInstall)
		framework.ExpectNoError(err)
	})

	ginkgo.It(" create recovery and ensure recovery successful ", func() {
		ginkgo.By(" create recovery ")
		_, err := f.Client.CreateRecovery(context.TODO(), clu.Name, initRecovery(backupName))
		framework.ExpectError(err)
		ginkgo.By(" check recovery successful")
		err = cluster.WaitForRecovery(f.Client, clu.Name, f.Timeouts.ComponentInstall)
		framework.ExpectNoError(err)
	})
})

func initRecovery(backupName string) *corev1.Recovery {
	return &corev1.Recovery{
		TypeMeta:      metav1.TypeMeta{},
		ObjectMeta:    metav1.ObjectMeta{},
		UseBackupName: backupName,
		Description:   "",
	}
}
