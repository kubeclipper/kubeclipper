package cluster

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Slow] [Serial] Force delete cluster", func() {
	f := framework.NewDefaultFramework("aio")

	clu := &corev1.Cluster{}

	ginkgo.BeforeEach(func() {
		ginkgo.By("create aio cluster")
		clusters, err := createClusterBeforeEach(f, "cluster-aio", initAIOCluster)
		framework.ExpectNoError(err)
		clu = clusters.Items[0].DeepCopy()
	})

	ginkgo.It("force delete aio cluster and ensure cluster is deleted", func() {
		ginkgo.By("force delete aio cluster by kcctl")
		err := deleteClusterByKcctl(clu.Name)
		framework.ExpectNoError(err)

		ginkgo.By("waiting for cluster to be deleted")
		err = cluster.WaitForClusterNotFound(f.Client, clu.Name, f.Timeouts.ClusterDelete)
		framework.ExpectNoError(err)
	})
})

func deleteClusterByKcctl(clusterName string) error {
	cmd := fmt.Sprintf("echo y | kcctl delete cluster %s -F", clusterName)
	ec := cmdutil.NewExecCmd(context.TODO(), "bash", "-c", cmd)
	return ec.Run()
}
