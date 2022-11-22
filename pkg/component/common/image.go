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

const (
	imageName  = "image"
	AgentImage = "AgentImager"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, imageName, version, AgentImage), &Imager{}); err != nil {
		panic(err)
	}
}

type Imager struct {
	PkgName string `json:"pkgName"`
	Version string `json:"version"`
	Offline bool   `json:"offline"`
	CriName string `json:"criName"`
	// Optional. If the value of the change field is not empty, the DownloadCustomImages and RemoveCustomImages operations will be performed
	CustomImageList []string `json:"customConfig"`
}

func (i *Imager) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, i.PkgName, i.Version, runtime.GOARCH, !i.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}

	var dstFiles []string

	if len(i.CustomImageList) > 0 {
		dstFiles, err = instance.DownloadCustomImages(i.CustomImageList...)
		if err != nil {
			return nil, fmt.Errorf("download %s-%s custom image failed: %v", i.PkgName, i.Version, err)
		}
	} else {
		dstFile, err := instance.DownloadImages()
		if err != nil {
			return nil, fmt.Errorf("download %s-%s image failed: %v", i.PkgName, i.Version, err)
		}
		dstFiles = []string{dstFile}
	}

	// load image package
	for _, dstFile := range dstFiles {
		if err := utils.LoadImage(ctx, opts.DryRun, dstFile, i.CriName); err != nil {
			return nil, fmt.Errorf("load %s-%s image failed: %v", i.PkgName, i.Version, err)
		}
	}

	logger.Infof("%s-%s image packages offline install successfully", i.PkgName, i.Version)
	return nil, err
}

func (i *Imager) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, imageName, i.Version, runtime.GOARCH, !i.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}

	if len(i.CustomImageList) > 0 {
		if err = instance.RemoveCustomImages(i.CustomImageList...); err != nil {
			logger.Errorf("remove %s-%s custom-images compressed file failed", i.PkgName, i.Version, zap.Error(err))
		}
	} else {
		if err = instance.RemoveImages(); err != nil {
			logger.Errorf("remove %s-%s images compressed file failed", i.PkgName, i.Version, zap.Error(err))
		}
	}

	return nil, nil
}

func (i *Imager) NewInstance() component.ObjectMeta {
	return &Imager{}
}

func (i *Imager) InstallSteps(nodeList component.NodeList) ([]v1.Step, error) {
	customCommand, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       fmt.Sprintf("%s-imageLoad", i.PkgName),
			Timeout:    metav1.Duration{Duration: 30 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 0,
			Nodes:      utils.UnwrapNodeList(nodeList),
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, imageName, version, AgentImage),
					CustomCommand: customCommand,
				},
			},
		},
	}, nil
}
