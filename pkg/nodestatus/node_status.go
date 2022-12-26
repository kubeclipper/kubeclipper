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

package nodestatus

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sysutil"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

// Setter modifies the node in-place, and returns an error if the modification failed.
// Setters may partially mutate the node before returning an error.
type Setter func(node *v1.Node) error

func NodeAddress(ipDetectMethod, nodeIPDetectMethod string) Setter {
	log.Printf("ip detect method: %s, node ip detect methd: %s \n", ipDetectMethod, nodeIPDetectMethod)
	return func(node *v1.Node) error {
		var nodeAddress []v1.NodeAddress
		addresses, err := net.InterfaceAddrs()
		if err != nil {
			return err
		}
		for _, address := range addresses {
			isIpv4 := !strings.Contains(address.String(), ":")
			if isIpv4 {
				nodeAddress = append(nodeAddress, v1.NodeAddress{Type: v1.NodeIPV4Addr, Address: address.String()})
			} else {
				nodeAddress = append(nodeAddress, v1.NodeAddress{Type: v1.NodeIPV6Addr, Address: address.String()})
			}
		}
		node.Status.Addresses = nodeAddress
		ip, err := netutil.GetDefaultIP(true, ipDetectMethod)
		if err != nil {
			return err
		}
		nodeIP, err := netutil.GetDefaultIP(true, nodeIPDetectMethod)
		if err != nil {
			return err
		}
		gw, err := netutil.GetDefaultGateway(true)
		if err != nil {
			return err
		}
		node.Status.Ipv4DefaultIP = ip.To4().String()
		node.Status.Ipv4DefaultGw = gw.To4().String()
		node.Status.NodeIpv4DefaultIP = nodeIP.To4().String()
		return nil
	}
}

func Metadata() Setter {
	return func(node *v1.Node) error {
		conf, err := config.TryLoadFromDisk()
		if err != nil {
			logger.Error("Error getting metadata", zap.Error(err))
			return err
		}

		node.Labels[common.LabelTopologyRegion] = conf.Metadata.Region

		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		if conf.Metadata.FloatIP != "" {
			node.Annotations[common.AnnotationMetadataFloatIP] = conf.Metadata.FloatIP
		} else {
			delete(node.Annotations, common.AnnotationMetadataFloatIP)
		}

		if conf.Metadata.ProxyServer != "" {
			node.Annotations[common.AnnotationMetadataProxyServer] = conf.Metadata.ProxyServer
		} else {
			delete(node.Annotations, common.AnnotationMetadataProxyServer)
		}

		if conf.Metadata.ProxyAPIServer != "" {
			node.Annotations[common.AnnotationMetadataProxyAPIServer] = conf.Metadata.ProxyAPIServer
		} else {
			delete(node.Annotations, common.AnnotationMetadataProxyAPIServer)
		}

		if conf.Metadata.ProxySSH != "" {
			node.Annotations[common.AnnotationMetadataProxySSH] = conf.Metadata.ProxySSH
		} else {
			delete(node.Annotations, common.AnnotationMetadataProxySSH)
		}

		return nil
	}
}

