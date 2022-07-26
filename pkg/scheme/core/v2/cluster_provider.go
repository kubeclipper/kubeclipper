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

package v2

import (
	"context"

	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Kubeadm struct {
}

type Rancher struct {
	APIEndpoint string `json:"apiEndpoint"`
	User        string `json:"user"`
	Token       string `json:"token"`
	APIVersion  string `json:"apiVersion"`
}

var (
	AllowedProvider = sets.NewString("Kubeadm", "Rancher")
)

type ProviderSpec struct {
	Name            string
	ExternalKubeadm *Kubeadm
	Rancher         *Rancher
}

type Interface interface {
	// need node ip:id
	Import(ctx context.Context) ([]Cluster, error)
	CreateClusterOperation(c *Cluster) (v1.Operation, error)
	DeleteClusterOperation(c *Cluster) (v1.Operation, error)
	AddNodesOperation(c *Cluster, nodes WorkerNodeList) (v1.Operation, error)
	RemoveNodesOperation(c *Cluster, nodes WorkerNodeList) (v1.Operation, error)
}

func ImportCluster() {
	// unmarshall body parameter to Kubeadm or Rancher
	// k := Kubeadm{} or r := Rancher{}

	// Check network and API is ok...
	// ...

	// Import Cluster
	// clus := k.Import() or r.Import()

	// Cluster Controller will Reconcile Cluster
}

// k8s节点信息里面要包含kc-agent的id

//

// Stage1

// Stage2

/*
确定的信息如下:
	1. kc-agent 部署的时候支持元数据注入， 将现有的 region 信息作为元数据之一
	2. kc-server 需要添加一个配置下发的接口，添加 kc-agent或者 kc-server时，访问接口获取配置信息
待预研:
	1. (kc-agent无法直接和 kc-server互相访问)
		1. kc-agent 反向代理或其他方式 (Konnectivity)
	2. 	k8s 集群 node 的元数据中要有 kc-agent 信息，怎么注入？
*/
