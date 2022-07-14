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

package controller

import (
	"context"
	"sync"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"

	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	retrySleepTime         = 20 * time.Millisecond
	NodeHealthUpdateRetry  = 5
	nodeStartupGracePeriod = 1 * time.Minute
	nodeMonitorGracePeriod = 4 * time.Minute
	nodeMonitorPeriod      = 10 * time.Second
	namespaceNodeLease     = "node-lease"
)

type nodeHealthData struct {
	probeTimestamp           metav1.Time
	readyTransitionTimestamp metav1.Time
	status                   *v1.NodeStatus
	lease                    *coordinationv1.Lease
}

func (n *nodeHealthData) deepCopy() *nodeHealthData {
	if n == nil {
		return nil
	}
	return &nodeHealthData{
		probeTimestamp:           n.probeTimestamp,
		readyTransitionTimestamp: n.readyTransitionTimestamp,
		status:                   n.status.DeepCopy(),
		lease:                    n.lease.DeepCopy(),
	}
}

type nodeHealthMap struct {
	lock        sync.RWMutex
	nodeHealths map[string]*nodeHealthData
}

func newNodeHealthMap() *nodeHealthMap {
	return &nodeHealthMap{
		nodeHealths: make(map[string]*nodeHealthData),
	}
}

// getDeepCopy - returns copy of node health data.
// It prevents data being changed after retrieving it from the map.
func (n *nodeHealthMap) getDeepCopy(name string) *nodeHealthData {
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.nodeHealths[name].deepCopy()
}

func (n *nodeHealthMap) set(name string, data *nodeHealthData) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.nodeHealths[name] = data
}

type NodeStatusMon struct {
	// per Node map storing last observed health together with a local time when it was observed.
	nodeHealthMap *nodeHealthMap

	NodeLister  listerv1.NodeLister
	LeaseLister listerv1.LeaseLister
	// This timestamp is to be used instead of LastProbeTime stored in Condition. We do this
	// to avoid the problem with time skew across the cluster.
	now func() metav1.Time
	log logger.Logging

	NodeWriter cluster.NodeWriter
}

func (s *NodeStatusMon) mon() {
	nodes, err := s.NodeLister.List(labels.Everything())
	if err != nil {
		s.log.Warnf("list node error", zap.Error(err))
		return
	}
	for i := range nodes {
		// var gracePeriod time.Duration
		// var observedReadyCondition v1.NodeCondition
		// var currentReadyCondition *v1.NodeCondition
		node := nodes[i].DeepCopy()
		if err := wait.PollImmediate(retrySleepTime, retrySleepTime*NodeHealthUpdateRetry, func() (bool, error) {
			_, _, _, err = s.tryUpdateNodeHealth(context.TODO(), node)
			if err == nil {
				return true, nil
			}
			return false, err
		}); err != nil {
			s.log.Warnf("Update health of Node '%v' from Controller error: %v. "+
				"Skipping.", node.Name, err)
			continue
		}
	}
}

func (s *NodeStatusMon) SetupWithManager(mgr manager.Manager) {
	s.nodeHealthMap = newNodeHealthMap()
	s.now = metav1.Now
	s.log = mgr.GetLogger().WithName("node-status-monitor")
	mgr.AddWorkerLoop(s.mon, nodeMonitorPeriod)
}

