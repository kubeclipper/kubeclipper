package cni

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CalicoNetworkIPIPAll IPIP-All mode
	CalicoNetworkIPIPAll = "Overlay-IPIP-All"
	// CalicoNetworkIPIPSubnet IPIP-Cross-Subnet mode
	CalicoNetworkIPIPSubnet = "Overlay-IPIP-Cross-Subnet"
	// CalicoNetworkVXLANAll Vxlan-All mode
	CalicoNetworkVXLANAll = "Overlay-Vxlan-All"
	// CalicoNetworkVXLANSubnet Vxlan-Cross-Subnet mode
	CalicoNetworkVXLANSubnet = "Overlay-Vxlan-Cross-Subnet"
	// CalicoNetworkBGP BGP mode
	CalicoNetworkBGP = "BGP"
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

type NodeAddressDetection struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type CalicoRunnable struct {
	BaseCni
	NodeAddressDetectionV4 NodeAddressDetection
	NodeAddressDetectionV6 NodeAddressDetection
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
	stepper.PodIPv6CIDR = ipv6
	stepper.NodeAddressDetectionV4 = ParseNodeAddressDetection(cni.Calico.IPv4AutoDetection)
	stepper.NodeAddressDetectionV6 = ParseNodeAddressDetection(cni.Calico.IPv6AutoDetection)

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

func (runnable *CalicoRunnable) InstallSteps(nodes []v1.StepNode, kubernetesVersion string) ([]v1.Step, error) {
	var steps []v1.Step
	bytes, err := json.Marshal(runnable)
	if err != nil {
		return nil, err
	}
	if IsHighKubeVersion(kubernetesVersion) {
		chart := &common.Chart{
			PkgName: "calico",
			Version: runnable.Version,
			Offline: runnable.Offline,
		}

		cLoadSteps, err := chart.InstallStepsV2(nodes)
		if err != nil {
			return nil, err
		}
		steps = append(steps, cLoadSteps...)
		steps = append(steps, RenderYaml("calico", bytes, nodes))
		steps = append(steps, InstallCalicoRelease(filepath.Join(downloader.BaseDstDir, "."+chart.PkgName, chart.Version, downloader.ChartFilename), filepath.Join(manifestDir, "calico.yaml"), nodes))
	} else {
		steps = append(steps, RenderYaml("calico", bytes, nodes))
		steps = append(steps, ApplyYaml(filepath.Join(manifestDir, "calico.yaml"), nodes))
	}

	return steps, nil
}

func (runnable *CalicoRunnable) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(runnable)
	if err != nil {
		return nil, err
	}
	var steps []v1.Step
	if runnable.Offline && runnable.LocalRegistry == "" {
		steps = append(steps, RemoveImage("calico", bytes, nodes))
	}

	if runnable.Calico != nil {
		steps = append(steps, runnable.clear(runnable.Calico, nodes)...)
	}

	return steps, nil
}

func (runnable *CalicoRunnable) clear(calico *v1.Calico, nodes []v1.StepNode) []v1.Step {
	if calico == nil {
		return nil
	}
	var steps []v1.Step

	switch calico.Mode {
	case CalicoNetworkIPIPAll, CalicoNetworkIPIPSubnet:
		steps = append(steps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "removeTunl",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"modprobe", "-r", "ipip"},
				},
			},
		})
	case CalicoNetworkVXLANAll, CalicoNetworkVXLANSubnet:
		steps = append(steps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "removeVtep",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"ip", "link", "delete", "vxlan.calico"},
				},
			},
		})
	}
	// clean all cali* interface
	steps = append(steps, v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "removeCali",
		Timeout:    metav1.Duration{Duration: 30 * time.Second},
		ErrIgnore:  true,
		Nodes:      nodes,
		Action:     v1.ActionUninstall,
		RetryTimes: 1,
		Commands: []v1.Command{
			{
				Type: v1.CommandShell,
				// ip addr | grep cali | awk '{cmd="ip link delete "$2;system(cmd)}'
				ShellCommand: []string{"ip", "addr", "|", "grep", "cali", "|", "awk", "'{cmd=\"ip link delete \"$2;system(cmd)}'"},
			},
		},
	})

	return steps
}

// CmdList cni kubectl cmd list
func (runnable *CalicoRunnable) CmdList(namespace string) map[string]string {
	cmdList := make(map[string]string)
	cmdList["get"] = fmt.Sprintf("kubectl get po -n %s | grep calico", namespace)
	cmdList["restart"] = fmt.Sprintf("kubectl rollout restart ds calico-node -n %s", namespace)

	return cmdList
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
	case "v3.24.5":
		return calicoV3245, nil
	case "v3.26.1":
		return calicoV3261, nil
	}
	return "", fmt.Errorf("calico dose not support version: %s", runnable.Version)
}
