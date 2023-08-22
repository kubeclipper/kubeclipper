package common

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

const DefaultHelmChartRepo = "kubeclipper"

const (
	chartName  = "chart"
	AgentChart = "AgentChart"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, chartName, version, AgentChart), &Chart{}); err != nil {
		panic(err)
	}
}

type Chart struct {
	PkgName string `json:"pkgName"`
	Version string `json:"version"`
	Offline bool   `json:"offline"`
}

func (i *Chart) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, i.PkgName, i.Version, runtime.GOARCH, !i.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}

	if _, err = instance.DownloadCharts(); err != nil {
		return nil, fmt.Errorf("download %s-%s chart packages failed: %v", i.PkgName, i.Version, err)
	}

	logger.Infof("%s-%s chart packages offline install successfully", i.PkgName, i.Version)
	return nil, err
}

func (i *Chart) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, i.PkgName, i.Version, runtime.GOARCH, !i.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}

	if err = instance.RemoveCharts(); err != nil {
		logger.Errorf("remove %s-%s chart file failed", i.PkgName, i.Version, zap.Error(err))
	}

	return nil, nil
}

func (i *Chart) NewInstance() component.ObjectMeta {
	return &Chart{}
}

func (i *Chart) InstallStepsV2(nodes []v1.StepNode) ([]v1.Step, error) {
	customCommand, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       fmt.Sprintf("%s-chartLoad", i.PkgName),
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, chartName, version, AgentChart),
					CustomCommand: customCommand,
				},
			},
		},
	}, nil
}

func (i *Chart) InstallSteps(nodeList component.NodeList) ([]v1.Step, error) {
	customCommand, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       fmt.Sprintf("%s-chartLoad", i.PkgName),
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      utils.UnwrapNodeList(nodeList),
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, chartName, version, AgentChart),
					CustomCommand: customCommand,
				},
			},
		},
	}, nil
}

func GetAddHelmRepoStep(nodes []v1.StepNode, repo string) v1.Step {
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "addHelmRepo",
		Timeout:    metav1.Duration{Duration: 10 * time.Second},
		ErrIgnore:  false,
		RetryTimes: 0,
		Nodes:      nodes,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"bin/sh", "-c", fmt.Sprintf("helm repo remove %s || true", DefaultHelmChartRepo)}, // forward action, ignore errors
			},
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/sh", "-c", fmt.Sprintf("helm repo add %s %s", DefaultHelmChartRepo, repo)},
			},
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"bin/sh", "-c", fmt.Sprintf("helm repo update %s", DefaultHelmChartRepo)},
			},
		},
	}
}
