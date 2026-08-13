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
	"os"
	"runtime"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	operationv2 "github.com/kubeclipper/kubeclipper/pkg/agent/operationv2"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

type Server struct {
	worker    *operationv2.Worker
	logServer *operationv2.LogServer
	client    clientset.Interface
	Config    *config.Config
}

func (s *Server) PrepareRun(stopCh <-chan struct{}) error {
	opLog, err := oplog.NewOperationLog(s.Config.OpLogOptions)
	if err != nil {
		return err
	}
	if s.Config.APIServer == nil {
		return fmt.Errorf("apiServer configuration is required")
	}
	restConfig := &rest.Config{
		Host: s.Config.APIServer.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: s.Config.APIServer.CAFile, CertFile: s.Config.APIServer.CertFile,
			KeyFile: s.Config.APIServer.KeyFile, ServerName: s.Config.APIServer.ServerName,
		},
		Timeout: 30 * time.Second,
		QPS:     20, Burst: 40,
	}
	client, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create kc-server client: %w", err)
	}
	s.client = client
	node, err := client.CoreV1().Nodes().Get(context.Background(), s.Config.AgentID, metav1.GetOptions{})
	if apierrors.IsNotFound(err) && s.Config.RegisterNode {
		node, err = client.CoreV1().Nodes().Create(context.Background(), s.initialNode(), metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			node, err = client.CoreV1().Nodes().Get(context.Background(), s.Config.AgentID, metav1.GetOptions{})
		}
	}
	if err != nil {
		return fmt.Errorf("get registered Node %q: %w", s.Config.AgentID, err)
	}
	registry := operationv2.NewRegistry()
	if err := registry.Register(operationv2.NoopExecutorName, operationv2.NoopExecutor{}); err != nil {
		return err
	}
	if err := registry.Register(operationv2.NodePreflightExecutorName, operationv2.NodePreflight{}); err != nil {
		return err
	}
	if err := registry.Register(operationv2.CommandStepExecutorName, operationv2.CommandStepExecutor{
		OpLog: opLog, RepoMirror: s.Config.ImageProxyOptions.KcImageRepoMirror,
	}); err != nil {
		return err
	}
	s.worker, err = operationv2.NewWorker(operationv2.WorkerOptions{
		AgentID: s.Config.AgentID, NodeUID: node.UID,
		Client: client.OperationsV1alpha1().OperationTasks(), Registry: registry, OpLog: opLog,
	})
	if err != nil {
		return err
	}
	if err := s.worker.PrepareRun(stopCh); err != nil {
		return err
	}
	s.logServer, err = operationv2.NewLogServer(operationv2.LogServerOptions{
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
	hostname, _ := os.Hostname()
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: corev1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: s.Config.AgentID, Labels: map[string]string{
			common.LabelOSStable: runtime.GOOS, common.LabelArchStable: runtime.GOARCH,
			common.LabelTopologyRegion: s.Config.Metadata.Region, common.LabelHostname: hostname,
		}},
		Status: corev1.NodeStatus{NodeInfo: corev1.NodeSystemInfo{Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH}},
	}
	if ip, err := netutil.GetDefaultIP(true, s.Config.IPDetect); err == nil {
		node.Status.Ipv4DefaultIP = ip.String()
		node.Status.NodeIpv4DefaultIP = ip.String()
		node.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: ip.String()}}
	}
	if gateway, err := netutil.GetDefaultGateway(true); err == nil {
		node.Status.Ipv4DefaultGw = gateway.String()
	}
	return node
}

func (s *Server) Run(stopCh <-chan struct{}) error {
	if err := s.worker.Run(stopCh); err != nil {
		return err
	}
	if err := s.logServer.Run(stopCh); err != nil {
		return err
	}
	go s.reportNodeStatus(stopCh)
	<-stopCh
	logger.Debugf("get stopCh signal, exit...")
	s.logServer.Close()
	s.worker.Close()
	return nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	_, err = s.client.CoreV1().Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
	return err
}
