package cluster

import (
	"time"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Slow] [Serial] HA", func() {
	f := framework.NewDefaultFramework("ha")

	f.AddAfterEach("cleanup ha", func(f *framework.Framework, failed bool) {
		ginkgo.By("delete aio cluster")
		// TODO...
	})
	ginkgo.It("should create a HA minimal kubernetes cluster and ensure cluster is running.", func() {
		time.Sleep(5 * time.Second)
		ginkgo.By("create aio cluster")
		// ...
		ginkgo.By("wait for cluster is healthy")
		// ...
		ginkgo.By("check cluster status is running")
		// ...
		ginkgo.By("ensure operation status is successful")
	})
})
