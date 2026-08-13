/*
 *
 *  * Copyright 2026 KubeClipper Authors.
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

package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestNodeCreateAndUpdateStatus(t *testing.T) {
	t.Parallel()

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if got, want := r.Method, http.MethodPost; got != want {
				t.Errorf("create method = %q, want %q", got, want)
			}
			if got, want := r.URL.Path, "/api/core.kubeclipper.io/v1/nodes"; got != want {
				t.Errorf("create path = %q, want %q", got, want)
			}
		case 2:
			if got, want := r.Method, http.MethodPut; got != want {
				t.Errorf("status method = %q, want %q", got, want)
			}
			if got, want := r.URL.Path, "/api/core.kubeclipper.io/v1/nodes/agent-a/status"; got != want {
				t.Errorf("status path = %q, want %q", got, want)
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}

		var node corev1.Node
		if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
			t.Errorf("decode node: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&node)
	}))
	defer server.Close()

	client, err := NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatalf("NewForConfig() error = %v", err)
	}
	node := &corev1.Node{
		TypeMeta:   metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Node"},
		ObjectMeta: metav1.ObjectMeta{Name: "agent-a"},
	}
	created, err := client.Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := client.Nodes().UpdateStatus(context.Background(), created, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("request count = %d, want 2", requests)
	}
}
