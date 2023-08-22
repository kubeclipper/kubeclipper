package cni

import (
	"context"
	"errors"
	"runtime"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"go.uber.org/zap"
)

var cniFactories = make(map[string]CniFactory)

type CniFactory interface {
	Type() string
	Create() Stepper
}

func Register(factory CniFactory) {
	cniFactories[factory.Type()] = factory
}

func Load(cniType string) (CniFactory, error) {
	if _, ok := cniFactories[cniType]; !ok {
		return nil, errors.New("this cni is not supported at this time")
	}
	return cniFactories[cniType], nil
}

const (
	version     = "v1"
	cniInfo     = "cniInfo"
	manifestDir = "/tmp/.cni"
)

type BaseCni struct {
	v1.CNI
	DualStack   bool   `json:"dualStack"`
	PodIPv4CIDR string `json:"podIPv4CIDR"`
	PodIPv6CIDR string `json:"podIPv6CIDR"`
}

type Stepper interface {
	InitStep(metadata *component.ExtraMetadata, cni *v1.CNI, networking *v1.Networking) Stepper
	LoadImage(nodes []v1.StepNode) ([]v1.Step, error)
	InstallSteps(nodes []v1.StepNode, kubeVersion string) ([]v1.Step, error)
	UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error)
	CmdList(namespace string) map[string]string
}

func (runnable *BaseCni) NewInstance() component.ObjectMeta {
	return &BaseCni{}
}

func (runnable *BaseCni) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, runnable.Type, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}

	if runnable.Offline && runnable.LocalRegistry == "" {
		dstFile, err := instance.DownloadImages()
		if err != nil {
			return nil, err
		}
		// load image package
		if err = utils.LoadImage(ctx, opts.DryRun, dstFile, runnable.CriType); err != nil {
			return nil, err
		}
		logger.Info("calico packages offline install successfully")
	}

	return nil, nil
}

func (runnable *BaseCni) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, runnable.Type, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err = instance.RemoveImages(); err != nil {
		logger.Error("remove calico images compressed file failed", zap.Error(err))
	}
	return nil, nil
}

// RecoveryCNICmd get recovery cni cmd
func RecoveryCNICmd(metadata *component.ExtraMetadata) (cmdList map[string]string, err error) {
	c, err := Load(metadata.CNI)
	if err != nil {
		return
	}
	if metadata.CNINamespace == "" {
		err = errors.New("the namespace of cni is empty")
		return
	}

	return c.Create().CmdList(metadata.CNINamespace), nil
}
