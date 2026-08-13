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

package operationv2

import (
	"bytes"
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestNodePreflight(t *testing.T) {
	task := &operations.OperationTask{Spec: operations.OperationTaskSpec{Payload: runtime.RawExtension{Raw: []byte(`{"requiredExecutables":["go"],"requiredPaths":["/"]}`)}}}
	result, err := (NodePreflight{}).Reconcile(context.Background(), task, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outputs["os"] == "" || result.Outputs["arch"] == "" || result.Outputs["hostname"] == "" {
		t.Fatalf("missing preflight outputs: %#v", result.Outputs)
	}
}

func TestNodePreflightRejectsUnknownPayload(t *testing.T) {
	task := &operations.OperationTask{Spec: operations.OperationTaskSpec{Payload: runtime.RawExtension{Raw: []byte(`{"command":"rm -rf /"}`)}}}
	if _, err := (NodePreflight{}).Reconcile(context.Background(), task, &bytes.Buffer{}); err == nil {
		t.Fatal("unknown payload field was accepted")
	}
}

func TestRegistryRejectsDuplicateExecutor(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(NodePreflightExecutorName, NodePreflight{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(NodePreflightExecutorName, NodePreflight{}); err == nil {
		t.Fatal("duplicate executor was accepted")
	}
}
