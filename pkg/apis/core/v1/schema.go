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

package v1

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type PatchComponents struct {
	Uninstall bool           `json:"uninstall"`
	Addons    []corev1.Addon `json:"addons"`
}

// checkComponent check whether the component is installed in the current cluster
func (p *PatchComponents) checkComponents(cluster *corev1.Cluster) error {
	for _, v := range p.Addons {
		itf, ok := component.Load(fmt.Sprintf(component.RegisterFormat, v.Name, v.Version))
		if !ok {
			return fmt.Errorf("kubeclipper does not support %s-%s component", v.Name, v.Version)
		}
		meta := itf.GetComponentMeta(component.English)
		var installed bool
		for _, comp := range cluster.Addons {
			if comp.Name == v.Name {
				installed = true
			}
		}
		if p.Uninstall && !installed {
			return fmt.Errorf("%s-%s component is not installed in the current cluster", v.Name, v.Version)
		}
		// add component and component is unique
		if !p.Uninstall && installed && meta.Unique {
			return fmt.Errorf("%s-%s component has been installed in the current cluster", v.Name, v.Version)
		}
		// add component and component is not unique, check whether component StorageClassName is equal
		// TODO: Make validation logic specific to component configurations
		if !p.Uninstall && installed && !meta.Unique {
			var existedComps []corev1.Addon
			// components of the same type already exist in the collection cluster
			for _, comp := range cluster.Addons {
				if comp.Name == v.Name {
					existedComps = append(existedComps, comp)
				}
			}
			// resolves the config of current component to be operated
			currentCompMeta := itf.NewInstance()
			if err := json.Unmarshal(v.Config.Raw, currentCompMeta); err != nil {
				return fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
			}
			currentNewComp, _ := currentCompMeta.(component.Interface)
			for _, existedComp := range existedComps {
				// resolve that the cluster already has the same type of component config
				existCompItf, _ := component.Load(fmt.Sprintf(component.RegisterFormat, existedComp.Name, existedComp.Version))
				existCompMeta := existCompItf.NewInstance()
				if err := json.Unmarshal(existedComp.Config.Raw, existCompMeta); err != nil {
					return fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
				}
				existNewComp, _ := existCompMeta.(component.Interface)
				if currentNewComp.GetInstanceName() == existNewComp.GetInstanceName() {
					return fmt.Errorf("%s-%s component StorageClassName must be unique", v.Name, v.Version)
				}
			}
		}
	}
	return nil
}

// addOrRemoveComponentFromCluster update cluster components slice
func (p *PatchComponents) addOrRemoveComponentFromCluster(cluster *corev1.Cluster) (*corev1.Cluster, error) {
	if p.Uninstall {
		for _, v := range p.Addons {
			itf, ok := component.Load(fmt.Sprintf(component.RegisterFormat, v.Name, v.Version))
			if !ok {
				return nil, fmt.Errorf("kubeclipper does not support %s-%s component", v.Name, v.Version)
			}
			currentCompMeta := itf.NewInstance()
			if err := json.Unmarshal(v.Config.Raw, currentCompMeta); err != nil {
				return nil, fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
			}
			currentNewComp, _ := currentCompMeta.(component.Interface)
			for k, comp := range cluster.Addons {
				existCompItf, _ := component.Load(fmt.Sprintf(component.RegisterFormat, comp.Name, comp.Version))
				existCompMeta := existCompItf.NewInstance()
				if err := json.Unmarshal(comp.Config.Raw, existCompMeta); err != nil {
					return nil, fmt.Errorf("%s-%s component configuration resolution error: %s", v.Name, v.Version, err.Error())
				}
				existNewComp, _ := existCompMeta.(component.Interface)
				if currentNewComp.GetInstanceName() == existNewComp.GetInstanceName() {
					cluster.Addons = append(cluster.Addons[:k], cluster.Addons[k+1:]...)
				}
			}
		}
		return cluster, nil
	}
	cluster.Addons = append(cluster.Addons, p.Addons...)
	return cluster, nil
}

type StepLog struct {
	Content      string          `json:"content,omitempty"`
	Node         string          `json:"node,omitempty"`
	Timeout      metav1.Duration `json:"timeout,omitempty"`
	Status       string          `json:"status,omitempty"`
	DeliverySize int64           `json:"deliverySize,omitempty"`
	LogSize      int64           `json:"logSize,omitempty"`
}

type ClusterUpgrade struct {
	Version       string `json:"version"`
	Offline       bool   `json:"offline"`
	LocalRegistry string `json:"localRegistry"`
}