func MachineInfo() Setter {
	return func(node *v1.Node) error {
		if node.Status.Capacity == nil {
			node.Status.Capacity = v1.ResourceList{}
		}
		info, err := sysutil.GetSysInfo()
		if err != nil {
			node.Status.Capacity[v1.ResourceCPU] = *resource.NewMilliQuantity(0, resource.DecimalSI)
			node.Status.Capacity[v1.ResourceMemory] = resource.MustParse("0Gi")
			logger.Error("Error getting machine info", zap.Error(err))
		} else {
			// set node info
			node.Labels[common.LabelHostname] = info.Host.Hostname
			node.Status.NodeInfo.Hostname = info.Host.Hostname
			node.Status.NodeInfo.HostID = info.Host.HostID
			node.Status.NodeInfo.OS = info.Host.OS
			node.Status.NodeInfo.Arch = runtime.GOARCH
			node.Status.NodeInfo.Platform = info.Host.Platform
			node.Status.NodeInfo.PlatformVersion = info.Host.PlatformVersion
			node.Status.NodeInfo.PlatformFamily = info.Host.PlatformFamily
			node.Status.NodeInfo.KernelArch = info.Host.KernelArch
			node.Status.NodeInfo.KernelVersion = info.Host.KernelVersion

			// set cpu memory size
			node.Status.Capacity[v1.ResourceCPU] = *resource.NewMilliQuantity(int64(info.CPU.Cores*1000), resource.DecimalSI)
			node.Status.Capacity[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dMi", info.Memory.Total))
			node.Status.Capacity[v1.ResourceStorage] = resource.MustParse(fmt.Sprintf("%dGi", info.Disk.Total))

			// set net info
			node.Status.Addresses = toNodeAddress(info.Net)
			// set volume info
			node.Status.VolumesAttached = attachedVolumes(info.Disk)
		}
		return nil
	}
}

// ReadyCondition returns a Setter that updates the v1.NodeReady condition on the node.
func ReadyCondition(
	nowFunc func() time.Time, // typically Kubelet.clock.Now
	runtimeErrorsFunc func() error, // typically Kubelet.runtimeState.runtimeErrors
	networkErrorsFunc func() error, // typically Kubelet.runtimeState.networkErrors
	storageErrorsFunc func() error, // typically Kubelet.runtimeState.storageErrors
) Setter {
	return func(node *v1.Node) error {
		// NOTE(aaronlevy): NodeReady condition needs to be the last in the list of node conditions.
		// This is due to an issue with version skewed kubelet and master components.
		// ref: https://github.com/kubernetes/kubernetes/issues/16961
		currentTime := metav1.NewTime(nowFunc())
		newNodeReadyCondition := v1.NodeCondition{
			Type:              v1.NodeReady,
			Status:            v1.ConditionTrue,
			Reason:            "KcAgentReady",
			Message:           "kc agent is posting ready status",
			LastHeartbeatTime: currentTime,
		}
		errs := []error{runtimeErrorsFunc(), networkErrorsFunc(), storageErrorsFunc()}
		requiredCapacities := []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory}
		// requiredCapacities = append(requiredCapacities, v1.ResourceEphemeralStorage)
		var missingCapacities []string
		for _, capacity := range requiredCapacities {
			if _, found := node.Status.Capacity[capacity]; !found {
				missingCapacities = append(missingCapacities, string(capacity))
			}
		}
		if len(missingCapacities) > 0 {
			errs = append(errs, fmt.Errorf("missing node capacity for resources: %s", strings.Join(missingCapacities, ", ")))
		}
		if aggregatedErr := errors.NewAggregate(errs); aggregatedErr != nil {
			newNodeReadyCondition = v1.NodeCondition{
				Type:              v1.NodeReady,
				Status:            v1.ConditionFalse,
				Reason:            "KcAgentNotReady",
				Message:           aggregatedErr.Error(),
				LastHeartbeatTime: currentTime,
			}
		}
		readyConditionUpdated := false
		for i := range node.Status.Conditions {
			if node.Status.Conditions[i].Type == v1.NodeReady {
				if node.Status.Conditions[i].Status == newNodeReadyCondition.Status {
					newNodeReadyCondition.LastTransitionTime = node.Status.Conditions[i].LastTransitionTime
				} else {
					newNodeReadyCondition.LastTransitionTime = currentTime
				}
				node.Status.Conditions[i] = newNodeReadyCondition
				readyConditionUpdated = true
				break
			}
		}
		if !readyConditionUpdated {
			newNodeReadyCondition.LastTransitionTime = currentTime
			node.Status.Conditions = append(node.Status.Conditions, newNodeReadyCondition)
		}
		return nil
	}
}

func attachedVolumes(d sysutil.Disk) []v1.AttachedVolume {
	m := make([]v1.AttachedVolume, len(d.DiskDevices))
	for i, item := range d.DiskDevices {
		m[i] = v1.AttachedVolume{
			Name:       v1.UniqueVolumeName(item.Device),
			DevicePath: item.Mountpoint,
		}
	}
	return m
}

func toNodeAddress(n []sysutil.Net) []v1.NodeAddress {
	addrs := make([]v1.NodeAddress, 0)
	for _, item := range n {
		for _, val := range item.Addrs {
			addrs = append(addrs, v1.NodeAddress{
				Address: val.Addr,
				Type:    v1.NodeAddressType(val.Family),
			})
		}
	}
	return addrs
}
