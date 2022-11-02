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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

type DockerRunnable struct {
	Base
	InsecureRegistry []string `json:"insecureRegistry,omitempty"`
	installSteps     []v1.Step
	uninstallSteps   []v1.Step
	upgradeSteps     []v1.Step
}

func (runnable *DockerRunnable) InitStep(ctx context.Context, cluster *v1.Cluster, nodes []v1.StepNode) error {
	metadata := component.GetExtraMetadata(ctx)

	runnable.Version = cluster.ContainerRuntime.Version
	runnable.Offline = metadata.Offline
	runnable.DataRootDir = cluster.ContainerRuntime.DataRootDir
	runnable.InsecureRegistry = ToDockerInsecureRegistry(cluster.Status.Registries)

	runtimeBytes, err := json.Marshal(runnable)
	if err != nil {
		return err
	}

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
						Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, criDocker, criVersion, component.TypeStep),
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
						Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, criDocker, criVersion, component.TypeStep),
						CustomCommand: runtimeBytes,
					},
				},
			},
		}
	}

	return nil
}

func (runnable *DockerRunnable) GetActionSteps(action v1.StepAction) []v1.Step {
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

func (runnable *DockerRunnable) NewInstance() component.ObjectMeta {
	return &DockerRunnable{}
}

func (runnable DockerRunnable) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, criDocker, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	// generate docker daemon config file
	if err = runnable.setupDockerConfig(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// launch and enable docker service
	if err = runnable.enableDockerService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	logger.Debug("install docker offline successfully")
	return nil, nil
}

func (runnable DockerRunnable) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	if err := runnable.disableDockerService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// remove related binary configuration files
	instance, err := downloader.NewInstance(ctx, criDocker, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err = instance.RemoveConfigs(); err != nil {
		logger.Error("remove docker configs compressed file failed", zap.Error(err))
	}
	// remove unix socket
	list := []string{"docker.sock", "dockershim.sock"}
	for _, sock := range list {
		if err = os.Remove(filepath.Join("/var/run", sock)); err != nil {
			logger.Debugf("remove %s successfully", sock)
		}
	}
	// remove docker data dir
	if err = os.RemoveAll(dockerDefaultCriDir); err == nil {
		logger.Debug("remove /etc/containerd cri dir successfully")
	}
	if err = os.RemoveAll(strutil.StringDefaultIfEmpty(dockerDefaultDataDir, runnable.DataRootDir)); err == nil {
		logger.Debug("remove docker data dir successfully")
	}
	// remove docker config dir
	if err = os.RemoveAll(dockerDefaultConfigDir); err == nil {
		logger.Debug("remove docker config dir successfully")
	}
	return nil, nil
}

func (runnable *DockerRunnable) OfflineUpgrade(ctx context.Context, dryRun bool) ([]byte, error) {
	return nil, fmt.Errorf("DockerRunnable dose not supported offlineUpgrade")
}

func (runnable *DockerRunnable) OnlineUpgrade(ctx context.Context, dryRun bool) ([]byte, error) {
	return nil, fmt.Errorf("DockerRunnable dose not supported onlineUpgrade")
}

func (runnable *DockerRunnable) setupDockerConfig(ctx context.Context, dryRun bool) error {
	cf := filepath.Join(dockerDefaultConfigDir, "daemon.json")
	if err := os.MkdirAll(dockerDefaultConfigDir, 0755); err != nil {
		return err
	}
	return fileutil.WriteFileWithContext(ctx, cf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644, runnable.renderTo, dryRun)
}

func (runnable *DockerRunnable) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, daemonConfigTemplate, runnable)
	return err
}

func (runnable *DockerRunnable) enableDockerService(ctx context.Context, dryRun bool) error {
	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "daemon-reload")
	if err != nil {
		return err
	}
	_, err = cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "enable", "docker", "--now")
	if err != nil {
		return err
	}
	return nil
}

func (runnable *DockerRunnable) disableDockerService(ctx context.Context, dryRun bool) error {
	// the following command execution error is ignored
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "stop", "docker"); err != nil {
		logger.Warn("stop systemd docker service failed", zap.Error(err))
	}
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "disable", "docker"); err != nil {
		logger.Warn("disable systemd docker service failed", zap.Error(err))
	}
	return nil
}

type DockerInsecureRegistryConfigure struct {
	InsecureRegistry []string `json:"insecureRegistry,omitempty"`
}

const (
	defaultDaemonConfigFile = "/etc/docker/daemon.json"
)

func (d *DockerInsecureRegistryConfigure) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	conf := make(map[string]interface{})
	// TODO：Maybe the configuration file is not here
	configFile := defaultDaemonConfigFile
	buf, err := os.ReadFile(configFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config file: %s failed: %w",
			configFile, err)
	}
	if len(buf) != 0 {
		err = json.Unmarshal(buf, &conf)
		if err != nil {
			return nil, fmt.Errorf("unmarshal config file: %s failed: %w",
				configFile, err)
		}
	}

	conf["insecure-registries"] = d.InsecureRegistry
	configContent, err := json.MarshalIndent(&conf, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("encode config: %w:%#v", err, &conf)
	}
	if opts.DryRun {
		logger.Infof("dryRun: render config :%s", configContent)
	} else {
		f, err := os.OpenFile(configFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open config file: %s :%w", configFile, err)
		}
		defer func() {
			_ = f.Close()
		}()
		_, err = f.Write(configContent)
		if err != nil {
			return nil, fmt.Errorf("write config file to: %s :%w", configFile, err)
		}
		_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "systemctl reload docker")
		if err != nil {
			return nil, fmt.Errorf("reload docker:%w", err)
		}
	}
	return nil, nil
}

func (d *DockerInsecureRegistryConfigure) Uninstall(_ context.Context, _ component.Options) ([]byte, error) {
	// nothing
	return nil, nil
}

func (d *DockerInsecureRegistryConfigure) NewInstance() component.ObjectMeta {
	return new(DockerInsecureRegistryConfigure)
}

func ToDockerInsecureRegistry(registries []v1.RegistrySpec) []string {
	if len(registries) == 0 {
		return nil
	}
	insecureRegistry := make([]string, 0, len(registries))
	for _, r := range registries {
		// registry used validate tls certificate
		if r.Scheme == "https" && !r.SkipVerify && len(r.CA) == 0 {
			continue
		}
		index, find := sort.Find(len(insecureRegistry), func(i int) int {
			return strings.Compare(r.Host, insecureRegistry[i])
		})
		if find {
			continue
		}
		if index == len(insecureRegistry) {
			insecureRegistry = append(insecureRegistry, r.Host)
		} else {
			insecureRegistry = append(insecureRegistry[:index+1], insecureRegistry[:index+1]...)
			insecureRegistry[index] = r.Host
		}
	}
	return insecureRegistry
}
