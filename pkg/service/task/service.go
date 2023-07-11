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

package task

import (
	"context"
	"encoding/json"
	"fmt"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/nodestatus"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
)

var _ service.Interface = (*Service)(nil)

const (
	nodeStatusUpdateRetry = 5
)

type Service struct {
	mqClient          natsio.Interface
	NodeReportSubject string
	AgentID           string
	IPDetect          string
	NodeIPDetect      string
	Region            string
	AgentSubject      string
	RegisterNode      bool

	// lastStatusReportTime is the time when node status was last reported.
	// lastStatusReportTime time.Time

	// syncNodeStatusMux is a lock on updating the node status, because this path is not thread-safe.
	// This lock is used by Kubelet.syncNodeStatus function and shouldn't be used anywhere else.
	syncNodeStatusMux sync.Mutex

	// nodeStatusUpdateFrequency specifies how often kubelet computes node status. If node lease
	// feature is not enabled, it is also the frequency that kubelet posts node status to master.
	// In that case, be cautious when changing the constant, it must work with nodeMonitorGracePeriod
	// in nodecontroller. There are several constraints:
	// 1. nodeMonitorGracePeriod must be N times more than nodeStatusUpdateFrequency, where
	//    N means number of retries allowed for kubelet to post node status. It is pointless
	//    to make nodeMonitorGracePeriod be less than nodeStatusUpdateFrequency, since there
	//    will only be fresh values from Kubelet at an interval of nodeStatusUpdateFrequency.
	//    The constant must be less than podEvictionTimeout.
	// 2. nodeStatusUpdateFrequency needs to be large enough for kubelet to generate node
	//    status. Kubelet may fail to update node status reliably if the value is too small,
	//    as it takes time to gather all necessary node information.
	NodeStatusUpdateFrequency time.Duration
	registrationCompleted     bool

	// clock is an interface that provides time related functionality in a way that makes it
	// easy to test the code.
	clock clock.Clock

	// handlers called during the tryUpdateNodeStatus cycle
	setNodeStatusFuncs []func(*v1.Node) error

	// node lease
	leaseDurationSeconds       int32
	leaseRenewInterval         time.Duration
	onRepeatedHeartbeatFailure func()
	// latestLease is the latest lease which the controller updated or created
	latestLease *coordinationv1.Lease
	oplog       component.OperationLogFile
	backupStore bs.BackupStore
	repoMirror  string
}

type ServiceOption func(*Service)

func WithNodeStatusUpdateFrequency(frequency time.Duration) ServiceOption {
	return func(s *Service) {
		s.NodeStatusUpdateFrequency = frequency
	}
}

func WithOplog(ol component.OperationLogFile) ServiceOption {
	return func(s *Service) {
		s.oplog = ol
	}
}

func WithBackupStore(backupStore bs.BackupStore) ServiceOption {
	return func(s *Service) {
		s.backupStore = backupStore
	}
}

func WithRepoMirror(mirror string) ServiceOption {
	return func(s *Service) {
		s.repoMirror = mirror
	}
}

func WithLeaseDurationSeconds(seconds int32) ServiceOption {
	return func(s *Service) {
		s.leaseDurationSeconds = seconds
	}
}

func defaultMQReconnectHandler(conn *nats.Conn) {
	logger.Debug("message queue reconnecting...", zap.Uint64("reconnect", conn.Reconnects))
}

func defaultMQDisconnectHandler(conn *nats.Conn, err error) {
	logger.Error("message queue disconnect with error", zap.Error(err))
}

func defaultMQErrorHandler(conn *nats.Conn, subscription *nats.Subscription, err error) {
	logger.Error("message queue handler error", zap.Error(err),
		zap.String("subj", subscription.Subject), zap.String("queue_group", subscription.Queue))
}

func defaultMQClosedHandler(conn *nats.Conn) {
	logger.Debug("message queue closed ...")
}

func defaultRepeatedHeartbeatFailure() {
	logger.Debug("repeated heartbeat failure ...")
}

