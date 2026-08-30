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

package agent

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	operationv2 "github.com/kubeclipper/kubeclipper/pkg/agent/operationv2"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/nodestatus"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

type Server struct {
	worker    *operationv2.Worker
	logServer *operationv2.LogServer
	client    clientset.Interface
	Config    *config.Config
}

type taskStatusClient interface {
	Get(context.Context, string, metav1.GetOptions) (*operations.OperationTask, error)
	List(context.Context, metav1.ListOptions) (*operations.OperationTaskList, error)
	Watch(context.Context, metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(context.Context, *operations.OperationTask, metav1.UpdateOptions) (*operations.OperationTask, error)
}

type taskClientAdapter struct{ client taskStatusClient }

func (a taskClientAdapter) Get(ctx context.Context, name string, options metav1.GetOptions) (*operations.OperationTask, error) {
	return a.client.Get(ctx, name, options)
}

func (a taskClientAdapter) List(ctx context.Context, options *metav1.ListOptions) (*operations.OperationTaskList, error) {
	return a.client.List(ctx, *options)
}

func (a taskClientAdapter) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return a.client.Watch(ctx, *options)
}

func (a taskClientAdapter) UpdateStatus(ctx context.Context, task *operations.OperationTask) (*operations.OperationTask, error) {
	return a.client.UpdateStatus(ctx, task, metav1.UpdateOptions{})
}

const (
	nodeStatusUpdateTimeout   = 30 * time.Second
	nodeLeaseNamespace        = "node-lease"
	nodeLeaseDurationSeconds  = int32(240)
	nodeLeaseRenewInterval    = time.Duration(nodeLeaseDurationSeconds) * time.Second / 4
	nodeLeaseMaxUpdateRetries = 5
)

func (s *Server) PrepareRun(stopCh <-chan struct{}) error {
	opLog, err := oplog.NewOperationLog(s.Config.OpLogOptions)
	if err != nil {
		return err
	}
	if s.Config.APIServer == nil {
		return fmt.Errorf("apiServer configuration is required")
	}
	// No client-wide Timeout: http.Client.Timeout covers the whole response
	// body and would sever the informer's long-lived task watch on every
	// interval. One-shot calls carry per-call deadlines instead
	// (nodeStatusUpdateTimeout here, serverCallTimeout in the task worker).
	restConfig := &rest.Config{
		Host: s.Config.APIServer.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: s.Config.APIServer.CAFile, CertFile: s.Config.APIServer.CertFile,
			KeyFile: s.Config.APIServer.KeyFile, ServerName: s.Config.APIServer.ServerName,
		},
		QPS: 20, Burst: 40,
	}
	client, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kc-server client: %w", err)
	}
	s.client = client
	nodeCtx, nodeCancel := context.WithTimeout(context.Background(), nodeStatusUpdateTimeout)
	defer nodeCancel()
	node, err := client.CoreV1().Nodes().Get(nodeCtx, s.Config.AgentID, metav1.GetOptions{})
	if apierrors.IsNotFound(err) && s.Config.RegisterNode {
		node, err = client.CoreV1().Nodes().Create(nodeCtx, s.initialNode(), &metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			node, err = client.CoreV1().Nodes().Get(nodeCtx, s.Config.AgentID, metav1.GetOptions{})
		}
	}
	if err != nil {
		return fmt.Errorf("get registered Node %q: %w", s.Config.AgentID, err)
	}
	registry := operationv2.NewRegistry()
	if registerErr := registry.Register(operationv2.NoopExecutorName, operationv2.NoopExecutor{}); registerErr != nil {
		return registerErr
	}
	if registerErr := registry.Register(operationv2.NodePreflightExecutorName, operationv2.NodePreflight{}); registerErr != nil {
		return registerErr
	}
	if registerErr := registry.Register(operationv2.CommandStepExecutorName, operationv2.CommandStepExecutor{
		OpLog: opLog, RepoMirror: s.Config.ImageProxyOptions.KcImageRepoMirror,
	}); registerErr != nil {
		return registerErr
	}
	s.worker, err = operationv2.NewWorker(&operationv2.WorkerOptions{
		AgentID: s.Config.AgentID, NodeUID: node.UID,
		Client: taskClientAdapter{client: client.OperationsV1alpha1().OperationTasks()}, Registry: registry, OpLog: opLog,
	})
	if err != nil {
		return err
	}
	if prepareErr := s.worker.PrepareRun(stopCh); prepareErr != nil {
		return prepareErr
	}
	s.logServer, err = operationv2.NewLogServer(&operationv2.LogServerOptions{
		Address: s.Config.APIServer.LogAddress, TLSCertFile: s.Config.APIServer.CertFile,
		TLSKeyFile: s.Config.APIServer.KeyFile, ClientCAFile: s.Config.APIServer.CAFile,
		ExpectedClientCommonName: operationv2.DefaultServerClientIdentity, Logs: opLog,
	})
	if err != nil {
		s.worker.Close()
		return err
	}
	if err := s.logServer.PrepareRun(stopCh); err != nil {
		s.worker.Close()
		return err
	}
	return nil
}

