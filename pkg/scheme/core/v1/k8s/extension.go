package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, extension, extensionVersion, component.TypeStep), &Extension{}); err != nil {
		panic(err)
	}
}

var (
	_ component.StepRunnable = (*Extension)(nil)
)

const (
	k8sExtension     = "k8s-extension"
	extensionVersion = "v1"
	extension        = "extension"
)

// Extension is a step to install k8s extension,which include some useful tools
type Extension struct {
	Offline       bool   `json:"offline"`
	Version       string `json:"version"`
	CriType       string `json:"criType"`
	LocalRegistry string `json:"localRegistry"`
}

func (stepper *Extension) NewInstance() component.ObjectMeta {
	return &Extension{}
}

func (stepper *Extension) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, k8sExtension, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	// install configs.tar.gz
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	// local registry not filled and is in offline mode, download images.tar.gz file from tarballs
	if stepper.Offline && stepper.LocalRegistry == "" {
		imageSrc, err := instance.DownloadImages()
		if err != nil {
			return nil, err
		}
		if err = utils.LoadImage(ctx, opts.DryRun, imageSrc, stepper.CriType); err != nil {
			return nil, err
		}
		logger.Info("image tarball decompress successfully")
	}
	logger.Debug("k8s packages offline install successfully")
	return nil, nil
}

func (stepper *Extension) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	// remove related binary configuration files
	instance, err := downloader.NewInstance(ctx, k8sExtension, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err = instance.RemoveAll(); err != nil {
		logger.Error("remove k8s configs and images compressed files failed", zap.Error(err))
	}
	return nil, nil
}

func (stepper *Extension) InitStepper(c *v1.Cluster) *Extension {
	stepper.Offline = c.Offline()
	stepper.Version = extensionVersion
	stepper.CriType = c.ContainerRuntime.Type
	stepper.LocalRegistry = c.LocalRegistry
	return stepper
}

func (stepper *Extension) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(&stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "installExtension",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, extension, extensionVersion, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *Extension) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "unInstallExtension",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, extension, extensionVersion, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}
