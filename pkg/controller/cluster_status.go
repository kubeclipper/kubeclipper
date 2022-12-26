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
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage/kubeadm"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"github.com/kubeclipper/kubeclipper/pkg/clustermanage"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"

	"github.com/kubeclipper/kubeclipper/pkg/service"

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
	clusterStatusMonitorPeriod            = 3 * time.Minute
	updateClusterCertStatusRetryTimes     = 2
	updateClusterCertStatusRetrySleepTime = 3 * time.Second
)

var (
	exemptStatus = sets.NewString(string(v1.ClusterInstalling), string(v1.ClusterInstallFailed), string(v1.ClusterTerminating), string(v1.ClusterTerminateFailed))
)

type ClusterStatusMon struct {
	ClusterWriter       cluster.ClusterWriter
	ClusterLister       listerv1.ClusterLister
	NodeLister          listerv1.NodeLister
	CmdDelivery         service.CmdDelivery
	mgr                 manager.Manager
	log                 logger.Logging
	CloudProviderLister listerv1.CloudProviderLister
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
		if exemptStatus.Has(string(clu.Status.Phase)) {
			continue
		}
		err = s.updateClusterControlPlaneStatus(clu.DeepCopy())
		if err != nil {
			s.log.Error("update cluster control plane status failed", zap.Error(err))
		}
		cc, exist := s.mgr.GetClusterClientSet(clu.Name)
		if !exist {
			s.log.Debug("clientset not exist, clientset may have not been finished", zap.String("cluster", clu.Name))
			continue
		}
		// TODO: need refactor
		// use backoff retry utils instead
		for i := 0; i < updateClusterCertStatusRetryTimes; i++ {
			if err = s.updateClusterCertification(clu.Name); err == nil {
				break
			}
			time.Sleep(updateClusterCertStatusRetrySleepTime)
		}
		if err != nil {
			s.log.Error("update cluster certification failed", zap.Error(err))
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
		for _, com := range clu.Addons {
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

func (s *ClusterStatusMon) updateClusterCertification(clusterName string) error {
	clu, err := s.ClusterLister.Get(clusterName)
	if err != nil {
		s.log.Warn("get cluster failed when update cluster certification status, skip it", zap.String("cluster", clusterName))
		return err
	}
	var certifications []v1.Certification

	provider, ok := clu.Labels[common.LabelClusterProviderName]
	// get certifications from provider
	if ok && provider != kubeadm.ProviderKubeadm {
		certifications, err = s.getCertificationFromProvider(clu)
		if err != nil {
			return err
		}
	} else {
		// get certifications from kc
		certifications, err = s.GetCertificationFromKC(clu)
		if err != nil {
			return err
		}
	}

	clu.Status.Certifications = certifications
	if _, err = s.ClusterWriter.UpdateCluster(context.TODO(), clu); err != nil {
		s.log.Warn("update cluster certification status failed", zap.String("cluster", clu.Name), zap.Error(err))
	}
	return nil
}

func (s *ClusterStatusMon) getCertificationFromProvider(clu *v1.Cluster) ([]v1.Certification, error) {
	providerName := clu.Labels[common.LabelClusterProviderName]
	provider, err := s.CloudProviderLister.Get(providerName)
	if err != nil {
		return nil, err
	}
	cp, err := clustermanage.GetProvider(clustermanage.Operator{}, *provider)
	if err != nil {
		return nil, err
	}
	return cp.GetCertification(context.TODO(), clu.Name)
}

func (s *ClusterStatusMon) GetCertificationFromKC(clu *v1.Cluster) ([]v1.Certification, error) {
	var cmd []string
	if clu.KubernetesVersion[1:] < k8s.KubeCertsCluVersion {
		cmd = []string{"kubeadm", "alpha", "certs", "check-expiration"}
	} else {
		cmd = []string{"kubeadm", "certs", "check-expiration"}
	}
	res, err := s.CmdDelivery.DeliverCmd(context.TODO(), clu.Masters[0].ID, cmd, 3*time.Minute)
	if err != nil {
		s.log.Warn("get cluster failed when get cluster certification status, skip it", zap.String("cluster", clu.Name))
		return nil, err
	}
	splitRes := strings.Split(string(res), "\n\n")
	if len(splitRes) != 3 {
		logger.Errorf("read cluster certs error")
		return nil, err
	}
	crts := strings.Split(splitRes[1], "\n")
	cas := strings.Split(splitRes[2], "\n")
	certification := make([]v1.Certification, 0)
	for _, ca := range cas[1:4] {
		crt := strings.Fields(ca)
		expire, parErr := time.Parse("Jan 02, 2006 15:04 MST", strings.Join(crt[1:6], " "))
		if parErr != nil {
			s.log.Warn("get cluster failed when get cluster ca expiration time", zap.String("cluster", clu.Name))
			return nil, err
		}
		certification = append(certification, v1.Certification{
			Name:           crt[0],
			CAName:         "",
			ExpirationTime: metav1.Time{Time: expire},
		})
	}
	for _, cert := range crts[1:] {
		crt := strings.Fields(cert)
		expire, parErr := time.Parse("Jan 02, 2006 15:04 MST", strings.Join(crt[1:6], " "))
		if parErr != nil {
			s.log.Warn("get cluster failed when get cluster cert expiration time", zap.String("cluster", clu.Name))
			return nil, err
		}
		certification = append(certification, v1.Certification{
			Name:           crt[0],
			CAName:         crt[7],
			ExpirationTime: metav1.Time{Time: expire},
		})
	}
	return certification, nil
}

func getClusterComponentIndex(clu *v1.Cluster, component string) int {
	for i := range clu.Status.ComponentConditions {
		if clu.Status.ComponentConditions[i].Name == component {
			return i
		}
	}
	return -1
}

func (s *ClusterStatusMon) updateClusterControlPlaneStatus(clu *v1.Cluster) error {
	controlplanestatus := make([]v1.ControlPlaneHealth, len(clu.Masters))
	for index, id := range clu.Masters.GetNodeIDs() {
		node, err := s.NodeLister.Get(id)
		if err != nil {
			s.log.Error("get master node error", zap.String("node", id), zap.String("cluster", clu.Name))
			return err
		}
		proxyAPIServer := node.Annotations[common.AnnotationMetadataProxyAPIServer]
		floatIP := node.Annotations[common.AnnotationMetadataFloatIP]
		apiServer := node.Status.NodeIpv4DefaultIP + ":6443"
		if floatIP != "" {
			apiServer = floatIP + ":6443"
		}
		if proxyAPIServer != "" {
			apiServer = proxyAPIServer
		}
		health := v1.ControlPlaneHealth{
			ID:       id,
			Hostname: node.Labels[common.LabelHostname],
			Address:  apiServer,
		}
		resp, _, respErr := httputil.CommonRequest(fmt.Sprintf("https://%s/livez", apiServer), "GET", nil, nil, nil)
		if respErr != nil {
			health.Status = v1.ComponentUnKnown
		}
		if string(resp) == "ok" {
			health.Status = v1.ComponentHealthy
		} else {
			health.Status = v1.ComponentUnhealthy
		}
		controlplanestatus[index] = health
	}

	if isControlPlaneHealthChange(clu.Status.ControlPlaneHealth, controlplanestatus) {
		clu.Status.ControlPlaneHealth = controlplanestatus
		_, err := s.ClusterWriter.UpdateCluster(context.TODO(), clu)
		return err
	}

	s.log.Debug("control plane status has no change, skip update", zap.String("cluster", clu.Name))
	return nil
}

func isControlPlaneHealthChange(a, b []v1.ControlPlaneHealth) bool {
	if len(a) != len(b) {
		return true
	}

	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}

	return false
}
