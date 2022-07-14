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

package cri

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ContainerdRunnable struct {
	Base
	LocalRegistry string `json:"localRegistry"`
	KubeVersion   string `json:"kubeVersion"`
	PauseVersion  string `json:"pauseVersion"`

	installSteps   []v1.Step
	uninstallSteps []v1.Step
	upgradeSteps   []v1.Step
}

func (runnable *ContainerdRunnable) InitStep(ctx context.Context, containerd *v1.Containerd, nodes []v1.StepNode) error {
	metadata := component.GetExtraMetadata(ctx)
	runnable.Version = containerd.Version
	runnable.Offline = metadata.Offline
	runnable.DataRootDir = strutil.StringDefaultIfEmpty(containerdDefaultConfigDir, containerd.DataRootDir)
	runnable.LocalRegistry = metadata.LocalRegistry
	runnable.InsecureRegistry = containerd.InsecureRegistry
	runnable.PauseVersion = runnable.matchPauseVersion(metadata.KubeVersion)
	runtimeBytes, err := json.Marshal(runnable)
	if err != nil {
		return err
	}

	//nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	if len(runnable.installSteps) == 0 {
		runnable.installSteps = []v1.Step{
			{
				ID:         strutil.GetUUID(),
				Name:       "installRuntime",
				Timeout:    metav1.Duration{Duration: 10 * time.Minute},
				ErrIgnore:  false,
				RetryTimes: 1,
				Nodes:      nodes,
				Action:     v1.ActionInstall,
				Commands: []v1.Command{
					{
						Type:          v1.CommandCustom,
						Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, criContainerd, criVersion, component.TypeStep),
						CustomCommand: runtimeBytes,
					},
				},
			},
		}
	}
	if len(runnable.uninstallSteps) == 0 {
		runnable.uninstallSteps = []v1.Step{
			{
				ID:         strutil.GetUUID(),
				Name:       "uninstallRuntime",
				Timeout:    metav1.Duration{Duration: 10 * time.Minute},
				ErrIgnore:  false,
				RetryTimes: 1,
				Nodes:      nodes,
				Action:     v1.ActionUninstall,
				Commands: []v1.Command{
					{
						Type:          v1.CommandCustom,
						Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, criContainerd, criVersion, component.TypeStep),
						CustomCommand: runtimeBytes,
					},
				},
			},
		}
	}

	return nil
}

func (runnable *ContainerdRunnable) GetActionSteps(action v1.StepAction) []v1.Step {
	switch action {
	case v1.ActionInstall:
		return runnable.installSteps
	case v1.ActionUninstall:
		return runnable.uninstallSteps
	case v1.ActionUpgrade:
		return runnable.upgradeSteps
	}

	return nil
}

func (runnable *ContainerdRunnable) setParams() {
	runnable.Arch = component.OSArchAMD64
}

func (runnable *ContainerdRunnable) NewInstance() component.ObjectMeta {
	return &ContainerdRunnable{}
}

func (runnable ContainerdRunnable) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	runnable.setParams()
	instance, err := downloader.NewInstance(ctx, criContainerd, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	// generate containerd daemon config file
	if err = runnable.setupContainerdConfig(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// launch and enable containerd service
	if err = runnable.enableContainerdService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// crictl config runtime-endpoint /run/containerd/containerd.sock
	_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "crictl", "config", "runtime-endpoint", "/run/containerd/containerd.sock")
	if err != nil {
		return nil, err
	}
	logger.Debugf("install containerd successfully, online: %b", !runnable.Offline)
	return nil, nil
}

func (runnable ContainerdRunnable) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	runnable.setParams()
	if err := runnable.disableContainerdService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// remove related binary configuration files
	instance, err := downloader.NewInstance(ctx, criContainerd, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err = instance.RemoveConfigs(); err != nil {
		return nil, err
	}
	// remove containerd run dir
	if err = os.RemoveAll("/run/containerd"); err == nil {
		logger.Debug("remove containerd config dir successfully")
	}
	// remove containerd data dir
	if err = os.RemoveAll(strutil.StringDefaultIfEmpty(containerdDefaultConfigDir, runnable.DataRootDir)); err == nil {
		logger.Debug("remove containerd data dir successfully")
	}
	// remove containerd config dir
	if err = os.RemoveAll(containerdDefaultConfigDir); err == nil {
		logger.Debug("remove containerd config dir successfully")
	}
	// remove containerd data
	if err = os.RemoveAll(containerdDefaultDataDir); err == nil {
		logger.Debug("remove containerd systemd config successfully")
	}
	logger.Debug("uninstall containerd successfully")
	return nil, nil
}

func (runnable *ContainerdRunnable) OfflineUpgrade(ctx context.Context, dryRun bool) ([]byte, error) {
	return nil, fmt.Errorf("no support offlineUpgrade containerdRunnable")
}

func (runnable *ContainerdRunnable) OnlineUpgrade(ctx context.Context, dryRun bool) ([]byte, error) {
	return nil, fmt.Errorf("no support onlineUpgrade containerdRunnable")
}

func (runnable *ContainerdRunnable) matchPauseVersion(kubeVersion string) string {
	if kubeVersion == "" {
		return ""
	}
	kubeVersion = strings.ReplaceAll(kubeVersion, "v", "")
	kubeVersion = strings.ReplaceAll(kubeVersion, ".", "")

	kubeVersion = strings.Join(strings.Split(kubeVersion, "")[0:3], "")
	return k8sMatchPauseVersion[kubeVersion]
}

func (runnable *ContainerdRunnable) setupContainerdConfig(ctx context.Context, dryRun bool) error {
	cf := filepath.Join(containerdDefaultConfigDir, "config.toml")
	if err := os.MkdirAll(containerdDefaultConfigDir, 0755); err != nil {
		return err
	}
	return fileutil.WriteFileWithContext(ctx, cf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644, runnable.renderTo, dryRun)
}

func (runnable *ContainerdRunnable) enableContainerdService(ctx context.Context, dryRun bool) error {
	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "daemon-reload")
	if err != nil {
		return err
	}
	_, err = cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "enable", "containerd")
	if err != nil {
		return err
	}
	// restart containerd to active config, if it is already running
	_, err = cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "restart", "containerd")
	if err != nil {
		return err
	}
	logger.Debug("enable containerd systemd service successfully")
	return nil
}

func (runnable *ContainerdRunnable) disableContainerdService(ctx context.Context, dryRun bool) error {
	// stop systemd containerd service
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "stop", "containerd"); err != nil {
		return err
	}
	// disable systemd containerd service
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "disable", "containerd"); err != nil {
		return err
	}
	return nil
}

func (runnable *ContainerdRunnable) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, configTomlTemplate, runnable)
	return err
}
