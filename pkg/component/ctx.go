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

package component

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type (
	extraKey     struct{}
	metaKey      struct{}
	operationKey struct{}
	stepKey      struct{}
	oplogKey     struct{}
	retryKey     struct{}
	repoMirror   struct{}
)

type ExtraMetadata struct {
	Masters                   NodeList
	Workers                   NodeList
	ClusterStatus             v1.ClusterPhase
	Offline                   bool
	LocalRegistry             string
	CRI                       string
	ClusterName               string
	KubeVersion               string
	OperationID               string
	OperationType             string
	KubeletDataDir            string
	ControlPlaneStatus        []v1.ControlPlaneHealth
	Addons                    []v1.Addon
	CNI                       string
	CNINamespace              string
	OnlyInstallKubernetesComp bool
}

type Node struct {
	ID       string
	IPv4     string
	NodeIPv4 string
	Region   string
	Hostname string
	Role     string
	Disable  bool
}

type NodeList []Node

func (l NodeList) GetNodeIDs() (nodes []string) {
	for _, node := range l {
		nodes = append(nodes, node.ID)
	}
	return
}

func (l NodeList) AvailableKubeMasters() (NodeList, error) {
	masters := l.ReachableNodes("tcp", 6443, time.Second*2)
	if len(masters) == 0 {
		return nil, fmt.Errorf("no master node available: all master nodes 6443 port unreachable")
	}

	return masters, nil
}

func (l NodeList) ReachableNodes(protocol string, port int, timeout time.Duration) NodeList {
	wg := sync.WaitGroup{}
	ch := make(chan Node, len(l))
	defer close(ch)
	for _, no := range l {
		wg.Add(1)
		go func(no Node) {
			defer wg.Done()
			if netutil.Reachable(protocol, no.IPv4+":"+strconv.Itoa(port), timeout) == nil {
				ch <- no
				return
			}
		}(no)
	}
	wg.Wait()
	var list NodeList
	length := len(ch)
	for i := 0; i < length; i++ {
		list = append(list, <-ch)
	}
	return list
}

func (e ExtraMetadata) GetAllNodeIDs() []string {
	var nodes []string
	nodes = append(nodes, e.GetMasterNodeIDs()...)
	nodes = append(nodes, e.GetWorkerNodeIDs()...)
	return nodes
}

func (e ExtraMetadata) GetAllNodes() (nodes NodeList) {
	nodes = append(nodes, e.Masters...)
	nodes = append(nodes, e.Workers...)
	return nodes
}

func (e ExtraMetadata) GetMasterHostname(id string) string {
	for _, node := range e.Masters {
		if node.ID == id {
			return node.Hostname
		}
	}
	return ""
}

func (e ExtraMetadata) GetWorkerHostname(id string) string {
	for _, node := range e.Workers {
		if node.ID == id {
			return node.Hostname
		}
	}
	return ""
}

// GetMasterNodeIDs
// Deprecated. Use GetAvailableMasterNodes() instead
func (e ExtraMetadata) GetMasterNodeIDs() []string {
	var nodes []string
	for _, node := range e.Masters {
		nodes = append(nodes, node.ID)
	}
	return nodes
}

func (e ExtraMetadata) GetWorkerNodeIDs() []string {
	var nodes []string
	for _, node := range e.Workers {
		nodes = append(nodes, node.ID)
	}
	return nodes
}

func (e ExtraMetadata) GetMasterNodeIP() map[string]string {
	nodes := make(map[string]string)
	for _, node := range e.Masters {
		nodes[node.ID] = node.IPv4
	}
	return nodes
}

func (e ExtraMetadata) GetWorkerNodeIP() map[string]string {
	nodes := make(map[string]string)
	for _, node := range e.Workers {
		nodes[node.ID] = node.IPv4
	}
	return nodes
}

func (e ExtraMetadata) GetMasterNodeClusterIP() map[string]string {
	nodes := make(map[string]string)
	for _, node := range e.Masters {
		nodes[node.ID] = node.NodeIPv4
	}
	return nodes
}

func (e ExtraMetadata) GetWorkerNodeClusterIP() map[string]string {
	nodes := make(map[string]string)
	for _, node := range e.Workers {
		nodes[node.ID] = node.NodeIPv4
	}
	return nodes
}

func (e ExtraMetadata) GetAvailableMasterNodes() []string {
	var nodes []string
	for _, node := range e.ControlPlaneStatus {
		if node.Status == v1.ComponentHealthy {
			nodes = append(nodes, node.ID)
		}
	}
	return nodes
}

func (e ExtraMetadata) IsAllMasterAvailable() bool {
	for _, node := range e.ControlPlaneStatus {
		if node.Status != v1.ComponentHealthy {
			return false
		}
	}
	return true
}

func WithExtraData(ctx context.Context, data []byte) context.Context {
	return context.WithValue(ctx, extraKey{}, data)
}

func GetExtraData(ctx context.Context) []byte {
	if v := ctx.Value(extraKey{}); v != nil {
		return v.([]byte)
	}
	return nil
}

func WithExtraMetadata(ctx context.Context, metadata ExtraMetadata) context.Context {
	return context.WithValue(ctx, metaKey{}, metadata)
}

func GetExtraMetadata(ctx context.Context) ExtraMetadata {
	if v := ctx.Value(metaKey{}); v != nil {
		return v.(ExtraMetadata)
	}
	return ExtraMetadata{}
}

func WithOperationID(ctx context.Context, opID string) context.Context {
	return context.WithValue(ctx, operationKey{}, opID)
}

func WithStepID(ctx context.Context, stepID string) context.Context {
	return context.WithValue(ctx, stepKey{}, stepID)
}

func GetOperationID(ctx context.Context) string {
	if v := ctx.Value(operationKey{}); v != nil {
		return v.(string)
	}
	return ""
}

func GetStepID(ctx context.Context) string {
	if v := ctx.Value(stepKey{}); v != nil {
		return v.(string)
	}
	return ""
}

func WithOplog(ctx context.Context, ol OperationLogFile) context.Context {
	return context.WithValue(ctx, oplogKey{}, ol)
}

func GetOplog(ctx context.Context) OperationLogFile {
	if v := ctx.Value(oplogKey{}); v != nil {
		return v.(OperationLogFile)
	}
	return nil
}

func WithRetry(ctx context.Context, retry bool) context.Context {
	return context.WithValue(ctx, retryKey{}, retry)
}

func GetRetry(ctx context.Context) bool {
	if v := ctx.Value(retryKey{}); v != nil {
		return v.(bool)
	}
	return false
}

func WithRepoMirror(ctx context.Context, mirror string) context.Context {
	return context.WithValue(ctx, repoMirror{}, mirror)
}

func GetRepoMirror(ctx context.Context) string {
	if v := ctx.Value(repoMirror{}); v != nil {
		return v.(string)
	}
	return ""
}
