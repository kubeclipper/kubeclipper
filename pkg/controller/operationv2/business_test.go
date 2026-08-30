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
	"errors"
	"fmt"
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

func TestFailedClusterPhaseForNodeOperationsKeepsClusterRunning(t *testing.T) {
	tests := []struct {
		action string
	}{
		{action: corev1.OperationAddNodes},
		{action: corev1.OperationRemoveNodes},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			if got := failedClusterPhase(tt.action); got != corev1.ClusterRunning {
				t.Fatalf("failedClusterPhase(%q) = %q, want %q", tt.action, got, corev1.ClusterRunning)
			}
		})
	}
}

func TestBusinessReconcileRetriesClusterUpdateConflict(t *testing.T) {
	const clusterUID = types.UID("cluster-uid")
	operation := &operations.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "add-nodes"},
		Spec: operations.OperationSpec{
			Action: corev1.OperationAddNodes,
			TargetRef: operations.ObjectReference{
				Name: "cluster-a",
				UID:  clusterUID,
			},
		},
		Status: operations.OperationStatus{Phase: operations.OperationTimedOut},
	}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := indexer.Add(operation); err != nil {
		t.Fatal(err)
	}

	clusterObject := &corev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a", UID: clusterUID},
		Status:     corev1.ClusterStatus{Phase: corev1.ClusterUpdateFailed},
	}
	controller := gomock.NewController(t)
	clusters := clustermock.NewMockOperator(controller)
	clusters.EXPECT().GetClusterEx(gomock.Any(), "cluster-a", "").Return(clusterObject, nil).Times(3)
	clusters.EXPECT().UpdateCluster(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *corev1.Cluster) (*corev1.Cluster, error) {
			if c.Status.Phase != corev1.ClusterRunning {
				t.Fatalf("first update phase=%s, want Running", c.Status.Phase)
			}
			return nil, apierrors.NewConflict(corev1.Resource("cluster"), c.Name, fmt.Errorf("stale object"))
		},
	).Times(1)
	clusters.EXPECT().UpdateCluster(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *corev1.Cluster) (*corev1.Cluster, error) {
			if c.Status.Phase != corev1.ClusterRunning {
				t.Fatalf("retry update phase=%s, want Running", c.Status.Phase)
			}
			return c, nil
		},
	).Times(1)

	reconciler := &BusinessReconciler{
		Operations: operationslister.NewOperationLister(indexer),
		Clusters:   clusters,
	}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: operation.Name}}); err != nil {
		t.Fatalf("reconcile conflict: %v", err)
	}
}

type recordingCleaner struct {
	uid   types.UID
	err   error
	calls int
}

func (r *recordingCleaner) CleanupByTargetUID(_ context.Context, targetUID types.UID) error {
	r.calls++
	r.uid = targetUID
	return r.err
}

// A succeeded delete-cluster operation must purge the target's history before
// the Cluster object is removed, and must keep the cluster when the purge
// cannot confirm completion yet.
func TestBusinessReconcileCleansHistoryBeforeClusterDelete(t *testing.T) {
	newOperation := func() *operations.Operation {
		return &operations.Operation{
			ObjectMeta: metav1.ObjectMeta{Name: "delete-cluster"},
			Spec: operations.OperationSpec{
				Action:    corev1.OperationDeleteCluster,
				TargetRef: operations.ObjectReference{Kind: "Cluster", Name: "cluster", UID: "target-uid"},
			},
			Status: operations.OperationStatus{Phase: operations.OperationSucceeded},
		}
	}
	clusterObject := &corev1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", UID: "target-uid"}}

	t.Run("deletes cluster after successful cleanup", func(t *testing.T) {
		operation := newOperation()
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		if err := indexer.Add(operation); err != nil {
			t.Fatal(err)
		}
		controller := gomock.NewController(t)
		clusters := clustermock.NewMockOperator(controller)
		clusters.EXPECT().GetClusterEx(gomock.Any(), "cluster", "").Return(clusterObject, nil)
		clusters.EXPECT().DeleteCluster(gomock.Any(), "cluster").Return(nil)
		cleaner := &recordingCleaner{}
		reconciler := &BusinessReconciler{
			Operations: operationslister.NewOperationLister(indexer),
			Clusters:   clusters,
			Cleaner:    cleaner,
		}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: operation.Name}}); err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if cleaner.calls != 1 || cleaner.uid != "target-uid" {
			t.Fatalf("cleaner calls=%d uid=%s, want 1 call for target-uid", cleaner.calls, cleaner.uid)
		}
	})

	t.Run("keeps cluster while cleanup fails", func(t *testing.T) {
		operation := newOperation()
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		if err := indexer.Add(operation); err != nil {
			t.Fatal(err)
		}
		controller := gomock.NewController(t)
		clusters := clustermock.NewMockOperator(controller)
		clusters.EXPECT().GetClusterEx(gomock.Any(), "cluster", "").Return(clusterObject, nil)
		cleaner := &recordingCleaner{err: errors.New("task task-1 is not terminal")}
		reconciler := &BusinessReconciler{
			Operations: operationslister.NewOperationLister(indexer),
			Clusters:   clusters,
			Cleaner:    cleaner,
		}
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: operation.Name}}); err == nil {
			t.Fatal("expected the reconcile error so the workqueue retries")
		}
		if cleaner.calls != 1 {
			t.Fatalf("cleaner calls = %d, want 1", cleaner.calls)
		}
	})
}
