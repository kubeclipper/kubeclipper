/*
 * Copyright 2021 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
)

func TestInitialNodePublishesHostnameLabel(t *testing.T) {
	cfg := config.New()
	cfg.AgentID = "agent-1"
	server := &Server{Config: cfg}
	node := server.initialNode()

	if node.Status.NodeInfo.Hostname == "" {
		t.Fatal("expected hostname in node status")
	}
	if got := node.Labels[common.LabelHostname]; got != node.Status.NodeInfo.Hostname {
		t.Fatalf("hostname label = %q, want %q", got, node.Status.NodeInfo.Hostname)
	}
}
