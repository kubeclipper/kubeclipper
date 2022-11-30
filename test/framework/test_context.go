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

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/onsi/ginkgo/config"
)

const (
	defaultHost          = "http://127.0.0.1:8080"
	defaultLocalRegistry = "127.0.0.1:5000"
)

// TestContext should be used by all tests to access common context data.
var TestContext TestContextType

type TestContextType struct {
	Host          string
	LocalRegistry string
	Username      string
	Password      string
	Client        *kc.Client
}

func RegisterCommonFlags(flags *flag.FlagSet) {
	// Turn on verbose by default to get spec names
	config.DefaultReporterConfig.Verbose = true

	// Turn on EmitSpecProgress to get spec progress (especially on interrupt)
	config.GinkgoConfig.EmitSpecProgress = true

	// Randomize specs as well as suites
	config.GinkgoConfig.RandomizeAllSpecs = true

	flag.StringVar(&TestContext.Host, "server-address", defaultHost,
		"Kc Server API Server IP/DNS, default 127.0.0.1:8080")
	flag.StringVar(&TestContext.LocalRegistry, "registry", defaultLocalRegistry,
		"cri image registry addr, default 127.0.0.1:5000")
	flag.DurationVar(&clusterInstallShort, "cluster-install-short-timeout", clusterInstallShort,
		"cluster install short timeout interval")
	flag.StringVar(&TestContext.Username, "username", "admin",
		"kc server auth user")
	flag.StringVar(&TestContext.Password, "password", "Thinkbig1",
		"kc server auth user password")
}
