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
	"encoding/json"
	"time"

	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
	"github.com/kubeclipper/kubeclipper/pkg/utils/pointer"
)

const (
	// maxUpdateRetries is the number of immediate, successive retries the controller will attempt
	// when renewing the lease before it waits for the renewal interval before trying again,
	// similar to what we do for node status retries
	maxUpdateRetries = 5
	// maxBackoff is the maximum sleep time during backoff (e.g. in backoffEnsureLease)
	maxBackoff                     = 7 * time.Second
	nodeLeaseRenewIntervalFraction = 0.25
	namespaceNodeLease             = "node-lease"
)

// ProcessLeaseFunc processes the given lease in-place
type ProcessLeaseFunc func(*coordinationv1.Lease) error

func (s *Service) syncNodeLease() {
	if s.latestLease != nil {
		err := s.tryUpdateNodeLease(s.latestLease)
		if err == nil {
			return
		}
		logger.Error("failed to update lease using latest lease, fallback to ensure lease", zap.Error(err))
	}
	lease, created := s.backoffEnsureNodeLease()
	s.latestLease = lease
	if !created && lease != nil {
		if err := s.tryUpdateNodeLease(lease); err != nil {
			logger.Error("failed to update lease", zap.Error(err), zap.Duration("retry_after", s.leaseRenewInterval))
		}
	}
}

func (s *Service) tryUpdateNodeLease(base *coordinationv1.Lease) error {
	for i := 0; i < maxUpdateRetries; i++ {
		leaseToUpdate, _ := s.newNodeLease(base)
		lease, err := s.updateNodeLease(leaseToUpdate)
		if err == nil {
			s.latestLease = lease
			return nil
		}
		// etcd OptimisticLockError requires getting the newer version of lease to proceed.
		if errors.IsConflict(err) {
			base, _ = s.backoffEnsureNodeLease()
			continue
		}
		if i > 0 && s.onRepeatedHeartbeatFailure != nil {
			s.onRepeatedHeartbeatFailure()
		}
	}
	return nil
}

func (s *Service) backoffEnsureNodeLease() (*coordinationv1.Lease, bool) {
	var (
		lease   *coordinationv1.Lease
		created bool
		err     error
	)
	sleep := 100 * time.Millisecond
	for {
		lease, created, err = s.ensureNodeLease()
		if err == nil {
			break
		}
		sleep = minDuration(2*sleep, maxBackoff)
		logger.Error("failed to ensure lease exist", zap.Error(err), zap.Duration("backoff", sleep))
		s.clock.Sleep(sleep)
	}
	return lease, created
}

func (s *Service) ensureNodeLease() (*coordinationv1.Lease, bool, error) {
	getLeasePayload := &service.NodeStatusPayload{
		Op:       service.OperationGetNodeLease,
		NodeName: s.AgentID,
		Data:     []byte(namespaceNodeLease),
	}
	getLeasePayloadBytes, err := json.Marshal(getLeasePayload)
	if err != nil {
		logger.Error("marshal payload error", zap.Error(err))
		return nil, false, err
	}
	msg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 3 * time.Second,
		Data:    getLeasePayloadBytes,
	}
	msgResp, err := s.mqClient.Request(msg, nil)
	if err != nil {
		logger.Error("get node lease error", zap.Error(err))
		return nil, false, err
	}
	resp := &service.CommonReply{}
	if err := json.Unmarshal(msgResp, resp); err != nil {
		logger.Error("unmarshal get node lease reply error", zap.Error(err))
		return nil, false, err
	}
	logger.Debug("get node lease", zap.Any("reply", resp))
	if resp.Error != nil {
		if errors.IsNotFound(resp.Error) {
			return s.createNodeLease()
		}
		logger.Error("get node lease error", zap.String("node_id", s.AgentID), zap.Error(resp.Error))
		return nil, false, resp.Error
	}
	logger.Debug("get node lease", zap.ByteString("data", resp.Data))
	lease := &coordinationv1.Lease{}
	if err := json.Unmarshal(resp.Data, lease); err != nil {
		logger.Error("unmarshal get node lease reply data error", zap.Error(err))
		return nil, false, err
	}
	// lease already existed
	return lease, false, nil
}

