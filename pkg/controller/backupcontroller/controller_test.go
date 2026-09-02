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

package backupcontroller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestMapObjectsForOperationUsesOperationNameIndex(t *testing.T) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		OperationNameIndex: func(raw any) ([]string, error) {
			backup, ok := raw.(*corev1.Backup)
			if !ok || backup.Labels == nil {
				return nil, nil
			}
			return []string{backup.Labels[common.LabelOperationName]}, nil
		},
	})
	for _, backup := range []*corev1.Backup{
		{ObjectMeta: metav1.ObjectMeta{Name: "matching", Labels: map[string]string{common.LabelOperationName: "operation-1"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "other", Labels: map[string]string{common.LabelOperationName: "operation-2"}}},
	} {
		if err := indexer.Add(backup); err != nil {
			t.Fatal(err)
		}
	}

	requests := mapObjectsForOperation(indexer)(&operations.Operation{ObjectMeta: metav1.ObjectMeta{Name: "operation-1"}})
	if len(requests) != 1 || requests[0].Name != "matching" {
		t.Fatalf("requests = %#v, want only matching backup", requests)
	}
}
