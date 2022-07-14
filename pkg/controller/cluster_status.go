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
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"

	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	clusterStatusMonitorPeriod = 3 * time.Minute
)

type ClusterStatusMon struct {
	ClusterWriter cluster.ClusterWriter
	ClusterLister listerv1.ClusterLister
	mgr           manager.Manager
	log           logger.Logging
}

func (s *ClusterStatusMon) SetupWithManager(mgr manager.Manager) {
	s.mgr = mgr
	s.log = mgr.GetLogger().WithName("cluster-status-monitor")
	mgr.AddWorkerLoop(s.monitorClusterStatus, clusterStatusMonitorPeriod)
}

func (s *ClusterStatusMon) monitorClusterStatus() {
	// use k8s feature
	// ref: https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster-services/#manually-constructing-apiserver-proxy-urls
	clusters, err := s.ClusterLister.List(labels.Everything())
	if err != nil {
		s.log.Error("list clusters failed, monitor cluster status next period", zap.Error(err))
		return
	}
	for _, clu := range clusters {
		cc, exist := s.mgr.GetClusterClientSet(clu.Name)
		if !exist {
			s.log.Debug("clientset not exist, clientset may have not been finished", zap.String("cluster", clu.Name))
			continue
		}
		clientset := cc.Kubernetes()
		content, err := clientset.Discovery().RESTClient().Get().AbsPath("/healthz").Timeout(3 * time.Second).DoRaw(context.TODO())
		if err != nil {
			s.log.Error("get k8s cluster healthz failed", zap.Error(err))
			s.updateClusterComponentStatus(clu.Name, "kubernetes", "kubernetes", v1.ComponentUnKnown)
			return
		}
		if "ok" == string(content) {
			s.updateClusterComponentStatus(clu.Name, "kubernetes", "kubernetes", v1.ComponentHealthy)
		} else {
			s.updateClusterComponentStatus(clu.Name, "kubernetes", "kubernetes", v1.ComponentUnhealthy)
		}
		for _, com := range clu.Kubeadm.Components {
			comp, ok := component.Load(fmt.Sprintf(component.RegisterFormat, com.Name, com.Version))
			if !ok {
				s.log.Warn("load component failed", zap.String("component", com.Name), zap.String("version", com.Version))
				continue
			}
			compMeta := comp.NewInstance()
			if err = json.Unmarshal(com.Config.Raw, compMeta); err != nil {
				s.log.Error("unmarshall component failed", zap.String("component", com.Name), zap.String("version", com.Version), zap.Error(err))
				continue
			}
			newComp, ok := compMeta.(component.Interface)
			if !ok {
				s.log.Error("component type assert failed", zap.String("component", com.Name), zap.String("version", com.Version))
				continue
			}
			if newComp.Supported() {
				path := fmt.Sprintf("/api/v1/namespaces/%s/services/%s/proxy/%s", newComp.Ns(), newComp.Svc(), newComp.RequestPath())
				s.log.Debug("prepare to check component healthy status", zap.String("comp", newComp.GetInstanceName()), zap.String("request_path", path))
				content, err = clientset.Discovery().RESTClient().Get().AbsPath(path).DoRaw(context.TODO())
				s.log.Debug("check component healthy status", zap.ByteString("resp", content), zap.Error(err))
				if err != nil {
					s.updateClusterComponentStatus(clu.Name, newComp.GetComponentMeta(component.English).Category, newComp.GetInstanceName(), v1.ComponentUnhealthy)
				} else {
					s.updateClusterComponentStatus(clu.Name, newComp.GetComponentMeta(component.English).Category, newComp.GetInstanceName(), v1.ComponentHealthy)
				}
			} else {
				s.updateClusterComponentStatus(clu.Name, newComp.GetComponentMeta(component.English).Category, newComp.GetInstanceName(), v1.ComponentUnsupported)
			}
		}
	}
}
func (s *ClusterStatusMon) updateClusterComponentStatus(clusterName string, category, component string, statusType v1.ComponentStatus) {
	clu, err := s.ClusterLister.Get(clusterName)
	if err != nil {
		s.log.Warn("get cluster failed when update cluster component status, skip it", zap.String("cluster", clusterName), zap.String("component", component), zap.String("status", string(statusType)))
		return
	}
	index := getClusterComponentIndex(clu, component)
	if index != -1 && clu.Status.ComponentConditions[index].Status == statusType {
		s.log.Debug("cluster component status has no change", zap.String("cluster", clusterName), zap.String("component", component), zap.String("status", string(statusType)))
		return
	}

	if index == -1 {
		clu.Status.ComponentConditions = append(clu.Status.ComponentConditions, v1.ComponentConditions{
			Name:     component,
			Category: category,
			Status:   statusType,
		})
	} else {
		clu.Status.ComponentConditions[index].Status = statusType
	}
	if _, err = s.ClusterWriter.UpdateCluster(context.TODO(), clu); err != nil {
		s.log.Warn("update cluster component status failed", zap.String("cluster", clusterName), zap.String("component", component), zap.String("status", string(statusType)), zap.Error(err))
		return
	}
}

func getClusterComponentIndex(clu *v1.Cluster, component string) int {
	for i := range clu.Status.ComponentConditions {
		if clu.Status.ComponentConditions[i].Name == component {
			return i
		}
	}
	return -1
}
