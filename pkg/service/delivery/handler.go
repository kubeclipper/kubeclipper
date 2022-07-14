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

package delivery

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
)

func (s *Service) nodeStateReportInHandler(msg *nats.Msg) {
	payload := &service.NodeStatusPayload{}
	if err := json.Unmarshal(msg.Data, payload); err != nil {
		logger.Error("unmarshal node status report handler error", zap.Error(err))
		return
	}
	//logger.Debug("receive node status message", zap.Int32("operation",
	//	int32(payload.Op)), zap.ByteString("data", payload.Data))
	switch payload.Op {
	case service.OperationRegisterNode:
		resp := s.registerNodeOperation(msg, payload.Data)
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("marshal node status reply error", zap.Error(err))
			return
		}
		if err := msg.Respond(respBytes); err != nil {
			logger.Error("failed to reply message to notify server", zap.Error(err))
			return
		}
	case service.OperationReportNodeStatus:
		if err := s.updateNodeStatusOperation(msg, payload.NodeName, payload.Data); err != nil {
			logger.Error("failed to update node status", zap.Error(err))
			return
		}
	case service.OperationGetNode:
		resp := s.getNodeOperation(msg, string(payload.Data))
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("failed to marshal node status reply", zap.Error(err))
			return
		}
		if err := msg.Respond(respBytes); err != nil {
			logger.Error("failed to reply message to notify server", zap.Error(err))
			return
		}
	case service.OperationUpdateNodeLease:
		resp := s.UpdateNodeLeaseOperation(msg, payload.Data)
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("failed to marshal node status reply", zap.Error(err))
			return
		}
		if err := msg.Respond(respBytes); err != nil {
			logger.Error("failed to reply message to notify server", zap.Error(err))
			return
		}
	case service.OperationGetNodeLease:
		resp := s.getNodeLeaseOperation(msg, payload.NodeName, string(payload.Data))
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("failed to marshal node status reply", zap.Error(err))
			return
		}
		if err := msg.Respond(respBytes); err != nil {
			logger.Error("failed to reply message to notify server", zap.Error(err))
			return
		}
	case service.OperationCreateNodeLease:
		resp := s.createNodeLeaseOperation(msg, payload.Data)
		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("failed to marshal node status reply", zap.Error(err))
			return
		}
		if err := msg.Respond(respBytes); err != nil {
			logger.Error("failed to reply message to notify server", zap.Error(err))
			return
		}
	}
}