func (s *NodeStatusMon) tryUpdateNodeHealth(ctx context.Context, node *v1.Node) (time.Duration, v1.NodeCondition, *v1.NodeCondition, error) {
	nodeHealth := s.nodeHealthMap.getDeepCopy(node.Name)
	defer func() {
		s.nodeHealthMap.set(node.Name, nodeHealth)
	}()

	var gracePeriod time.Duration
	var observedReadyCondition v1.NodeCondition
	_, currentReadyCondition := GetNodeCondition(&node.Status, v1.NodeReady)
	if currentReadyCondition == nil {
		// If ready condition is nil, then kc-agent never posted node status.
		// A fake ready condition is created, where LastHeartbeatTime and LastTransitionTime is set
		// to node.CreationTimestamp to avoid handle the corner case.
		observedReadyCondition = v1.NodeCondition{
			Type:               v1.NodeReady,
			Status:             v1.ConditionUnknown,
			LastHeartbeatTime:  node.CreationTimestamp,
			LastTransitionTime: node.CreationTimestamp,
		}
		gracePeriod = nodeStartupGracePeriod
		if nodeHealth != nil {
			nodeHealth.status = &node.Status
		} else {
			nodeHealth = &nodeHealthData{
				status:                   &node.Status,
				probeTimestamp:           node.CreationTimestamp,
				readyTransitionTimestamp: node.CreationTimestamp,
			}
		}
	} else {
		// If ready condition is not nil, make a copy of it, since we may modify it in place later.
		observedReadyCondition = *currentReadyCondition
		gracePeriod = nodeMonitorGracePeriod
	}

	var savedCondition *v1.NodeCondition
	var savedLease *coordinationv1.Lease
	if nodeHealth != nil {
		_, savedCondition = GetNodeCondition(nodeHealth.status, v1.NodeReady)
		savedLease = nodeHealth.lease
	}
	if nodeHealth == nil {
		s.log.Debugf("Missing timestamp for Node %s. Assuming now as a timestamp.", node.Name)
		nodeHealth = &nodeHealthData{
			status:                   &node.Status,
			probeTimestamp:           s.now(),
			readyTransitionTimestamp: s.now(),
		}
	} else if savedCondition == nil && currentReadyCondition != nil {
		s.log.Errorf("ReadyCondition was removed from Status of Node %s", node.Name)
		nodeHealth = &nodeHealthData{
			status:                   &node.Status,
			probeTimestamp:           s.now(),
			readyTransitionTimestamp: s.now(),
		}
	} else if savedCondition != nil && currentReadyCondition != nil && savedCondition.LastHeartbeatTime != currentReadyCondition.LastHeartbeatTime {
		var transitionTime metav1.Time
		if savedCondition.LastTransitionTime != currentReadyCondition.LastTransitionTime {
			s.log.Debugf("ReadyCondition for Node %s transitioned from %v to %v", node.Name, savedCondition, currentReadyCondition)
			transitionTime = s.now()
		} else {
			transitionTime = nodeHealth.readyTransitionTimestamp
		}
		s.log.Debugf("Node %s ReadyCondition updated. Updating timestamp: %+v vs %+v.", node.Name, nodeHealth.status, node.Status)
		nodeHealth = &nodeHealthData{
			status:                   &node.Status,
			probeTimestamp:           s.now(),
			readyTransitionTimestamp: transitionTime,
		}
	}

	// Always update the probe time if node lease is renewed.
	// Note: If kc agent never posted the node status, but continues renewing the
	// heartbeat leases, the node controller will assume the node is healthy and
	// take no action.
	observedLease, _ := s.LeaseLister.Leases(namespaceNodeLease).Get(node.Name)
	if observedLease != nil && (savedLease == nil || savedLease.Spec.RenewTime.Before(observedLease.Spec.RenewTime)) {
		nodeHealth.lease = observedLease
		nodeHealth.probeTimestamp = s.now()
	}

	if s.now().After(nodeHealth.probeTimestamp.Add(gracePeriod)) {
		// NodeReady condition or lease was last set longer ago than gracePeriod, so
		// update it to Unknown (regardless of its current value) in the master.
		nodeConditionTypes := []v1.NodeConditionType{
			v1.NodeReady,
		}
		nowTimestamp := s.now()
		for _, nodeConditionType := range nodeConditionTypes {
			_, currentCondition := GetNodeCondition(&node.Status, nodeConditionType)
			if currentCondition == nil {
				s.log.Debugf("Condition %v of node %v was never updated by kc agent", nodeConditionType, node.Name)
				node.Status.Conditions = append(node.Status.Conditions, v1.NodeCondition{
					Type:               nodeConditionType,
					Status:             v1.ConditionUnknown,
					Reason:             "NodeStatusNeverUpdated",
					Message:            "KcAgent never posted node status.",
					LastHeartbeatTime:  node.CreationTimestamp,
					LastTransitionTime: nowTimestamp,
				})
			} else {
				s.log.Debugf("node %v hasn't been updated for %+v. Last %v is: %+v",
					node.Name, s.now().Time.Sub(nodeHealth.probeTimestamp.Time), nodeConditionType, currentCondition)
				if currentCondition.Status != v1.ConditionUnknown {
					currentCondition.Status = v1.ConditionUnknown
					currentCondition.Reason = "NodeStatusUnknown"
					currentCondition.Message = "KcAgent stopped posting node status."
					currentCondition.LastTransitionTime = nowTimestamp
				}
			}
		}
		// We need to update currentReadyCondition due to its value potentially changed.
		_, currentReadyCondition = GetNodeCondition(&node.Status, v1.NodeReady)

		if !apiequality.Semantic.DeepEqual(currentReadyCondition, &observedReadyCondition) {
			if _, err := s.NodeWriter.UpdateNode(ctx, node); err != nil {
				s.log.Debugf("Error updating node %s: %v", node.Name, err)
				return gracePeriod, observedReadyCondition, currentReadyCondition, err
			}
			nodeHealth = &nodeHealthData{
				status:                   &node.Status,
				probeTimestamp:           nodeHealth.probeTimestamp,
				readyTransitionTimestamp: s.now(),
				lease:                    observedLease,
			}
			return gracePeriod, observedReadyCondition, currentReadyCondition, nil
		}
	}

	return gracePeriod, observedReadyCondition, currentReadyCondition, nil
}

func GetNodeCondition(status *v1.NodeStatus, conditionType v1.NodeConditionType) (int, *v1.NodeCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}
