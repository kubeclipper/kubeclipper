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

package framework

import (
	"flag"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"

	"github.com/onsi/ginkgo/config"
)

const (
	defaultHost          = "http://127.0.0.1:8080"
	defaultServiceSubnet = constatns.ClusterServiceSubnet
	defaultPodSubnet     = constatns.ClusterPodSubnet
	defaultLocalRegistry = "127.0.0.1:5000"
	defaultWorkerNodeVip = "169.254.169.100"
)

type TestContextType struct {
	Host          string
	InMemoryTest  bool
	ServiceSubnet string
	PodSubnet     string
	LocalRegistry string
	WorkerNodeVip string
}

// TestContext should be used by all tests to access common context data.
var TestContext TestContextType

func RegisterCommonFlags(flags *flag.FlagSet) {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true

	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	// Randomize specs as well as suites
	config.GinkgoConfig.RandomizeAllSpecs = true

	flag.BoolVar(&TestContext.InMemoryTest, "in-memory-test", false,
		"Whether Ki-server and Ki-agent be started in memory.")
	flag.StringVar(&TestContext.Host, "server-address", defaultHost,
		"Ki Server API Server IP/DNS, default 127.0.0.1:8080")
	flag.StringVar(&TestContext.ServiceSubnet, "svc-subnet", defaultServiceSubnet,
		"cluster svc sub net, default 10.96.0.0/12")
	flag.StringVar(&TestContext.PodSubnet, "pod-subnet", defaultPodSubnet,
		"cluster pod sub net, default 172.25.0.0/16")
	flag.StringVar(&TestContext.LocalRegistry, "registry", defaultLocalRegistry,
		"cri image registry addr, default 127.0.0.1:5000")
	flag.StringVar(&TestContext.WorkerNodeVip, "vip", defaultWorkerNodeVip,
		"cluster worker node loadblance vip, default 169.254.169.100")
	flag.DurationVar(&clusterInstallShort, "cluster-install-short-timeout", clusterInstallShort,
		"cluster install short timeout interval")
}
