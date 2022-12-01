package iam

import (
	"context"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("[Serial] users", func() {
	f := framework.NewDefaultFramework("acl")

	name := "e2e-user"

	f.AddAfterEach("after the spec is executed, the user is deleted", func(f *framework.Framework, failed bool) {
		ginkgo.By("after the spec is executed, the user is deleted")
		_ = f.KcClient().DeleteUser(context.TODO(), name)
	})

	ginkgo.It("[Fast] create user", func() {
		ctx := context.TODO()

		_, err := beforeCreateUser(ctx, f.KcClient(), name)
		framework.ExpectNoError(err)

		ginkgo.By("check the user has been created")
		_, err = f.KcClient().DescribeUser(ctx, name)
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Fast] update user", func() {
		ctx := context.TODO()

		users, err := beforeCreateUser(ctx, f.KcClient(), name)
		framework.ExpectNoError(err)

		user := users.Items[0]
		user.Spec.DisplayName = "e2e-test-display"
		user.Spec.Email = "e2e-test@kubeclipper.io"
		ginkgo.By("update user information")
		newUsers, err := f.KcClient().UpdateUser(ctx, &user)
		framework.ExpectNoError(err)

		newUser := newUsers.Items[0]

		framework.ExpectEqual("e2e-test-display", newUser.Spec.DisplayName, "The user display name is different from the expected value")
		framework.ExpectEqual("e2e-test@kubeclipper.io", newUser.Spec.Email, "The user email is different from the expected value")
	})

	ginkgo.It("[Fast] update user password", func() {
		ctx := context.TODO()

		users, err := beforeCreateUser(ctx, f.KcClient(), name)
		framework.ExpectNoError(err)
		user := users.Items[0]

		ginkgo.By("login with the old password")
		_, err = f.KcClient().Login(ctx, kc.LoginRequest{
			Username: user.Name,
			Password: "password",
		})
		framework.ExpectNoError(err)

		ginkgo.By("update user password")
		err = f.KcClient().UpdateUserPassword(ctx, user.Name, "password")
		framework.ExpectNoError(err)

		ginkgo.By("login with the new password")
		_, err = f.KcClient().Login(ctx, kc.LoginRequest{
			Username: user.Name,
			Password: "password",
		})
		framework.ExpectNoError(err)
	})

	ginkgo.It("[Fast] delete user", func() {
		ctx := context.TODO()

		users, err := beforeCreateUser(ctx, f.KcClient(), name)
		framework.ExpectNoError(err)
		user := users.Items[0]

		ginkgo.By("delete user")
		err = f.KcClient().DeleteUser(context.TODO(), user.Name)
		framework.ExpectNoError(err)

		_, err = f.KcClient().DescribeUser(ctx, user.Name)
		if err != nil && !errors.IsNotFound(err) {
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("[Slow] clear user login records", func() {
		// TODO: Turn this spec on when the trigger scenario is determined
		/*
			ctx := context.TODO()
			maxRecords := 100
			simulateTimes := maxRecords + 5

			users, err := beforeCreateUser(ctx, f.KcClient(), name)
			framework.ExpectNoError(err)
			user := users.Items[0]

			ginkgo.By(fmt.Sprintf("simulated login %d times", simulateTimes))
			// By default, only the latest 100 login logs are saved for each user.
			// Simulate 105 logins to see if they are as expected.
			for i := 0; i < simulateTimes; i++ {
				// sleep 1 second, avoid traffic limiting
				time.Sleep(1 * time.Second)
				_, err = f.KcClient().Login(ctx, kc.LoginRequest{
					Username: user.Name,
					Password: "password",
				})
				framework.ExpectNoError(err, fmt.Sprintf("An error occurred during the simulated login: times-%d", i+1))
			}

			// TODO: Triggers controller worker scheduling to clear login logs

			// Wait for the iam controller to process the login logs
			time.Sleep(20 * time.Second)

			queries := kc.Queries{
				Limit:         1,
				LabelSelector: fmt.Sprintf("%s=%s", common.LabelUserReference, user.Name),
			}
			resp, err := f.KcClient().ListLoginRecords(ctx, user.Name, queries)
			framework.ExpectNoError(err)
			framework.ExpectEqual(resp.TotalCount, maxRecords, fmt.Sprintf("The expected number of login logs is %d, but the actual number is %d", maxRecords, resp.TotalCount))
		*/
	})
})

func beforeCreateUser(ctx context.Context, f *kc.Client, name string) (*kc.UsersList, error) {
	ginkgo.By("create user")
	return f.CreateUser(ctx, initUser(name))
}

func initUser(name string) *v1.User {
	return &v1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindUser,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				common.RoleAnnotation: "platform-view",
			},
		},
		Spec: v1.UserSpec{
			DisplayName:       "e2e-test",
			EncryptedPassword: "password",
		},
	}
}
