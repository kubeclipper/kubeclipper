/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package cni

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func LoadImage(name string, custom []byte, nodes []v1.StepNode) v1.Step {
	return v1.Step{

		ID:         strutil.GetUUID(),
		Name:       "cniImageLoader",
		Timeout:    metav1.Duration{Duration: 5 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, cniInfo+"-"+name, version, component.TypeStep),
				CustomCommand: custom,
			},
		},
	}
}

func RenderYaml(name string, custom []byte, nodes []v1.StepNode) v1.Step {
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "renderCniYaml",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Commands: []v1.Command{
			{
				Type: v1.CommandTemplateRender,
				Template: &v1.TemplateCommand{
					Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, cniInfo+"-"+name, version, component.TypeTemplate),
					Data:     custom,
				},
			},
		},
	}
}

func ApplyYaml(yamlName string, nodes []v1.StepNode) v1.Step {
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "applyCniYaml",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"kubectl", "apply", "-f", yamlName},
			},
		},
	}
}

func RemoveImage(name string, custom []byte, nodes []v1.StepNode) v1.Step {
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "removeCniImage",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Action:     v1.ActionUninstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, cniInfo+"-"+name, version, component.TypeStep),
				CustomCommand: custom,
			},
		},
	}
}

func InstallCalicoRelease(chartPath string, yamlName string, calicoVersion string, nodes []v1.StepNode) v1.Step {
	// For Calico v3.29.6 and later, tigera-operator must be installed in tigera-operator namespace
	// For earlier versions, it can be installed in calico-system namespace
	namespace := "calico-system"
	if IsHighCalicoVersion(calicoVersion) {
		namespace = "tigera-operator"
	}

	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "installCalicoRelease",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"helm", "upgrade", "--install", "--create-namespace", "calico", "-n", namespace, chartPath, "-f", yamlName},
			},
		},
	}
}

func IsHighKubeVersion(kubeVersion string) bool {
	if kubeVersion == "" {
		return false
	}
	kubeVersion = strings.ReplaceAll(kubeVersion, "v", "")
	kubeVersion = strings.ReplaceAll(kubeVersion, ".", "")

	kubeVersion = strings.Join(strings.Split(kubeVersion, "")[0:3], "")

	if v, _ := strconv.Atoi(kubeVersion); v >= 126 {
		return true
	}
	return false
}

// IsHighCalicoVersion checks if the Calico version is v3.29.6 or later
// For v3.29.6+, tigera-operator must be installed in tigera-operator namespace
func IsHighCalicoVersion(calicoVersion string) bool {
	if calicoVersion == "" {
		return false
	}
	// Remove 'v' prefix if present
	calicoVersion = strings.TrimPrefix(calicoVersion, "v")

	// Split version into parts: major.minor.patch
	parts := strings.Split(calicoVersion, ".")
	if len(parts) < 3 {
		return false
	}

	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])

	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}

	// Check if version >= 3.29.6
	if major > 3 {
		return true
	}
	if major == 3 && minor > 29 {
		return true
	}
	if major == 3 && minor == 29 && patch >= 6 {
		return true
	}

	return false
}

// UseCalicoOperator checks if the Calico version should use tigera-operator (Helm) for deployment
// For Calico v3.26 and above, use Helm deployment with tigera-operator
// For versions below v3.26, use direct YAML deployment
func UseCalicoOperator(calicoVersion string) bool {
	if calicoVersion == "" {
		return false
	}
	// Remove 'v' prefix if present
	calicoVersion = strings.TrimPrefix(calicoVersion, "v")

	// Split version into parts: major.minor.patch
	parts := strings.Split(calicoVersion, ".")
	if len(parts) < 2 {
		return false
	}

	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])

	if err1 != nil || err2 != nil {
		return false
	}

	// Check if version >= 3.26
	if major > 3 {
		return true
	}
	if major == 3 && minor >= 26 {
		return true
	}

	return false
}

func ParseNodeAddressDetection(nodeAddressDetection string) NodeAddressDetection {
	// nodeAddressDetection first-found|can-reach=DESTINATION|interface=INTERFACE-REGEX|skip-interface=INTERFACE-REGEX
	detections := strings.Split(nodeAddressDetection, "=")
	if len(detections) == 0 {
		return NodeAddressDetection{
			Type: "first-found",
		}
	}
	switch detections[0] {
	case "first-found":
		return NodeAddressDetection{
			Type: "first-found",
		}
	case "can-reach":
		return NodeAddressDetection{
			Type:  "can-reach",
			Value: strings.TrimPrefix(nodeAddressDetection, "can-reach="),
		}
	case "interface":
		return NodeAddressDetection{
			Type:  "interface",
			Value: strings.TrimPrefix(nodeAddressDetection, "interface="),
		}
	case "skip-interface":
		return NodeAddressDetection{
			Type:  "skip-interface",
			Value: strings.TrimPrefix(nodeAddressDetection, "skip-interface="),
		}
	default:
		return NodeAddressDetection{
			Type: "first-found",
		}
	}
}