func (s *Server) initialNode() *corev1.Node {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = s.Config.AgentID
	}
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: corev1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: s.Config.AgentID, Labels: map[string]string{
			common.LabelOSStable: runtime.GOOS, common.LabelArchStable: runtime.GOARCH,
			common.LabelTopologyRegion: s.Config.Metadata.Region, common.LabelHostname: hostname,
		}},
		Status: corev1.NodeStatus{
			AgentLogPort: s.agentLogPort(),
			NodeInfo:     corev1.NodeSystemInfo{Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH},
		},
	}
	if ip, err := netutil.GetDefaultIP(true, s.Config.IPDetect); err == nil {
		node.Status.Ipv4DefaultIP = ip.String()
		node.Status.NodeIpv4DefaultIP = ip.String()
		node.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: ip.String()}}
	}
	if gateway, err := netutil.GetDefaultGateway(true); err == nil {
		node.Status.Ipv4DefaultGw = gateway.String()
	}
	if err := nodestatus.MachineInfo()(node); err != nil {
		logger.Errorf("collect node machine information: %v", err)
	}
	return node
}

func (s *Server) agentLogPort() int32 {
	address := operationv2.DefaultAgentLogAddress
	if s.Config != nil && s.Config.APIServer != nil && s.Config.APIServer.LogAddress != "" {
		address = s.Config.APIServer.LogAddress
	}
	_, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return 0
	}
	port, err := strconv.ParseInt(rawPort, 10, 32)
	if err != nil || port < 1 || port > 65535 {
		return 0
	}
	return int32(port)
}

func (s *Server) Run(stopCh <-chan struct{}) error {
	if err := s.worker.Run(stopCh); err != nil {
		return err
	}
	if err := s.logServer.Run(stopCh); err != nil {
		return err
	}
	go s.reportNodeStatus(stopCh)
	go s.reportNodeLease(stopCh)
	<-stopCh
	logger.Debugf("get stopCh signal, exit...")
	s.logServer.Close()
	s.worker.Close()
	return nil
}

func (s *Server) reportNodeLease(stopCh <-chan struct{}) {
	ticker := time.NewTicker(nodeLeaseRenewInterval)
	defer ticker.Stop()
	for {
		if err := s.renewNodeLease(); err != nil {
			logger.Errorf("renew Node Lease: %v", err)
		}
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) renewNodeLease() error {
	ctx, cancel := context.WithTimeout(context.Background(), nodeStatusUpdateTimeout)
	defer cancel()

	leases := s.client.CoreV1().Leases()
	for range nodeLeaseMaxUpdateRetries {
		lease, err := leases.GetWithNamespace(ctx, s.Config.AgentID, nodeLeaseNamespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = leases.Create(ctx, s.newNodeLease(nil), &metav1.CreateOptions{})
			if apierrors.IsAlreadyExists(err) {
				continue
			}
			return err
		}
		if err != nil {
			return err
		}
		_, err = leases.Update(ctx, s.newNodeLease(lease), &metav1.UpdateOptions{})
		if apierrors.IsConflict(err) {
			continue
		}
		return err
	}
	return fmt.Errorf("renew Node Lease %q: too many resource version conflicts", s.Config.AgentID)
}

func (s *Server) newNodeLease(base *coordinationv1.Lease) *coordinationv1.Lease {
	var lease *coordinationv1.Lease
	if base == nil {
		lease = &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{
			Name: s.Config.AgentID, Namespace: nodeLeaseNamespace,
		}}
	} else {
		lease = base.DeepCopy()
	}
	holderIdentity := s.Config.AgentID
	leaseDuration := nodeLeaseDurationSeconds
	lease.Spec.HolderIdentity = &holderIdentity
	lease.Spec.LeaseDurationSeconds = &leaseDuration
	lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
	return lease
}

func (s *Server) reportNodeStatus(stopCh <-chan struct{}) {
	frequency := s.Config.NodeStatusUpdateFrequency
	if frequency <= 0 {
		frequency = time.Minute
	}
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()
	for {
		if err := s.updateNodeStatus(); err != nil {
			logger.Errorf("update Node status: %v", err)
		}
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) updateNodeStatus() error {
	ctx, cancel := context.WithTimeout(context.Background(), nodeStatusUpdateTimeout)
	defer cancel()
	node, err := s.client.CoreV1().Nodes().Get(ctx, s.Config.AgentID, metav1.GetOptions{})
	if err != nil {
		return err
	}
	observed := s.initialNode().Status
	now := metav1.Now()
	transition := now
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			transition = condition.LastTransitionTime
			break
		}
	}
	observed.Conditions = []corev1.NodeCondition{{
		Type: corev1.NodeReady, Status: corev1.ConditionTrue,
		LastHeartbeatTime: now, LastTransitionTime: transition, Reason: "AgentReady",
	}}
	node.Status = observed
	_, err = s.client.CoreV1().Nodes().UpdateStatus(ctx, node, &metav1.UpdateOptions{})
	return err
}