func NewService(agentID, region, ipDetectMethod, nodeIPDetectMethod string, registerNode bool, natOpts *natsio.NatsOptions, opts ...ServiceOption) *Service {
	nc := natsio.NewNats(natOpts)
	nc.SetReconnectHandler(defaultMQReconnectHandler)
	nc.SetDisconnectErrHandler(defaultMQDisconnectHandler)
	nc.SetErrorHandler(defaultMQErrorHandler)
	nc.SetClosedHandler(defaultMQClosedHandler)
	s := &Service{
		mqClient:                   nc,
		NodeReportSubject:          natOpts.Client.NodeReportSubject,
		AgentID:                    agentID,
		IPDetect:                   ipDetectMethod,
		NodeIPDetect:               nodeIPDetectMethod,
		Region:                     region,
		AgentSubject:               fmt.Sprintf(service.MsgSubjectFormat, agentID, natOpts.Client.SubjectSuffix),
		RegisterNode:               registerNode,
		clock:                      clock.RealClock{},
		onRepeatedHeartbeatFailure: defaultRepeatedHeartbeatFailure,
	}
	for _, opt := range opts {
		opt(s)
	}

	s.leaseRenewInterval = time.Duration(float64(time.Duration(s.leaseDurationSeconds)*time.Second) * nodeLeaseRenewIntervalFraction)
	s.setNodeStatusFuncs = s.defaultNodeStatusFuncs()

	return s
}

func (s *Service) Run(stopCh <-chan struct{}) error {
	logger.Debug("mq client subscribe", zap.String("subject", s.AgentSubject))
	if err := s.mqClient.Subscribe(s.AgentSubject, s.msgHandler); err != nil {
		return err
	}
	go wait.Until(s.syncNodeStatus, s.NodeStatusUpdateFrequency, stopCh)
	go s.fastStatusUpdateOnce()
	// start syncing lease
	// TODO: disable node lease provisional
	go wait.Until(s.syncNodeLease, s.leaseRenewInterval, stopCh)
	return nil
}

func (s *Service) PrepareRun(stopCh <-chan struct{}) error {
	return s.mqClient.InitConn(stopCh)
}

func (s *Service) Close() {
	s.mqClient.Close()
}

func (s *Service) defaultNodeStatusFuncs() []func(*v1.Node) error {
	var setters []func(n *v1.Node) error
	setters = append(setters,
		nodestatus.Metadata(),
		nodestatus.NodeAddress(s.IPDetect, s.NodeIPDetect),
		nodestatus.MachineInfo(),
		nodestatus.ReadyCondition(s.clock.Now, TODO, TODO, TODO))

	return setters
}

func TODO() error {
	return nil
}

func (s *Service) fastStatusUpdateOnce() {
	s.syncNodeStatus()
}

func (s *Service) updateNodeStatus() error {
	logger.Debugf("Updating node status")
	for i := 0; i < nodeStatusUpdateRetry; i++ {
		if err := s.tryUpdateNodeStatus(i); err != nil {
			// if i > 0 && s.onRepeatedHeartbeatFailure != nil {
			//	s.onRepeatedHeartbeatFailure()
			// }
			logger.Error("Error updating node status, will retry", zap.Error(err))
		} else {
			return nil
		}
	}
	return fmt.Errorf("update node status exceeds retry count")
}

// syncNodeStatus should be called periodically from a goroutine.
// It synchronizes node status to master if there is any change or enough time
// passed from the last sync, registering the kubelet first if necessary.
func (s *Service) syncNodeStatus() {
	s.syncNodeStatusMux.Lock()
	defer s.syncNodeStatusMux.Unlock()

	if s.mqClient == nil {
		return
	}

	if s.RegisterNode {
		s.registerWithAPIServer()
	}

	if err := s.updateNodeStatus(); err != nil {
		logger.Error("Unable to update node status", zap.Error(err))
	}
}

func (s *Service) registerWithAPIServer() {
	if s.registrationCompleted {
		return
	}
	step := 100 * time.Millisecond
	for {
		time.Sleep(step)
		step = step * 2
		if step >= 7*time.Second {
			step = 7 * time.Second
		}

		node, err := s.initialNode(context.TODO())
		if err != nil {
			logger.Error("Unable to construct v1.Node object for kubeclipper-agent", zap.Error(err))
			continue
		}
		logger.Info("Attempting to register node", zap.String("node_id", node.Name))
		registered := s.tryRegisterWithAPIServer(node)
		if registered {
			logger.Info("Successfully registered node", zap.String("node_id", node.Name))
			s.registrationCompleted = true
			return
		}
	}
}

