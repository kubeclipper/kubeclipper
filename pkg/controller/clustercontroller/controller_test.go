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

package clustercontroller

import (
	"testing"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestFindOperationCluster(t *testing.T) {
	requests := findOperationCluster(&operations.Operation{Spec: operations.OperationSpec{
		TargetRef: operations.ObjectReference{Kind: "Cluster", Name: "cluster-a"},
	}})
	if len(requests) != 1 || requests[0].Name != "cluster-a" {
		t.Fatalf("findOperationCluster() = %#v, want cluster-a", requests)
	}
	if requests := findOperationCluster(&operations.Operation{Spec: operations.OperationSpec{
		TargetRef: operations.ObjectReference{Kind: "Node", Name: "node-a"},
	}}); len(requests) != 0 {
		t.Fatalf("findOperationCluster() = %#v, want no requests", requests)
	}
}