func (s *Service) registerNodeOperation(msg *nats.Msg, data []byte) *service.CommonReply {
	resp := &service.CommonReply{}
	node := &v1.Node{}
	if err := json.Unmarshal(data, node); err != nil {
		logger.Error("failed to unmarshal payload", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "unmarshal payload error with operation register node",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationRegisterNode),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Unmarshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	err := s.registerNode(node)
	if err != nil {
		logger.Error("register node error", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "register node error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationRegisterNode),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
	}
	return resp
}

func (s *Service) updateNodeStatusOperation(msg *nats.Msg, nodeName string, data []byte) error {
	node, err := s.getNode(nodeName)
	if err != nil {
		logger.Error("failed to get node", zap.Error(err))
		return err
	}
	nodeBytes, err := json.Marshal(node)
	if err != nil {
		return err
	}
	modifiedAlternative, err := jsonpatch.MergePatch(nodeBytes, data)
	if err != nil {
		logger.Error("failed to patch node", zap.Error(err))
		return err
	}
	logger.Debug("updated alternative node", zap.ByteString("alternative_node", modifiedAlternative))
	targetNode := &v1.Node{}
	if err := json.Unmarshal(modifiedAlternative, targetNode); err != nil {
		logger.Error("failed to unmarshal node", zap.Error(err))
		return err
	}
	// make resource version always be latest
	targetNode.ResourceVersion = node.ResourceVersion
	_, err = s.clusterOperator.UpdateNode(context.TODO(), targetNode)
	if err != nil {
		logger.Error("failed to update node", zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) getNodeOperation(msg *nats.Msg, nodeName string) *service.CommonReply {
	resp := &service.CommonReply{}
	node, err := s.getNode(nodeName)
	if err != nil {
		logger.Error("failed to get node", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "get node error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationGetNode),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	nodeBytes, err := json.Marshal(node)
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "marshal reply payload error with operation get node",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationGetNode),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Marshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	resp.Data = nodeBytes
	return resp
}

func (s *Service) registerNode(node *v1.Node) error {
	resp, err := s.clusterOperator.GetNode(context.TODO(), node.Name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if resp != nil {
		logger.Debug("node already register", zap.String("node", node.Name))
		return nil
	}
	_, err = s.clusterOperator.CreateNode(context.TODO(), node)
	if err != nil {
		logger.Error("create node error", zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) getNode(name string) (*v1.Node, error) {
	return s.clusterOperator.GetNode(context.TODO(), name)
}

func (s *Service) UpdateNodeLeaseOperation(msg *nats.Msg, data []byte) *service.CommonReply {
	resp := &service.CommonReply{}
	lease := &coordinationv1.Lease{}
	if err := json.Unmarshal(data, lease); err != nil {
		logger.Error("failed to unmarshal payload", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "unmarshal payload error with operation update node lease",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationUpdateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Unmarshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	lease, err := s.leaseOperator.GetLeaseWithNamespaceEx(context.TODO(), lease.Name, lease.Namespace, "0")
	if err != nil {
		logger.Error("failed to get node lease", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "update node lease error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationUpdateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		if apierrors.IsConflict(err) {
			resp.Error.Code = 409
		}
		return resp
	}
	leaseUpdated, err := s.leaseOperator.UpdateLease(context.TODO(), lease)
	if err != nil {
		logger.Error("failed to update node lease", zap.Error(err))
		resp.Error = &errors.StatusError{
			Message: "update node lease error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationUpdateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		if apierrors.IsConflict(err) {
			resp.Error.Code = 409
		}
		return resp
	}
	resp.Data, err = json.Marshal(leaseUpdated)
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "marshal payload error with operation update node lease",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationUpdateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Marshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
	}
	return resp
}

func (s *Service) getNodeLeaseOperation(msg *nats.Msg, name string, namespace string) *service.CommonReply {
	resp := &service.CommonReply{}
	lease, err := s.leaseOperator.GetLeaseWithNamespaceEx(context.TODO(), name, namespace, "0")
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "get node lease error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationGetNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		if apierrors.IsNotFound(err) {
			resp.Error.Code = 404
		}
		return resp
	}
	resp.Data, err = json.Marshal(lease)
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "marshal payload error with operation get node lease",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationGetNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Marshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
	}
	return resp
}

func (s *Service) createNodeLeaseOperation(msg *nats.Msg, data []byte) *service.CommonReply {
	resp := &service.CommonReply{}
	lease := &coordinationv1.Lease{}
	if err := json.Unmarshal(data, lease); err != nil {
		resp.Error = &errors.StatusError{
			Message: "unmarshal payload error with operation create node lease",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationCreateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Unmarshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	leaseCreated, err := s.leaseOperator.CreateLease(context.TODO(), lease)
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "create node lease error",
			Reason:  errors.StatusReasonStorageMethodCall,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationCreateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.StorageMethodCall,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
		return resp
	}
	resp.Data, err = json.Marshal(leaseCreated)
	if err != nil {
		resp.Error = &errors.StatusError{
			Message: "marshal payload error with operation create node lease",
			Reason:  errors.StatusReasonUnexpected,
			Details: &errors.StatusDetails{
				AgentID:   "",
				Subject:   msg.Subject,
				Operation: int32(service.OperationCreateNodeLease),
				Causes: []errors.StatusCause{
					{
						Type:    errors.Marshal,
						Message: err.Error(),
					},
				},
			},
			// TODO: Add code constants
			Code: 500,
		}
	}
	return resp
}