func (s *Service) createNodeLease() (*coordinationv1.Lease, bool, error) {
	leaseToCreate, _ := s.newNodeLease(nil)
	leaseToCreateBytes, err := json.Marshal(leaseToCreate)
	if err != nil {
		logger.Error("marshal payload error", zap.Error(err))
		return nil, false, err
	}
	createLeasePayload := &service.NodeStatusPayload{
		Op:       service.OperationCreateNodeLease,
		NodeName: s.AgentID,
		Data:     leaseToCreateBytes,
	}
	createLeasePayloadBytes, err := json.Marshal(createLeasePayload)
	if err != nil {
		logger.Error("marshal payload error", zap.Error(err))
		return nil, false, err
	}

	msg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 3 * time.Second,
		Data:    createLeasePayloadBytes,
	}
	msgResp, err := s.mqClient.Request(msg, nil)
	if err != nil {
		logger.Error("create node lease error", zap.Error(err))
		return nil, false, err
	}
	resp := &service.CommonReply{}
	if err := json.Unmarshal(msgResp, resp); err != nil {
		logger.Error("unmarshal create node lease reply error", zap.Error(err))
		return nil, false, err
	}
	if resp.Error != nil {
		return nil, false, resp.Error
	}
	lease := &coordinationv1.Lease{}
	if err := json.Unmarshal(resp.Data, lease); err != nil {
		logger.Error("unmarshal get node lease reply data error", zap.Error(err))
		return nil, false, err
	}
	return lease, true, nil
}

func (s *Service) updateNodeLease(lease *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	leaseToUpdateByte, err := json.Marshal(lease)
	if err != nil {
		logger.Error("marshal node lease error", zap.Error(err))
		return nil, err
	}
	updateNlPayload := &service.NodeStatusPayload{
		Op:   service.OperationUpdateNodeLease,
		Data: leaseToUpdateByte,
	}
	nlPayloadBytes, err := json.Marshal(updateNlPayload)
	if err != nil {
		logger.Error("marshal payload error", zap.Error(err))
		return nil, err
	}
	msg := &natsio.Msg{
		Subject: s.NodeReportSubject,
		From:    s.AgentID,
		To:      "",
		Step:    "",
		Timeout: 3 * time.Second,
		Data:    nlPayloadBytes,
	}
	msgResp, err := s.mqClient.Request(msg, nil)
	if err != nil {
		logger.Error("update node lease error", zap.Error(err))
		return nil, err
	}

	resp := &service.CommonReply{}
	if err := json.Unmarshal(msgResp, resp); err != nil {
		logger.Error("unmarshal update node lease reply error", zap.Error(err))
		return nil, err
	}
	if resp.Error != nil {
		logger.Error("update node lease error", zap.String("node_id", s.AgentID), zap.Error(resp.Error))
		return nil, resp.Error
	}

	l := &coordinationv1.Lease{}
	if err := json.Unmarshal(resp.Data, l); err != nil {
		logger.Error("unmarshal node lease error", zap.Error(err))
		return nil, err
	}
	return l, nil
}

// newNodeLease constructs a new lease if base is nil, or returns a copy of base
// with desired state asserted on the copy.
// Note that an error will block lease CREATE, causing the CREATE to be retried in
// the next iteration; but the error won't block lease refresh (UPDATE).
func (s *Service) newNodeLease(base *coordinationv1.Lease) (*coordinationv1.Lease, error) {
	var lease *coordinationv1.Lease
	if base == nil {
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.AgentID,
				Namespace: namespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       pointer.StringPtr(s.AgentID),
				LeaseDurationSeconds: pointer.Int32Ptr(s.leaseDurationSeconds),
			},
		}
	} else {
		lease = base.DeepCopy()
	}
	lease.Spec.RenewTime = &metav1.MicroTime{Time: s.clock.Now()}

	//if c.newLeasePostProcessFunc != nil {
	//	err := c.newLeasePostProcessFunc(lease)
	//	return lease, err
	//}
	return lease, nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
