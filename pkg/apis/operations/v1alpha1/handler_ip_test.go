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
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

type targetNodeReader map[string]*corev1.Node

func (r targetNodeReader) GetNodeEx(_ context.Context, name, _ string) (*corev1.Node, error) {
	return r[name], nil
}

func TestValidateAndEnrichTargetIPs(t *testing.T) {
	op := &operations.Operation{
		Spec: operations.OperationSpec{Steps: []operations.OperationStep{{
			Targets: []operations.NodeReference{{Name: "node-a", UID: types.UID("node-a-uid"), IP: "192.0.2.99"}},
		}}},
	}
	reader := targetNodeReader{
		"node-a": {
			ObjectMeta: metav1.ObjectMeta{Name: "node-a", UID: types.UID("node-a-uid")},
			Status:     corev1.NodeStatus{Ipv4DefaultIP: "192.0.2.10"},
		},
	}

	if err := validateAndEnrichTargetIPs(context.Background(), op, reader); err != nil {
		t.Fatal(err)
	}
	if got := op.Spec.Steps[0].Targets[0].IP; got != "192.0.2.10" {
		t.Fatalf("target IP = %q, want authoritative node IP", got)
	}
}
