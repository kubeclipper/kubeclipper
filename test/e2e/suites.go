/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package e2e

import (
	"context"

	"github.com/onsi/ginkgo"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

// CleanupSuite is the boilerplate that can be used after tests on ginkgo were run, on the SynchronizedAfterSuite step.
// Similar to SynchronizedBeforeSuite, we want to run some operations only once (such as collecting cluster logs).
// Here, the order of functions is reversed; first, the function which runs everywhere,
// and then the function that only runs on the first Ginkgo node.
func CleanupSuite() {
	// Run on all Ginkgo nodes
	framework.Logf("Running AfterSuite actions on all nodes")
	framework.RunCleanupActions()
}

// AfterSuiteActions are actions that are run on ginkgo's SynchronizedAfterSuite
func AfterSuiteActions() {
	// Run only Ginkgo on node 1
	framework.Logf("Running AfterSuite actions on node 1")
}

func SetupSuite() {
	ginkgo.By("Initial kc client")
	c, err := kc.NewClientWithOpts(kc.WithHost(framework.TestContext.Host))
	framework.ExpectNoError(err)
	resp, err := c.Login(context.TODO(), kc.LoginRequest{
		Username: framework.TestContext.Username,
		Password: framework.TestContext.Password,
	})
	framework.ExpectNoError(err)

	c, err = kc.NewClientWithOpts(kc.WithHost(framework.TestContext.Host), kc.WithBearerAuth(resp.AccessToken))
	framework.ExpectNoError(err)
	framework.TestContext.Client = c
}
