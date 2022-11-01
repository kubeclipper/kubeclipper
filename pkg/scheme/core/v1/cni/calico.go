package cni

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

func init() {
	Register(&CalicoRunnable{})
	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat,
		cniInfo+"-calico", version, component.TypeTemplate), &CalicoRunnable{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat,
		cniInfo+"-calico", version, component.TypeStep), &CalicoRunnable{}); err != nil {
		panic(err)
	}
}

type CalicoRunnable struct {
	BaseCni
}

func (runnable *CalicoRunnable) Type() string {
	return "calico"
}

func (runnable *CalicoRunnable) Create() Stepper {
	return &CalicoRunnable{}
}

func (runnable *CalicoRunnable) NewInstance() component.ObjectMeta {
	return &CalicoRunnable{}
}

func (runnable *CalicoRunnable) InitStep(metadata *component.ExtraMetadata, cni *v1.CNI, networking *v1.Networking) Stepper {
	stepper := &CalicoRunnable{}
	ipv6 := ""
	if networking.IPFamily == v1.IPFamilyDualStack {
		ipv6 = networking.Pods.CIDRBlocks[1]
	}
	stepper.CNI = *cni
	stepper.LocalRegistry = cni.LocalRegistry
	stepper.BaseCni.Type = "calico"
	stepper.Version = cni.Version
	stepper.CriType = metadata.CRI
	stepper.Offline = cni.Offline
	stepper.Namespace = cni.Namespace
	stepper.DualStack = networking.IPFamily == v1.IPFamilyDualStack
	stepper.PodIPv4CIDR = networking.Pods.CIDRBlocks[0]
	stepper.PodIPv4CIDR = ipv6

	return stepper
}

func (runnable *CalicoRunnable) LoadImage(nodes []v1.StepNode) ([]v1.Step, error) {
	var steps []v1.Step
	bytes, err := json.Marshal(runnable)
	if err != nil {
		return nil, err
	}

	if runnable.Offline && runnable.LocalRegistry == "" {
		return []v1.Step{LoadImage("calico", bytes, nodes)}, nil
	}

	return steps, nil
}

func (runnable *CalicoRunnable) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	var steps []v1.Step
	bytes, err := json.Marshal(runnable)
	if err != nil {
		return nil, err
	}

	steps = append(steps, RenderYaml("calico", bytes, nodes))
	steps = append(steps, ApplyYaml(filepath.Join(manifestDir, "calico.yaml"), nodes))

	return steps, nil
}

func (runnable *CalicoRunnable) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(runnable)
	if err != nil {
		return nil, err
	}
	if runnable.Offline && runnable.LocalRegistry == "" {
		return []v1.Step{RemoveImage("calico", bytes, nodes)}, nil
	}
	return nil, nil
}

func (runnable *CalicoRunnable) Render(ctx context.Context, opts component.Options) error {
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}
	manifestFile := filepath.Join(manifestDir, "calico.yaml")
	return fileutil.WriteFileWithContext(ctx, manifestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		runnable.renderCalicoTo, opts.DryRun)
}

func (runnable *CalicoRunnable) renderCalicoTo(w io.Writer) error {
	at := tmplutil.New()
	calicoTemp, err := runnable.CalicoTemplate()
	if err != nil {
		return err
	}
	if _, err := at.RenderTo(w, calicoTemp, runnable); err != nil {
		return err
	}
	return nil
}

func (runnable *CalicoRunnable) CalicoTemplate() (string, error) {
	switch runnable.Version {
	case "v3.11.2":
		return calicoV3112, nil
	case "v3.16.10":
		return calicoV31610, nil
	case "v3.21.2":
		return calicoV3212, nil
	case "v3.22.4":
		return calicoV3224, nil
	}
	return "", fmt.Errorf("calico dose not support version: %s", runnable.Version)
}
