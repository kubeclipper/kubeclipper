package region

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Fast] [Serial] List Region", func() {
	f := framework.NewDefaultFramework("region")
	ginkgo.It("list region and check is default region exist", func() {
		ctx := context.Background()
		ginkgo.By("list region")
		q := query.New()
		q.Limit = -1
		list, err := f.Client.ListRegion(ctx, kc.Queries(*q))
		framework.ExpectNoError(err)

		if getDefault(list.Items) == nil {
			ginkgo.Fail("default region not exist")
		}

		ginkgo.By("search default region")
		q = query.New()
		q.Limit = -1
		q.FuzzySearch = map[string]string{"name": "default"}
		list, err = f.Client.ListRegion(ctx, kc.Queries(*q))
		framework.ExpectNoError(err)
		region := getDefault(list.Items)
		if region == nil {
			ginkgo.Fail("default region not exist")
		}
		if !isValidTime(region.CreationTimestamp) {
			ginkgo.Fail(fmt.Sprintf("default region's create time(%s) is invalid", region.CreationTimestamp.Format(time.RFC3339)))
		}
	})
})

func getDefault(regions []v1.Region) *v1.Region {
	for _, region := range regions {
		if region.Name == "default" {
			return &region
		}
	}
	return nil
}

// isValidTime if 0 < time.unix < now, we think it's a valid time
func isValidTime(t metav1.Time) bool {
	now := metav1.Now()
	return t.Before(&now) && t.After(time.Unix(0, 0))
}
