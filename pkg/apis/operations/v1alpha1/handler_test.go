/*
 * Copyright 2026 KubeClipper Authors.
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

package v1alpha1

import (
	"net/url"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestTaskLogEndpointUsesReportedAgentPort(t *testing.T) {
	node := &corev1.Node{Status: corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.10", AgentLogPort: 18080}}
	task := &operations.OperationTask{ObjectMeta: metav1.ObjectMeta{UID: "task-uid"}}
	query := url.Values{"offset": {"64"}, "limit": {"128"}}

	endpoint, err := taskLogEndpoint(node, task, query)
	if err != nil {
		t.Fatalf("taskLogEndpoint() error = %v", err)
	}
	if want := "https://192.0.2.10:18080/v1/tasks/task-uid/logs?limit=128&offset=64"; endpoint != want {
		t.Fatalf("taskLogEndpoint() = %q, want %q", endpoint, want)
	}
}

func TestTaskLogEndpointDefaultsAndRejectsInvalidPort(t *testing.T) {
	task := &operations.OperationTask{ObjectMeta: metav1.ObjectMeta{UID: "task-uid"}}
	query := url.Values{}

	endpoint, err := taskLogEndpoint(&corev1.Node{Status: corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.10"}}, task, query)
	if err != nil {
		t.Fatalf("taskLogEndpoint() error = %v", err)
	}
	if want := "https://192.0.2.10:10260/v1/tasks/task-uid/logs?"; endpoint != want {
		t.Fatalf("taskLogEndpoint() = %q, want %q", endpoint, want)
	}
	if _, err := taskLogEndpoint(&corev1.Node{Status: corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.10", AgentLogPort: 65536}}, task, query); err == nil {
		t.Fatal("taskLogEndpoint() succeeded with invalid port")
	}
}
