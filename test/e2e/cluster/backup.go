package cluster

import (
	"context"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var (
	backupPointName = "e2e-backup-point"
	backupName      = "e2e-backup"
)

var _ = SIGDescribe("[Slow] [Serial] Backup", func() {
	f := framework.NewDefaultFramework("aio")

	clu := &corev1.Cluster{}
	bp := &corev1.Backup{}
	bpPoint := &corev1.BackupPoint{}

	f.AddAfterEach("cleanup aio", func(f *framework.Framework, failed bool) {
		ginkgo.By(" delete backup first ")
		err := f.Client.DeleteBackup(context.TODO(), clu.Name, bp.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for backup to be deleted")
		err = cluster.WaitForBackupNotFound(f.Client, clu.Name, bp.Name, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By("delete aio cluster")
		err = f.Client.DeleteCluster(context.TODO(), clu.Name)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)

		ginkgo.By(" delete backup-point ")
		err = f.Client.DeleteBackupPoint(context.TODO(), bpPoint.Name)
		framework.ExpectNoError(err)
	})

	ginkgo.BeforeEach(func() {
		ginkgo.By(" create backup point ")
		backupPoints, err := f.Client.CreateBackupPoint(context.TODO(), initBackUpPoint())
		framework.ExpectNoError(err)
		bpPoint = backupPoints.Items[0].DeepCopy()

		clus, err := createClusterBeforeEach(f, initClusterWithBackupPoint)
		framework.ExpectNoError(err)
		clu = clus.Items[0].DeepCopy()
	})

	ginkgo.It("create backup and ensure backup is available", func() {
		ginkgo.By(" get cluster node for backup ")
		nodes, err := f.Client.DescribeNode(context.TODO(), clu.Masters[0].ID)
		framework.ExpectNoError(err)

		ginkgo.By(" create backup ")
		backups, err := f.Client.CreateBackup(context.TODO(), clu.Name, initBackup(&nodes.Items[0]))
		framework.ExpectNoError(err)
		bp = backups.Items[0].DeepCopy()

		ginkgo.By(" check if the backup was available ")
		err = cluster.WaitForBackupAvailable(f.Client, clu.Name, backupName, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})
})

func initBackup(nodes *corev1.Node) *corev1.Backup {
	return &corev1.Backup{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: backupName,
		},
		Status: corev1.BackupStatus{},
		ClusterNodes: map[string]string{
			nodes.ProxyIpv4CIDR: nodes.Status.NodeInfo.Hostname,
		},
		PreferredNode:   "",
		BackupPointName: backupPointName,
	}
}

func initBackUpPoint() *corev1.BackupPoint {
	return &corev1.BackupPoint{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: backupPointName,
		},
		StorageType: "FS",
		FsConfig: &corev1.FsConfig{
			BackupRootDir: "/tmp",
			Description:   "",
		},
		S3Config: nil,
	}
}

func initClusterWithBackupPoint(clusterName, nodeID string) *corev1.Cluster {
	return &corev1.Cluster{
		Provider: corev1.ProviderSpec{
			Name: corev1.ClusterKubeadm,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Annotations: map[string]string{
				common.AnnotationOffline: "true",
			},
			Labels: map[string]string{
				common.LabelBackupPoint: backupPointName,
			},
		},
		Masters: corev1.WorkerNodeList{
			{
				ID: nodeID,
			},
		},
		Workers:           nil,
		KubernetesVersion: "v1.23.6",
		CertSANs:          nil,
		LocalRegistry:     "",
		ContainerRuntime: corev1.ContainerRuntime{
			Type:    corev1.CRIContainerd,
			Version: "1.6.4",
		},
		Networking: corev1.Networking{
			IPFamily:      corev1.IPFamilyIPv4,
			Services:      corev1.NetworkRanges{CIDRBlocks: []string{"10.96.0.0/16"}},
			Pods:          corev1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/24"}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},
		KubeProxy: corev1.KubeProxy{},
		Etcd:      corev1.Etcd{},
		CNI: corev1.CNI{
			LocalRegistry: "",
			Type:          "calico",
			Version:       "v3.21.2",
			Calico: &corev1.Calico{
				IPv4AutoDetection: "first-found",
				IPv6AutoDetection: "first-found",
				Mode:              "Overlay-Vxlan-All",
				IPManger:          true,
				MTU:               1440,
			},
		},
	}
}
