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

package cluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/server/runtime"

	"github.com/google/uuid"

	"github.com/kubeclipper/kubeclipper/pkg/models"

	"github.com/onsi/ginkgo"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

var _ = SIGDescribe("cluster", newClusterFunc)

var (
	apiPath      = fmt.Sprintf("%s/%s/%s/", runtime.APIRootPath, v1.SchemeGroupVersion.Group, v1.SchemeGroupVersion.Version)
	apiCluster   = "clusters"
	apiNode      = "nodes"
	apiOperation = "operations"
)

// dist/e2e.test --server-address 172.18.94.111:8080 --registry 172.18.94.144:5000 --vip 169.254.169.100
func newClusterFunc() {
	ginkgo.It("should list-node status successfully", func() {
		ginkgo.By("request list node data")
		listNode()
	})

	ginkgo.It("should create-cluster status successfully", func() {
		ginkgo.By("request get origin create-cluster data")
		createCluster()
	})

	ginkgo.It("should list-operation status successfully", func() {
		ginkgo.By("request get list-operation data")
		listOpertion()
	})
}

func listNode() (int, []v1.Node) {
	url := fmt.Sprintf("http://%s%s%s", framework.TestContext.Host, apiPath, apiNode)
	resp, err := http.DefaultClient.Get(url)
	framework.ExpectNoError(err, "failed to get node list by calling ", url)
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	framework.ExpectNoError(err, "failed to read response body of node list request")
	n := &models.PageableResponse{}
	err = json.Unmarshal(respData, n)
	framework.ExpectNoError(err, "failed to unmarshal response body of node list request")

	return parseNodeResponse(n)
}

func parseNodeResponse(node *models.PageableResponse) (int, []v1.Node) {
	nodes := []v1.Node{}

	nodeBytes, err := json.Marshal(&node.Items)
	framework.ExpectNoError(err, "failed to marshal node data struct")
	err = json.Unmarshal(nodeBytes, &nodes)
	framework.ExpectNoError(err, "failed to unmarshal node data struct")

	return node.TotalCount, nodes
}

func createCluster() {
	total, nodeList := listNode()
	if total == 0 {
		framework.ExpectError(nil, "no available nodes found in collection")
	}

	index := 0
	for i, item := range nodeList {
		if _, ok := item.Labels["kubeclipper.io/nodeRole"]; !ok {
			index = i
		}
	}

	url, reqbody := newRequestCreateCluster([]string{nodeList[index].Name}, []string{},
		framework.TestContext.LocalRegistry, framework.TestContext.WorkerNodeVip,
		framework.TestContext.ServiceSubnet, framework.TestContext.PodSubnet)
	resp, err := http.DefaultClient.Post(url, "application/json", strings.NewReader(string(reqbody)))
	framework.ExpectNoError(err, "failed to create cluster by calling ", url)
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	framework.ExpectNoError(err, "failed to read response body of create cluster request")

	c := &v1.Cluster{}
	err = json.Unmarshal(respData, c)
	framework.ExpectNoError(err, "failed to unmarshal cluster data struct: ", string(respData))

	op := v1.Operation{}
	for forNumber := 1; forNumber < 30; forNumber++ {
		<-time.After(10 * time.Second)
		_, operations := listOpertion()
		for _, v := range operations {
			if v.Labels["kubeclipper.io/cluster"] == c.Name {
				op = v
				continue
			}
		}
		if op.Status.Status != v1.OperationStatusRunning && op.Status.Status != "" {
			break
		}
	}

	if op.Status.Status != v1.OperationStatusSuccessful {
		framework.ExpectError(nil, "cluster setup appears to be unsuccessful, got cluster status: ", op.Status.Status)
	}
}

func newRequestCreateCluster(masters, workers []string, registry, vip, svcSubnet, podSubnet string) (string, []byte) {
	url := fmt.Sprintf("http://%s%s%s", framework.TestContext.Host, apiPath, apiCluster)
	var (
		mNodes v1.WorkerNodeList
		nNodes v1.WorkerNodeList
	)
	for _, v := range masters {
		mNodes = append(mNodes, v1.WorkerNode{
			ID:     v,
			Labels: nil,
			Taints: nil,
		})
	}
	for _, v := range masters {
		nNodes = append(nNodes, v1.WorkerNode{
			ID:     v,
			Labels: nil,
			Taints: nil,
		})
	}

	body := &v1.Cluster{
		TypeMeta:   metav1.TypeMeta{Kind: "Cluster", APIVersion: "core.kubeclipper.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("demo-create-cluster-%s", uuid.New().String()[:8])},
		Kubeadm: &v1.Kubeadm{
			Masters:           mNodes,
			Workers:           nNodes,
			KubernetesVersion: "v1.20.13",
			CertSANs:          []string{},
			LocalRegistry:     registry,
			ContainerRuntime: v1.ContainerRuntime{Type: "docker",
				Docker: v1.Docker{Version: "19.03.12", DataRootDir: "", InsecureRegistry: []string{registry}}},
			Networking: v1.Networking{ServiceSubnet: svcSubnet, PodSubnet: podSubnet, DNSDomain: "cluster.local"},
			KubeComponents: v1.KubeComponents{
				KubeProxy: v1.KubeProxy{IPvs: true},
				Etcd:      v1.Etcd{},
				CNI: v1.CNI{LocalRegistry: registry, Type: "calico", PodIPv4CIDR: podSubnet, PodIPv6CIDR: "", MTU: 1440,
					Calico: v1.Calico{IPv4AutoDetection: "first-found", IPv6AutoDetection: "first-found",
						Mode: "Overlay-Vxlan-All", DualStack: false, IPManger: true, Version: "v3.11.2"}},
			},
			Components:    []v1.Component{},
			WorkerNodeVip: vip,
		},
	}
	cBytes, err := json.Marshal(body)
	framework.ExpectNoError(err, "failed to marshal origin create cluster data struct")
	return url, cBytes
}

func listOpertion() (int, []v1.Operation) {
	url := fmt.Sprintf("http://%s%s%s", framework.TestContext.Host, apiPath, apiOperation)
	resp, err := http.DefaultClient.Get(url)
	framework.ExpectNoError(err, "list operation faild")
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	framework.ExpectNoError(err, "failed to read response body of operation list request")
	n := &models.PageableResponse{}
	err = json.Unmarshal(respData, n)
	framework.ExpectNoError(err, "failed to unmarshal response body of node list request")

	return parseOpertionResponse(n)
}

func parseOpertionResponse(opertion *models.PageableResponse) (int, []v1.Operation) {
	opertions := []v1.Operation{}

	nodeBytes, err := json.Marshal(&opertion.Items)
	framework.ExpectNoError(err, "failed to marshal operation data struct")
	err = json.Unmarshal(nodeBytes, &opertions)
	framework.ExpectNoError(err, "failed to unmarshal operation data struct")

	return opertion.TotalCount, opertions
}
