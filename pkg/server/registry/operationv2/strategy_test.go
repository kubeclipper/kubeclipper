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
	"context"
	"testing"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestOperationStrategyPreservesStatus(t *testing.T) {
	strategy := operationStrategy{}
	oldOperation := &operations.Operation{Status: operations.OperationStatus{Phase: operations.OperationPending}}

	updated := oldOperation.DeepCopy()
	updated.Status.Phase = operations.OperationSucceeded
	strategy.PrepareForUpdate(context.Background(), updated, oldOperation)
	if updated.Status.Phase != operations.OperationSucceeded {
		t.Fatalf("status phase = %q, want %q", updated.Status.Phase, operations.OperationSucceeded)
	}
}
