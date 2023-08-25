package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/clusteroperation"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/kubeclipper/kubeclipper/test/framework/cluster"
)

var _ = SIGDescribe("[Serial]", func() {
	f := framework.NewDefaultFramework("cluster")
	baseCluster := initCluster()
	clusterName := "e2e-cluster"

	ginkgo.AfterEach(afterEachDeleteCluster(f, &clusterName))

	ginkgo.It("[Slow] [AIO] should create a aio minimal kubernetes cluster", func() {
		clusterName = "e2e-aio"
		nodes := beforeEachCheckNodeEnough(f, 1)
		InitClusterWithSetter(baseCluster, []Setter{SetClusterName(clusterName), SetClusterNodes([]string{nodes[0]}, nil)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, baseCluster)()
	})

	ginkgo.It("[Slow] [HA] should create 3 master kubernetes cluster", func() {
		clusterName = "e2e-ha-3m"
		nodes := beforeEachCheckNodeEnough(f, 3)
		InitClusterWithSetter(baseCluster, []Setter{SetClusterName(clusterName), SetClusterNodes(nodes, nil)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, baseCluster)()
	})
	// other test case for 1 master 1 worker or 1 master 2 worker

	// TODO: spoilt, need fix
	ginkgo.It("[Slow] [AIO] [Backup] [Recovery] [Spoilt] should create backup and recovery from it", func() {
		bp := "e2e-bp"
		backup := "e2e-backup"
		clusterName = "e2e-aio-bk"

		nodes := beforeEachCheckNodeEnough(f, 1)

		ginkgo.By(" create backup point ")
		_, err := f.Client.CreateBackupPoint(context.TODO(), initBackUpPoint(bp))
		framework.ExpectNoError(err)

		f.AddAfterEach("delete backup point", func(f *framework.Framework, failed bool) {
			ginkgo.By(" delete backup-point ")
			_ = f.Client.DeleteBackupPoint(context.TODO(), bp)
		})

		InitClusterWithSetter(baseCluster, []Setter{SetClusterName(clusterName),
			SetClusterNodes([]string{nodes[0]}, nil),
			SetClusterBackupPoint(bp)})

		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, baseCluster)()

		ginkgo.By(" get cluster node for backup ")
		nodeLists, err := f.Client.DescribeNode(context.TODO(), nodes[0])
		framework.ExpectNoError(err)

		ginkgo.By(" create backup ")
		_, err = f.Client.CreateBackup(context.TODO(), clusterName, initBackup(&nodeLists.Items[0], backup, bp))
		framework.ExpectNoError(err)

		ginkgo.By(" check if the backup was available ")
		err = cluster.WaitForBackupAvailable(f.Client, clusterName, backup, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By(" create recovery ")
		_, err = f.Client.CreateRecovery(context.TODO(), clusterName, initRecovery(backup))
		framework.ExpectError(err)
		ginkgo.By(" check recovery successful")
		err = cluster.WaitForRecovery(f.Client, clusterName, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By(" delete backup first ")
		err = f.Client.DeleteBackup(context.TODO(), clusterName, backup)
		framework.ExpectNoError(err)
		ginkgo.By("waiting for backup to be deleted")
		err = cluster.WaitForBackupNotFound(f.Client, clusterName, backup, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Fast] [AIO] [Template] should create addon template", func() {
		addonList, err := initAddonList()
		framework.ExpectNoError(err)
		for _, addon := range addonList {
			temp := initAddonTemplate(addon.name, addon.labels, addon.data)
			ginkgo.By(fmt.Sprintf("Create %s addon-template", addon.name))
			list, err := f.Client.CreateTemplate(context.TODO(), temp)
			framework.ExpectNoError(err)
			actName := list.Items[0].Name

			ginkgo.By(fmt.Sprintf("Describe %s addon-template", addon.name))
			_, err = f.Client.DescribeTemplate(context.TODO(), actName)
			framework.ExpectNoError(err)

			ginkgo.By(fmt.Sprintf("Update %s addon-template", addon.name))
			t := &list.Items[0]
			err = editReplace(t)
			framework.ExpectNoError(err)
			_, err = f.Client.UpdateTemplate(context.TODO(), t)
			framework.ExpectNoError(err)

			ginkgo.By(fmt.Sprintf("List addon-template %s", addon.name))
			q := query.New()
			q.Limit = -1
			q.LabelSelector = fmt.Sprintf("%s=%s,%s=%s,%s=v1", common.LabelCategory, addon.category, common.LabelComponentName, addon.component, common.LabelComponentVersion)
			list, err = f.Client.ListTemplate(context.TODO(), kc.Queries(*q))
			framework.ExpectNoError(err)

			framework.ExpectNotEqual(list.TotalCount, 0, "template query result is empty")

			ginkgo.By(fmt.Sprintf("Delete addon-template %s", addon.name))
			err = f.Client.DeleteTemplate(context.TODO(), actName)
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("[Fast] [AIO] [Template] should create template", func() {
		templateName := "e2e-cluster-template"
		displayName := "e2e-cluster-template"
		cluster := baseCluster.DeepCopy()
		ctx := context.TODO()

		temp, err := initClusterTemplate(cluster, templateName, templateName)
		framework.ExpectNoError(err)

		ginkgo.By("Create cluster-template from cluster")
		_, err = f.Client.CreateTemplate(ctx, temp)
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("Describe cluster-template %s", displayName))
		list, err := f.Client.DescribeTemplate(ctx, displayName)
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("Update cluster-template %s", displayName))
		t := &list.Items[0]
		cluster.Description = "update cluster template"
		t.Config.Object = cluster
		_, err = f.Client.UpdateTemplate(ctx, t)
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("List cluster-template %s", displayName))
		q := query.New()
		q.LabelSelector = fmt.Sprintf("%s=kubernetes,%s=kubernetes,%s=v1", common.LabelCategory, common.LabelComponentName, common.LabelComponentVersion)
		list, err = f.Client.ListTemplate(ctx, kc.Queries(*q))
		framework.ExpectNoError(err)

		framework.ExpectNotEqual(list.TotalCount, 0, "template query result should not be empty")

		ginkgo.By(fmt.Sprintf("Delete cluster-template %s", displayName))
		err = f.Client.DeleteTemplate(ctx, templateName)
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Fast] [AIO] [Registry] should create registry", func() {
		ctx := context.TODO()

		afterEachDeleteRegistries(ctx, f)

		registries := initRegistries()

		for _, reg := range registries {
			ginkgo.By(fmt.Sprintf("Create %s registry", reg.Name))

			_, err := f.Client.CreateRegistry(ctx, reg)
			framework.ExpectNoError(err)
		}

		ginkgo.By("Check whether registries is created successfully")
		created, err := f.Client.ListRegistries(ctx, kc.Queries{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(created.Items), len(registries), fmt.Sprintf("%d registries have not been created", len(registries)))

		for _, reg := range registries {
			ginkgo.By(fmt.Sprintf("Delete %s registry", reg.Name))

			err := f.Client.DeleteRegistry(ctx, reg.Name)
			framework.ExpectNoError(err)
		}

		ginkgo.By("Check whether registries is deleted successfully")
		deleted, err := f.Client.ListRegistries(ctx, kc.Queries{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(deleted.Items), 0, fmt.Sprintf("%d registries have not been deleted", len(registries)))
	})

	ginkgo.It("[Slow] [AIO] [Registry] [Docker] should add registry after cluster running", func() {
		ctx := context.TODO()
		clusterName = "e2e-aio-docker"
		clu := baseCluster.DeepCopy()
		nodes := beforeEachCheckNodeEnough(f, 1)

		afterEachDeleteRegistries(ctx, f)

		registries := initRegistries()

		var registryHosts []string
		var criRegistries []corev1.CRIRegistry
		for _, reg := range registries {
			ginkgo.By(fmt.Sprintf("Create %s registry", reg.Name))

			_, err := f.Client.CreateRegistry(ctx, reg)
			framework.ExpectNoError(err)

			registryHosts = append(registryHosts, reg.Host)
			criRegistries = append(criRegistries, corev1.CRIRegistry{
				RegistryRef: &reg.Name,
			})
		}

		InitClusterWithSetter(clu, []Setter{SetClusterName(clusterName),
			SetClusterNodes([]string{nodes[0]}, nil),
			SetDockerRuntime()})

		ginkgo.By("create aio cluster with docker")
		beforeEachCreateCluster(f, clu)()

		clu.ContainerRuntime.Registries = criRegistries

		ginkgo.By("update registries to the cluster")
		err := f.Client.UpdateCluster(context.TODO(), clu)
		framework.ExpectNoError(err)

		ginkgo.By("check add registries successful")
		err = cluster.WaitForCriRegistry(f.Client, clu.Name, f.Timeouts.CommonTimeout, registryHosts)
		framework.ExpectNoError(err)

		ginkgo.By("delete registries")
		for _, reg := range registries {
			_ = f.Client.DeleteRegistry(ctx, reg.Name)
		}

		ginkgo.By("check whether the cluster registries is automatically removed after registries are deleted")
		err = cluster.WaitForClusterCondition(f.Client, clu.Name, "delete cluster registries successful", f.Timeouts.CommonTimeout, func(clu *corev1.Cluster) (bool, error) {
			return clu.Status.Registries == nil, nil
		}, true)
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Slow] [AIO] [Registry] [Containerd] should add registry after cluster running", func() {
		ctx := context.TODO()
		clusterName = "e2e-aio-containerd"
		clu := baseCluster.DeepCopy()
		nodes := beforeEachCheckNodeEnough(f, 1)

		afterEachDeleteRegistries(ctx, f)

		registries := initRegistries()

		var registryHosts []string
		var criRegistries []corev1.CRIRegistry
		for _, reg := range registries {
			ginkgo.By(fmt.Sprintf("Create %s registry", reg.Name))

			_, err := f.Client.CreateRegistry(ctx, reg)
			framework.ExpectNoError(err)

			registryHosts = append(registryHosts, reg.Host)
			criRegistries = append(criRegistries, corev1.CRIRegistry{
				RegistryRef: &reg.Name,
			})
		}

		InitClusterWithSetter(clu, []Setter{SetClusterName(clusterName),
			SetClusterNodes([]string{nodes[0]}, nil),
			SetContainerdRuntime()})
		ginkgo.By("create aio cluster with containerd")
		beforeEachCreateCluster(f, clu)()

		clus, err := f.Client.DescribeCluster(ctx, clusterName)
		framework.ExpectNoError(err)

		clu = &clus.Items[0]

		clu.ContainerRuntime.Registries = criRegistries

		ginkgo.By("update registries to the cluster")
		err = f.Client.UpdateCluster(context.TODO(), clu)
		framework.ExpectNoError(err)

		ginkgo.By("check add registries successful")
		err = cluster.WaitForCriRegistry(f.Client, clu.Name, f.Timeouts.CommonTimeout, registryHosts)
		framework.ExpectNoError(err)

		ginkgo.By("delete registries")
		for _, reg := range registries {
			_ = f.Client.DeleteRegistry(ctx, reg.Name)
		}

		ginkgo.By("check whether the cluster registries is automatically removed after registries are deleted")
		err = cluster.WaitForClusterCondition(f.Client, clu.Name, "delete cluster registries successful", f.Timeouts.CommonTimeout, func(clu *corev1.Cluster) (bool, error) {
			return clu.Status.Registries == nil, nil
		}, true)
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Slow] [AIO] [Certs] should renew cluster certs", func() {
		clusterName = "e2e-aio-certs"
		nodes := beforeEachCheckNodeEnough(f, 1)
		InitClusterWithSetter(baseCluster, []Setter{SetClusterName(clusterName), SetClusterNodes([]string{nodes[0]}, nil)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, baseCluster)()

		ginkgo.By("wait for cert init")
		err := cluster.WaitForCertInit(f.Client, clusterName, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)

		ginkgo.By("get cert expiration time")
		t, err := GetCertExpirationTime(f, clusterName)
		framework.ExpectNoError(err)

		framework.Logf("current cert expiration time: %v", t.Format(time.RFC3339))
		ginkgo.By("update cert")
		err = UpdateCert(f, clusterName)
		framework.ExpectNoError(err)

		ginkgo.By("wait for cert update")
		err = cluster.WaitForCertUpdated(f.Client, clusterName, *t, f.Timeouts.CommonTimeout)
		framework.ExpectNoError(err)
	})

	// TODO: spoilt, need fix
	ginkgo.It("[Slow] [AIO] [Storage] [Spoilt] should Install/Uninstall nfs addon", func() {
	})

	ginkgo.It("[Slow] [HA] [Node] should Add/Remove worker node in aio cluster", func() {
		clusterName = "e2e-aio-node"
		nodes := beforeEachCheckNodeEnough(f, 2)
		InitClusterWithSetter(baseCluster, []Setter{SetClusterName(clusterName), SetClusterNodes([]string{nodes[0]}, nil)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, baseCluster)()

		ginkgo.By("add worker node")
		_, err := f.Client.AddOrRemoveNode(context.TODO(), initPatchNode(clusteroperation.NodesOperationAdd, nodes[1]), clusterName)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)

		ginkgo.By("remove worker node")
		_, err = f.Client.AddOrRemoveNode(context.TODO(), initPatchNode(clusteroperation.NodesOperationRemove, nodes[1]), clusterName)
		framework.ExpectNoError(err)

		ginkgo.By("check cluster status is running")
		err = cluster.WaitForClusterRunning(f.Client, clusterName, f.Timeouts.ClusterInstallShort)
		framework.ExpectNoError(err)

		ginkgo.By("check node is removed")
		clus, err := f.Client.DescribeCluster(context.TODO(), clusterName)
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(clus.Items[0].Workers), 0, "cluster must not have worker node")
	})

	ginkgo.It("[Slow] [AIO] [Upgrade] [Online] should upgrade cluster version", func() {
		clusterName = "e2e-aio-upgrade-online"
		clu := baseCluster.DeepCopy()
		version := "v1.23.6"
		newVersion := "v1.23.9"

		nodes := beforeEachCheckNodeEnough(f, 1)
		InitClusterWithSetter(clu, []Setter{SetClusterName(clusterName),
			SetClusterNodes([]string{nodes[0]}, nil),
			SetClusterVersion(version)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, clu)()

		ginkgo.By("upgrade cluster")
		err := f.Client.UpgradeCluster(context.TODO(), clusterName, initUpgradeCluster(false, newVersion))
		framework.ExpectNoError(err)

		ginkgo.By("wait cluster upgrade")
		err = cluster.WaitForUpgrade(f.Client, clusterName, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)
	})

	// TODO: make version be parameter
	ginkgo.It("[Slow] [AIO] [Upgrade] [Offline] should upgrade cluster version", func() {
		clusterName = "e2e-aio-upgrade-offline"
		clu := baseCluster.DeepCopy()
		version := "v1.23.6"
		newVersion := "v1.23.9"

		nodes := beforeEachCheckNodeEnough(f, 1)
		InitClusterWithSetter(clu, []Setter{SetClusterName(clusterName),
			SetClusterNodes([]string{nodes[0]}, nil),
			SetClusterVersion(version)})
		ginkgo.By("create aio cluster")
		beforeEachCreateCluster(f, clu)()

		ginkgo.By("upgrade cluster")
		err := f.Client.UpgradeCluster(context.TODO(), clusterName, initUpgradeCluster(false, newVersion))
		framework.ExpectNoError(err)

		ginkgo.By("wait cluster upgrade")
		err = cluster.WaitForUpgrade(f.Client, clusterName, f.Timeouts.ClusterInstall)
		framework.ExpectNoError(err)
	})

})

func UpdateCert(f *framework.Framework, clusterName string) error {
	_, err := f.Client.UpdateCert(context.TODO(), clusterName)
	return err
}

func GetCertExpirationTime(f *framework.Framework, clusterName string) (*metav1.Time, error) {
	describeCluster, err := f.Client.DescribeCluster(context.TODO(), clusterName)
	if err != nil {
		return nil, err
	}
	if len(describeCluster.Items) == 0 {
		return nil, fmt.Errorf("unexpected problem, cluster %s not found", clusterName)
	}
	for _, certification := range describeCluster.Items[0].Status.Certifications {
		if !strings.Contains(certification.Name, "ca") {
			return &certification.ExpirationTime, nil
		}
	}
	return nil, fmt.Errorf("unexpected problem, cluster %s ca expiration time not exist", clusterName)
}

func initPatchNode(operation clusteroperation.NodesPatchOperation, node string) *clusteroperation.PatchNodes {
	return &clusteroperation.PatchNodes{
		Operation: operation,
		Role:      common.NodeRoleWorker,
		Nodes: corev1.WorkerNodeList{
			{
				ID: node,
			},
		},
	}
}

func initUpgradeCluster(offLine bool, version string) *apiv1.ClusterUpgrade {
	return &apiv1.ClusterUpgrade{
		Offline: offLine,
		Version: version,
	}
}
