package iam

import (
	"context"
	"encoding/json"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Fast] [Serial] roles", func() {
	f := framework.NewDefaultFramework("acl")

	roleName, aggRole := "test-role", "role-template-view-audit"
	newAggRole := "role-template-view-users"

	f.AddAfterEach("after the spec is executed, the role is deleted", func(f *framework.Framework, failed bool) {
		ginkgo.By("after the spec is executed, the role is deleted")
		_ = f.KcClient().DeleteRole(context.TODO(), roleName)
	})

	ginkgo.It("create role", func() {
		ctx := context.TODO()

		_, err := beforeCreateRole(ctx, f.KcClient(), roleName, aggRole)
		framework.ExpectNoError(err)

		ginkgo.By("check the role has been created")
		_, err = f.KcClient().DescribeRole(ctx, roleName)
		framework.ExpectNoError(err)
	})

	ginkgo.It("update role description", func() {
		ctx := context.TODO()

		roles, err := beforeCreateRole(ctx, f.KcClient(), roleName, aggRole)
		framework.ExpectNoError(err)

		role := roles.Items[0]
		role.Annotations[common.AnnotationDescription] = "description"
		ginkgo.By("update role description")
		newRoles, err := f.KcClient().UpdateRole(ctx, &role)
		framework.ExpectNoError(err)

		newRole := newRoles.Items[0]

		framework.ExpectEqual("description", newRole.Annotations[common.AnnotationDescription], "The role description is different from the expected value")
	})

	ginkgo.It("update role permissions", func() {
		ctx := context.TODO()

		roles, err := beforeCreateRole(ctx, f.KcClient(), roleName, aggRole)
		framework.ExpectNoError(err)
		role := roles.Items[0]

		var permissions []string
		err = json.Unmarshal([]byte(role.Annotations[common.AnnotationAggregationRoles]), &permissions)
		framework.ExpectNoError(err)

		permissions = append(permissions, newAggRole)

		newPermissions, err := json.Marshal(permissions)
		framework.ExpectNoError(err)

		role.Annotations[common.AnnotationAggregationRoles] = string(newPermissions)
		ginkgo.By("update role permission")
		newRoles, err := f.KcClient().UpdateRole(ctx, &role)
		framework.ExpectNoError(err)
		newRole := newRoles.Items[0]

		var nAgg []string
		err = json.Unmarshal([]byte(newRole.Annotations[common.AnnotationAggregationRoles]), &nAgg)
		framework.ExpectNoError(err)

		framework.ExpectEqual(len(permissions), len(nAgg), "The role permission is different from the expected value")
	})

	ginkgo.It("delete role", func() {
		ctx := context.TODO()

		_, err := beforeCreateRole(ctx, f.KcClient(), roleName, aggRole)
		framework.ExpectNoError(err)

		err = f.KcClient().DeleteRole(ctx, roleName)
		framework.ExpectNoError(err)

		_, err = f.KcClient().DescribeRole(ctx, roleName)
		framework.ExpectError(err, "The role should not be found")
	})
})

func initRole(name string, agRoles []string) *v1.GlobalRole {
	agSlice, _ := json.Marshal(agRoles)
	return &v1.GlobalRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindGlobalRole,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"kubeclipper.io/aggregation-roles": string(agSlice),
			},
		},
	}
}

func beforeCreateRole(ctx context.Context, f *kc.Client, roleName, aggRole string) (*kc.RoleList, error) {
	ginkgo.By("create role")
	return f.CreateRole(ctx, initRole(roleName, []string{aggRole}))
}
