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

func InstallCalicoRelease(chartPath string, yamlName string, nodes []v1.StepNode) v1.Step {
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
				ShellCommand: []string{"helm", "upgrade", "--install", "--create-namespace", "calico", "-n", "calico-system", chartPath, "-f", yamlName},
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
