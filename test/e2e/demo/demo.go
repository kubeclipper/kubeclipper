/*
 *
 *  * Copyright 2021 KubeClipper Authors.
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

package demo

import (
	"encoding/json"
	goruntime "runtime"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	jsonpatch "github.com/evanphx/json-patch"

	"github.com/kubeclipper/kubeclipper/test/framework"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var (
	t1 = time.Now()
	t2 = t1.Add(1 * time.Minute)
)

var _ = SIGDescribe("demo", func() {
	ginkgo.It("should merge node status successfully", func() {
		ginkgo.By("Creating a merge patch")
		originNode := newOriginNode()
		originalNodeBytes, err := json.Marshal(originNode)
		framework.ExpectNoError(err, "failed to marshal origin node")
		targetNode := newTargetNode()
		quantity := targetNode.Status.Capacity[v1.ResourceCPU]
		framework.Logf("node cpu %d", quantity.Value())
		targetNodeBytes, err := json.Marshal(targetNode)
		framework.ExpectNoError(err, "failed to marshal target node")
		patch, err := jsonpatch.CreateMergePatch(originalNodeBytes, targetNodeBytes)
		framework.ExpectNoError(err, "failed to create node merge patch")
		ginkgo.By("Apply a merge patch")
		modifiedAlternative, err := jsonpatch.MergePatch(originalNodeBytes, patch)
		framework.ExpectNoError(err, "failed to apply node merge patch")
		ginkgo.By("Ensure patch is successfully")
		// should be compare node struct instead of []byte
		framework.ExpectEqual(modifiedAlternative, modifiedAlternative, "expected node %s, but got %s", modifiedAlternative, modifiedAlternative)
	})
	// Add other test case
})

func newOriginNode() *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "947627ea-160f-48e2-9a90-07a9739425aa",
			Labels: map[string]string{
				common.LabelOSStable:   goruntime.GOOS,
				common.LabelArchStable: goruntime.GOARCH,
			},
		},
		ProxyIpv4CIDR: "",
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{
					Type:               v1.NodeReady,
					Status:             v1.ConditionTrue,
					Reason:             "KiAgentReady",
					Message:            "ki agent is posting ready status",
					LastHeartbeatTime:  metav1.NewTime(t1),
					LastTransitionTime: metav1.NewTime(t1),
				},
			},
			Addresses: nil,
			NodeInfo:  v1.NodeSystemInfo{},
		},
	}
}

func newTargetNode() *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "947627ea-160f-48e2-9a90-07a9739425aa",
			Labels: map[string]string{
				common.LabelOSStable:   goruntime.GOOS,
				common.LabelArchStable: goruntime.GOARCH,
				common.LabelHostname:   "worker",
			},
		},
		ProxyIpv4CIDR: "10.0.0.0/32",
		Status: v1.NodeStatus{
			Capacity: v1.ResourceList{
				v1.ResourceCPU: *resource.NewMilliQuantity(4*1000, resource.DecimalSI),
			},
			Allocatable: nil,
			Conditions: []v1.NodeCondition{
				{
					Type:               v1.NodeReady,
					Status:             v1.ConditionTrue,
					Reason:             "KiAgentReady",
					Message:            "ki agent is posting ready status",
					LastHeartbeatTime:  metav1.NewTime(t2),
					LastTransitionTime: metav1.NewTime(t1),
				},
			},
			Addresses: nil,
			NodeInfo:  v1.NodeSystemInfo{},
		},
	}
}
