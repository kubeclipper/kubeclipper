/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestLeaseCreateUpdateAndGetWithNamespace(t *testing.T) {
	t.Parallel()

	expected := []struct {
		method    string
		path      string
		namespace string
	}{
		{method: http.MethodPost, path: "/api/core.kubeclipper.io/v1/leases"},
		{method: http.MethodPut, path: "/api/core.kubeclipper.io/v1/leases/agent-a"},
		{method: http.MethodGet, path: "/api/core.kubeclipper.io/v1/leases/agent-a", namespace: "node-lease"},
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > len(expected) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			return
		}
		want := expected[requests-1]
		if got := r.Method; got != want.method {
			t.Errorf("request %d method = %q, want %q", requests, got, want.method)
		}
		if got := r.URL.Path; got != want.path {
			t.Errorf("request %d path = %q, want %q", requests, got, want.path)
		}
		if got := r.URL.Query().Get("namespace"); got != want.namespace {
			t.Errorf("request %d namespace = %q, want %q", requests, got, want.namespace)
		}

		lease := coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "agent-a", Namespace: "node-lease"}}
		if r.Method != http.MethodGet {
			if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
				t.Errorf("decode lease: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(&lease); err != nil {
			t.Errorf("encode lease: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig() error = %v", err)
	}
	lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{Name: "agent-a", Namespace: "node-lease"}}
	created, err := client.Leases().Create(context.Background(), lease, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := client.Leases().Update(context.Background(), created, &metav1.UpdateOptions{}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if _, err := client.Leases().GetWithNamespace(context.Background(), "agent-a", "node-lease", metav1.GetOptions{}); err != nil {
		t.Fatalf("GetWithNamespace() error = %v", err)
	}
	if requests != 3 {
		t.Fatalf("request count = %d, want 3", requests)
	}
}