func (s *Service) initialNode(ctx context.Context) (*v1.Node, error) {
	node := &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "core.kubeclipper.io",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.AgentID,
			Labels: map[string]string{
				common.LabelOSStable:       goruntime.GOOS,
				common.LabelArchStable:     goruntime.GOARCH,
				common.LabelTopologyRegion: s.Region,
			},
		},
		ProxyIpv4CIDR: "",
		Status:        v1.NodeStatus{},
	}
	s.setNodeStatus(node)

	return node, nil
}

func (s *Service) setNodeStatus(node *v1.Node) {
	for i, f := range s.setNodeStatusFuncs {
		logger.Debug("Setting node status condition code",
			zap.Int("position", i),
			zap.String("node_id", node.Name))
		if err := f(node); err != nil {
			logger.Error("Failed to set some node status fields",
				zap.Int("position", i),
				zap.String("node_id", node.Name),
				zap.Error(err))
		}
	}
}

func (s *Service) tryRegisterWithAPIServer(node *v1.Node) bool {
	nodeBytes, err := json.Marshal(node)
	if err != nil {
		logger.Error("marshal node error", zap.Error(err))
		return false
	}
	payload := service.NodeStatusPayload{
		Op:   service.OperationRegisterNode,
		Data: nodeBytes,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error("marshal payload error", zap.Error(err))
		return false
	}

	msg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 1 * time.Second,
		Data:    payloadBytes,
	}
	msgResp, err := s.mqClient.Request(msg, nil)
	if err != nil {
		logger.Error("register node error", zap.Error(err))
		return false
	}
	resp := &service.CommonReply{}
	if err := json.Unmarshal(msgResp, resp); err != nil {
		logger.Error("unmarshal node status reply error", zap.Error(err))
		return false
	}
	// Error type is *StatusError
	if resp.Error != nil {
		logger.Error("register node error", zap.String("node_id", node.Name), zap.Error(resp.Error))
		return false
	}
	return true
}

func (s *Service) tryUpdateNodeStatus(tryNumber int) error {
	getNodePayload := &service.NodeStatusPayload{
		Op:   service.OperationGetNode,
		Data: []byte(s.AgentID),
	}
	getPayloadBytes, err := json.Marshal(getNodePayload)
	if err != nil {
		logger.Error("marshal payload error", zap.Int("try_number", tryNumber), zap.Error(err))
		return err
	}
	msg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 1 * time.Second,
		Data:    getPayloadBytes,
	}
	msgResp, err := s.mqClient.Request(msg, nil)
	if err != nil {
		logger.Error("get node error", zap.Error(err))
		return err
	}

	resp := &service.CommonReply{}
	if err := json.Unmarshal(msgResp, resp); err != nil {
		logger.Error("unmarshal get node reply error", zap.Error(err))
		return err
	}
	if resp.Error != nil {
		logger.Error("get node error", zap.String("node_id", s.AgentID), zap.Error(resp.Error))
		return resp.Error
	}

	originNode := &v1.Node{}
	if err := json.Unmarshal(resp.Data, originNode); err != nil {
		logger.Error("unmarshal node  error", zap.Error(err))
		return err
	}

	s.setNodeStatus(originNode)
	targetNodeBytes, err := json.Marshal(originNode)
	if err != nil {
		logger.Error("marshal target node error", zap.Error(err))
		return err
	}
	patch, err := jsonpatch.CreateMergePatch(resp.Data, targetNodeBytes)
	if err != nil {
		logger.Error("create node merge patch error", zap.Error(err))
		return err
	}

	logger.Debugf("node merge patch is %s", patch)

	patchNodePayload := service.NodeStatusPayload{
		Op:       service.OperationReportNodeStatus,
		NodeName: originNode.Name,
		Data:     patch,
	}
	patchNodePayloadBytes, err := json.Marshal(patchNodePayload)
	if err != nil {
		logger.Error("marshal payload error", zap.Int("try_number", tryNumber), zap.Error(err))
		return err
	}
	patchNodeMsg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 1 * time.Second,
		Data:    patchNodePayloadBytes,
	}
	return s.mqClient.Publish(patchNodeMsg)
}

func (s *Service) parseStepLogOperationID(identity string) (resp oplog.LogContentRequest, err error) {
	if err = json.Unmarshal([]byte(identity), &resp); err != nil {
		return
	}
	return
}
