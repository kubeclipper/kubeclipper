package cni

import (
	"fmt"
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
