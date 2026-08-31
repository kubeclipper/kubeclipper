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
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	clustermock "github.com/kubeclipper/kubeclipper/pkg/models/cluster/mock"
	operationv2store "github.com/kubeclipper/kubeclipper/pkg/models/operationv2"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

type recordingOperationStore struct {
	operationv2store.Store
	cleanupErr   error
	cleanupUID   types.UID
	cleanupCalls int
}

func (s *recordingOperationStore) CleanupByTargetUID(_ context.Context, targetUID types.UID) error {
	s.cleanupCalls++
	s.cleanupUID = targetUID
	return s.cleanupErr
}

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

func TestFinalizeClusterCleansOperationHistoryBeforeReleasingFinalizer(t *testing.T) {
	clusterObject := &v1.Cluster{ObjectMeta: metav1.ObjectMeta{
		Name:       "cluster-a",
		UID:        "cluster-uid",
		Finalizers: []string{v1.ClusterFinalizer},
	}}

	t.Run("cleans history before releasing finalizer", func(t *testing.T) {
		controller := gomock.NewController(t)
		store := &recordingOperationStore{}
		cronBackups := clustermock.NewMockCronBackupWriter(controller)
		clusterWriter := clustermock.NewMockClusterWriter(controller)
		cronBackups.EXPECT().DeleteCronBackupCollection(gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, any) error {
				if store.cleanupCalls != 1 {
					t.Fatalf("cleanup calls = %d before cron backup cleanup, want 1", store.cleanupCalls)
				}
				return nil
			},
		)
		clusterWriter.EXPECT().UpdateCluster(gomock.Any(), clusterObject).DoAndReturn(
			func(context.Context, *v1.Cluster) (*v1.Cluster, error) {
				if store.cleanupCalls != 1 {
					t.Fatalf("cleanup calls = %d before finalizer removal, want 1", store.cleanupCalls)
				}
				for _, finalizer := range clusterObject.Finalizers {
					if finalizer == v1.ClusterFinalizer {
						t.Fatal("cluster finalizer was not removed")
					}
				}
				return clusterObject, nil
			},
		)

		reconciler := &ClusterReconciler{
			OperationStore:   store,
			CronBackupWriter: cronBackups,
			ClusterWriter:    clusterWriter,
		}
		if err := reconciler.finalizeCluster(context.Background(), clusterObject); err != nil {
			t.Fatalf("finalize cluster: %v", err)
		}
		if store.cleanupUID != clusterObject.UID {
			t.Fatalf("cleanup UID = %q, want %q", store.cleanupUID, clusterObject.UID)
		}
	})

	t.Run("keeps finalizer when history cleanup fails", func(t *testing.T) {
		cluster := clusterObject.DeepCopy()
		cluster.Finalizers = []string{v1.ClusterFinalizer}
		store := &recordingOperationStore{cleanupErr: errors.New("operation still active")}
		reconciler := &ClusterReconciler{OperationStore: store}
		if err := reconciler.finalizeCluster(context.Background(), cluster); err == nil {
			t.Fatal("finalize cluster succeeded while operation cleanup failed")
		}
		if store.cleanupCalls != 1 {
			t.Fatalf("cleanup calls = %d, want 1", store.cleanupCalls)
		}
		if len(cluster.Finalizers) != 1 || cluster.Finalizers[0] != v1.ClusterFinalizer {
			t.Fatalf("finalizers = %v, want cluster finalizer retained", cluster.Finalizers)
		}
	})
}
