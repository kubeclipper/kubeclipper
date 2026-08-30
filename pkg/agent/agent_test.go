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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
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
	if _, found := node.Status.Capacity[corev1.ResourceCPU]; !found {
		t.Fatal("expected CPU capacity in node status")
	}
	if _, found := node.Status.Capacity[corev1.ResourceMemory]; !found {
		t.Fatal("expected memory capacity in node status")
	}
	if got, want := node.Status.AgentLogPort, int32(10260); got != want {
		t.Fatalf("agent log port = %d, want %d", got, want)
	}
}

func TestInitialNodePublishesConfiguredAgentLogPort(t *testing.T) {
	cfg := config.New()
	cfg.AgentID = "agent-1"
	cfg.APIServer.LogAddress = ":18080"

	if got, want := (&Server{Config: cfg}).initialNode().Status.AgentLogPort, int32(18080); got != want {
		t.Fatalf("agent log port = %d, want %d", got, want)
	}
}

func TestNewNodeLease(t *testing.T) {
	server := &Server{Config: &config.Config{AgentID: "agent-1"}}
	lease := server.newNodeLease(nil)

	if got, want := lease.Name, "agent-1"; got != want {
		t.Fatalf("lease name = %q, want %q", got, want)
	}
	if got, want := lease.Namespace, nodeLeaseNamespace; got != want {
		t.Fatalf("lease namespace = %q, want %q", got, want)
	}
	if lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity != "agent-1" {
		t.Fatalf("lease holder identity = %v, want agent-1", lease.Spec.HolderIdentity)
	}
	if lease.Spec.LeaseDurationSeconds == nil || *lease.Spec.LeaseDurationSeconds != nodeLeaseDurationSeconds {
		t.Fatalf("lease duration = %v, want %d", lease.Spec.LeaseDurationSeconds, nodeLeaseDurationSeconds)
	}
	if lease.Spec.RenewTime == nil {
		t.Fatal("expected lease renew time")
	}
}

func TestRenewNodeLeaseCreatesMissingLease(t *testing.T) {
	t.Parallel()

	server, requests := newMissingLeaseServer(t)
	defer server.Close()

	client, err := clientset.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig() error = %v", err)
	}
	agent := &Server{Config: &config.Config{AgentID: "agent-1"}, client: client}
	if err := agent.renewNodeLease(); err != nil {
		t.Fatalf("renewNodeLease() error = %v", err)
	}
	if *requests != 2 {
		t.Fatalf("request count = %d, want 2", *requests)
	}
}

func newMissingLeaseServer(t *testing.T) (server *httptest.Server, requests *int) {
	t.Helper()
	requestCount := 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			if got, want := r.Method, http.MethodGet; got != want {
				t.Errorf("get method = %q, want %q", got, want)
			}
			if got, want := r.URL.Path, "/api/core.kubeclipper.io/v1/leases/agent-1"; got != want {
				t.Errorf("get path = %q, want %q", got, want)
			}
			if got, want := r.URL.Query().Get("namespace"), nodeLeaseNamespace; got != want {
				t.Errorf("get namespace = %q, want %q", got, want)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(&metav1.Status{Status: metav1.StatusFailure, Reason: metav1.StatusReasonNotFound, Code: http.StatusNotFound}); err != nil {
				t.Errorf("encode status: %v", err)
			}
		case 2:
			if got, want := r.Method, http.MethodPost; got != want {
				t.Errorf("create method = %q, want %q", got, want)
			}
			if got, want := r.URL.Path, "/api/core.kubeclipper.io/v1/leases"; got != want {
				t.Errorf("create path = %q, want %q", got, want)
			}
			var lease coordinationv1.Lease
			if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
				t.Errorf("decode lease: %v", err)
			}
			if got, want := lease.Namespace, nodeLeaseNamespace; got != want {
				t.Errorf("lease namespace = %q, want %q", got, want)
			}
			if lease.Spec.RenewTime == nil {
				t.Error("lease renew time is nil")
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(&lease); err != nil {
				t.Errorf("encode lease: %v", err)
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	return server, &requestCount
}
