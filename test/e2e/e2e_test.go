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
	"flag"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/kubeclipper/kubeclipper/test/e2e/cluster"
	_ "github.com/kubeclipper/kubeclipper/test/e2e/iam"
	_ "github.com/kubeclipper/kubeclipper/test/e2e/node"
	_ "github.com/kubeclipper/kubeclipper/test/e2e/region"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	framework.RegisterCommonFlags(flag.CommandLine)
	flag.Parse()
}

func TestMain(m *testing.M) {
	handleFlags()
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestRunE2ETests(t *testing.T) {
	RunE2ETests(t)
}
