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

package nodecontroller

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type missingClusterLister struct{}

func (missingClusterLister) List(_ labels.Selector) ([]*corev1.Cluster, error) {
	return nil, nil
}

type clusterLister struct{ cluster *corev1.Cluster }

func (l clusterLister) List(_ labels.Selector) ([]*corev1.Cluster, error) {
	return []*corev1.Cluster{l.cluster}, nil
}
func (l clusterLister) Get(_ string) (*corev1.Cluster, error) { return l.cluster, nil }
func (missingClusterLister) Get(name string) (*corev1.Cluster, error) {
	return nil, errors.NewNotFound(schema.GroupResource{Group: corev1.GroupName, Resource: "clusters"}, name)
}

type recordingNodeWriter struct{ updated *corev1.Node }

func (w *recordingNodeWriter) UpdateNode(_ context.Context, node *corev1.Node) (*corev1.Node, error) {
	w.updated = node.DeepCopy()
	return node, nil
}

func (*recordingNodeWriter) CreateNode(_ context.Context, _ *corev1.Node) (*corev1.Node, error) {
	panic("unexpected call")
}

func (*recordingNodeWriter) DeleteNode(context.Context, string) error {
	panic("unexpected call")
}

func TestSyncNodeRoleClearsOrphanedClusterOwnership(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
		common.LabelClusterName:    "deleted-cluster",
		common.LabelNodeRole:       string(common.NodeRoleWorker),
		common.LabelTopologyRegion: "default",
	}}}
	writer := &recordingNodeWriter{}
	r := &NodeReconciler{ClusterLister: missingClusterLister{}, NodeWriter: writer}

	if err := r.syncNodeRole(context.Background(), node); err != nil {
		t.Fatalf("syncNodeRole() error = %v", err)
	}
	if writer.updated == nil {
		t.Fatal("syncNodeRole() did not update the node")
	}
	if _, ok := writer.updated.Labels[common.LabelClusterName]; ok {
		t.Fatalf("cluster ownership label was not removed: %#v", writer.updated.Labels)
	}
	if _, ok := writer.updated.Labels[common.LabelNodeRole]; ok {
		t.Fatalf("node role label was not removed: %#v", writer.updated.Labels)
	}
	if writer.updated.Labels[common.LabelTopologyRegion] != "default" {
		t.Fatalf("unrelated labels were changed: %#v", writer.updated.Labels)
	}
}

func TestSyncNodeRoleClearsNodeRemovedFromCluster(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "removed-worker", Labels: map[string]string{
		common.LabelClusterName: "cluster-a",
		common.LabelNodeRole:    string(common.NodeRoleWorker),
	}}}
	writer := &recordingNodeWriter{}
	r := &NodeReconciler{
		ClusterLister: clusterLister{cluster: &corev1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-a"}}},
		NodeWriter:    writer,
	}

	if err := r.syncNodeRole(context.Background(), node); err != nil {
		t.Fatalf("syncNodeRole() error = %v", err)
	}
	if writer.updated == nil {
		t.Fatal("syncNodeRole() did not update the node")
	}
	if _, ok := writer.updated.Labels[common.LabelClusterName]; ok {
		t.Fatalf("cluster ownership label was not removed: %#v", writer.updated.Labels)
	}
	if _, ok := writer.updated.Labels[common.LabelNodeRole]; ok {
		t.Fatalf("node role label was not removed: %#v", writer.updated.Labels)
	}
}
