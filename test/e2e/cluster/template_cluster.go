package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = SIGDescribe("[Slow] [Serial] TemplateCluster", func() {
	f := framework.NewDefaultFramework("cluster-template")
	ctx := context.TODO()
	clu := initAIOCluster("e2e-cluster-aio", []string{"demo-node-id"})
	temp, err := initClusterTemplate(clu)
	framework.ExpectNoError(err)
	displayName := temp.Annotations[common.AnnotationDisplayName]
	name := temp.Name

	ginkgo.It("cluster-template CURD", func() {
		ginkgo.By("Create cluster-template from cluster")
		_, err := f.Client.CreateTemplate(context.TODO(), temp)
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("Describe cluster-template %s", displayName))
		list, err := f.Client.DescribeTemplate(ctx, name)
		framework.ExpectNoError(err)

		ginkgo.By(fmt.Sprintf("Update cluster-template %s", displayName))
		t := &list.Items[0]
		clu.Description = "update cluster template"
		t.Config.Object = clu
		_, err = f.Client.UpdateTemplate(ctx, t)
		framework.ExpectNoError(err)

		_, err = f.Client.ListClusters(context.TODO(), kc.Queries(*query.New()))
		framework.ExpectNoError(err)
		ginkgo.By(fmt.Sprintf("List cluster-template %s", displayName))
		q := query.New()
		q.LabelSelector = fmt.Sprintf("%s=kubernetes,%s=kubernetes,%s=v1", common.LabelCategory, common.LabelComponentName, common.LabelComponentVersion)
		list, err = f.Client.ListTemplate(ctx, kc.Queries(*q))
		framework.ExpectNoError(err)
		if list.TotalCount == 0 {
			framework.ExpectNoError(errors.New("template query result is empty"))
		}

		ginkgo.By(fmt.Sprintf("Delete cluster-template %s", displayName))
		err = f.Client.DeleteTemplate(ctx, name)
		framework.ExpectNoError(err)
	})
})

func initClusterTemplate(clu *corev1.Cluster) (*corev1.Template, error) {
	data, err := json.Marshal(clu)
	if err != nil {
		return nil, err
	}

	return &corev1.Template{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Template",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				common.AnnotationDisplayName: "e2e-cluster-template",
			},
			Labels: map[string]string{
				common.LabelCategory:         "kubernetes",
				common.LabelComponentName:    "kubernetes",
				common.LabelComponentVersion: "v1",
			},
			Name: "e2e-cluster-template",
		},
		Config: runtime.RawExtension{Raw: data},
	}, nil
}
