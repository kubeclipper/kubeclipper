package cluster

import (
	"context"
	"encoding/json"

	"github.com/onsi/ginkgo"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/component/nfscsi"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

func initNFSCSIComponents(addr, sharedPath, mirrorRegistry string) *apiv1.PatchComponents {
	nfs := nfscsi.NFS{
		ImageRepoMirror:  mirrorRegistry,
		ServerAddr:       addr,
		SharedPath:       sharedPath,
		Replicas:         1,
		StorageClassName: "nfs-sc",
		IsDefault:        false,
		ReclaimPolicy:    "Delete",
		MountOptions:     []string{},
	}
	buf, _ := json.Marshal(nfs)
	return &apiv1.PatchComponents{
		Addons: []corev1.Addon{
			{
				Name:    "nfs-csi",
				Version: "v1",
				Config: runtime.RawExtension{
					Raw: buf,
				},
			},
		},
	}
}

func nfsComponentTestBlock(f *framework.Framework, clu *corev1.Cluster, nfsAddr, nfsSharedPath, mirrorRegistry string) {
	clusterName := clu.Name
	nfsComponent := initNFSCSIComponents(nfsAddr, nfsSharedPath, mirrorRegistry)
	nodes := beforeEachCheckNodeEnough(f, 1)
	InitClusterWithSetter(clu, []Setter{SetClusterName(clusterName),
		SetClusterNodes([]string{nodes[0]}, nil)})

	ginkgo.By("create aio cluster", beforeEachCreateCluster(f, clu))

	ginkgo.By("install NFS storage component", func() {
		_, err := f.KcClient().InstallOrUninstallComponent(context.TODO(), clusterName, nfsComponent)
		framework.ExpectNoError(err)
	})

	ginkgo.By("check cluster status is Updating", func() {
		clusterItem, err := cluster.DescribeCluster(f.KcClient(), clusterName)
		framework.ExpectNoError(err)
		framework.ExpectEqual(clusterItem.Status.Phase, corev1.ClusterUpdating)
	})

	ginkgo.By("check cluster status is running", func() {
		err := cluster.WaitForClusterRunning(
			f.KcClient(), clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)
	})

	ginkgo.By("check nfs component installed", func() {
		exist, err := cluster.ExistComponent(f.KcClient(), clusterName, "nfs-csi")
		framework.ExpectNoError(err)
		framework.ExpectEqual(exist, true)
	})

	ginkgo.By("remove NFS storage component", func() {
		nfsComponent.Uninstall = true
		_, err := f.KcClient().InstallOrUninstallComponent(context.TODO(), clusterName, nfsComponent)
		framework.ExpectNoError(err)
	})

	ginkgo.By("check cluster status is Updating", func() {
		clusterItem, err := cluster.DescribeCluster(f.KcClient(), clusterName)
		framework.ExpectNoError(err)
		framework.ExpectEqual(clusterItem.Status.Phase, corev1.ClusterUpdating)
	})

	ginkgo.By("check cluster status is running", func() {
		err := cluster.WaitForClusterRunning(
			f.KcClient(), clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)
	})

	ginkgo.By("check nfs component removed", func() {
		exist, err := cluster.ExistComponent(f.KcClient(), clusterName, "nfs-csi")
		framework.ExpectNoError(err)
		framework.ExpectEqual(exist, false)
	})
}
