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

package operationv2

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	operationslister "github.com/kubeclipper/kubeclipper/pkg/client/lister/operations/v1alpha1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	clustermock "github.com/kubeclipper/kubeclipper/pkg/models/cluster/mock"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestBusinessReconcileIgnoresMissingTargetForTerminalOperation(t *testing.T) {
	operation := &operations.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "create-cluster"},
		Spec: operations.OperationSpec{
			Action:    corev1.OperationCreateCluster,
			TargetRef: operations.ObjectReference{Name: "deleted-cluster"},
		},
		Status: operations.OperationStatus{Phase: operations.OperationFailed},
	}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := indexer.Add(operation); err != nil {
		t.Fatal(err)
	}

	controller := gomock.NewController(t)
	clusters := clustermock.NewMockOperator(controller)
	clusters.EXPECT().GetClusterEx(gomock.Any(), "deleted-cluster", "").Return(nil,
		apierrors.NewNotFound(corev1.Resource("cluster"), "deleted-cluster"))
	reconciler := &BusinessReconciler{
		Operations: operationslister.NewOperationLister(indexer),
		Clusters:   clusters,
	}

	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: operation.Name}}); err != nil {
		t.Fatalf("reconcile terminal operation with deleted target: %v", err)
	}
}
