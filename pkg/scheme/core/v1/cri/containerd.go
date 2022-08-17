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
	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ContainerdRunnable struct {
	Base
	RegistryConfigDir string `json:"registryConfigDir"`
	LocalRegistry     string `json:"localRegistry"`
	KubeVersion       string `json:"kubeVersion"`
	PauseVersion      string `json:"pauseVersion"`

	installSteps   []v1.Step
	uninstallSteps []v1.Step
	upgradeSteps   []v1.Step
}

func (runnable *ContainerdRunnable) InitStep(ctx context.Context, containerd *v1.ContainerRuntime, nodes []v1.StepNode) error {
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

	// nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
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
		logger.Error("remove contanierd configs compressed file failed", zap.Error(err))
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
	// local registry not filled and is in online mode, the default repo mirror proxy will be used
	if !runnable.Offline && runnable.LocalRegistry == "" {
		runnable.LocalRegistry = component.GetRepoMirror(ctx)
		logger.Info("render containerd config, the default repo mirror proxy will be used", zap.String("local_registry", runnable.LocalRegistry))
	}
	cf := filepath.Join(containerdDefaultConfigDir, "config.toml")
	if err := os.MkdirAll(containerdDefaultConfigDir, 0755); err != nil {
		return err
	}
	if err := fileutil.WriteFileWithContext(ctx, cf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644, runnable.renderTo, dryRun); err != nil {
		return err
	}
	return runnable.renderRegistryConfig(dryRun)
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
	// the following command execution error is ignored
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "stop", "containerd"); err != nil {
		logger.Warn("stop systemd containerd service failed", zap.Error(err))
	}
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "disable", "containerd"); err != nil {
		logger.Warn("disable systemd containerd service failed", zap.Error(err))
	}
	return nil
}

func (runnable *ContainerdRunnable) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, configTomlTemplate, runnable)
	return err
}

func (runnable *ContainerdRunnable) renderRegistryConfig(dryRun bool) error {
	if dryRun {
		return nil
	}
	configDir := containerdDefaultRegistryConfigDir
	if runnable.RegistryConfigDir != "" {
		configDir = runnable.RegistryConfigDir
	}
	regCfgs := make([]ContainerdRegistry, 0, len(runnable.InsecureRegistry)+1)

	for _, regHost := range runnable.InsecureRegistry {
		regCfgs = append(regCfgs, ContainerdRegistry{
			Server: regHost,
			Hosts: []ContainerdHost{
				{
					Host:         regHost,
					Capabilities: []string{CapabilityPull, CapabilityResolve},
					SkipVerify:   true,
				},
			},
		},
		)
	}

	if runnable.LocalRegistry != "" {
		regCfgs = append(
			regCfgs,
			ContainerdRegistry{
				Server: runnable.LocalRegistry,
				Hosts: []ContainerdHost{
					{
						Host:         runnable.LocalRegistry,
						Capabilities: []string{CapabilityPull, CapabilityResolve},
						SkipVerify:   true,
					},
				},
			},
		)
	}

	for _, cfg := range regCfgs {
		if err := cfg.renderConfigs(configDir); err != nil {
			return err
		}
	}
	return nil
}

const (
	CapabilityPull    = "pull"
	CapabilityPush    = "push"
	CapabilityResolve = "resolve"
)

type ContainerdHost struct {
	Host         string
	Capabilities []string
	SkipVerify   bool
	CA           []byte
}

type ContainerdRegistry struct {
	Server string // not contain scheme, example: docker.io
	Hosts  []ContainerdHost
}

// generate hosts.toml and ca file
func (h *ContainerdRegistry) renderConfigs(dir string) error {
	hostDir := filepath.Join(dir, h.Server)
	err := os.MkdirAll(hostDir, 0755)
	if err != nil {
		return err
	}

	c := HostFile{
		Server:      h.Server,
		HostConfigs: make(map[string]HostFileConfig),
	}
	for _, host := range h.Hosts {
		var (
			caFile     = ""
			skipVerify *bool
		)
		if host.SkipVerify {
			b := host.SkipVerify
			skipVerify = &b
		}
		if len(host.CA) > 0 {
			caFile = filepath.Join(hostDir, fmt.Sprintf("%s.pem", host.Host))
			if err = os.WriteFile(caFile, host.CA, 0666); err != nil {
				return err
			}
		}
		hc := HostFileConfig{
			Capabilities: host.Capabilities,
			SkipVerify:   skipVerify,
		}
		if caFile != "" {
			hc.CACert = caFile
		}
		c.HostConfigs[fmt.Sprintf("https://%s", host.Host)] = hc

		// support http scheme
		if skipVerify != nil && *skipVerify {
			c.HostConfigs[fmt.Sprintf("http://%s", host.Host)] = HostFileConfig{
				Capabilities: host.Capabilities,
			}
		}
	}
	f, err := os.Create(filepath.Join(hostDir, "hosts.toml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

type HostFileConfig struct {
	// Capabilities determine what operations a host is
	// capable of performing. Allowed values
	//  - pull
	//  - resolve
	//  - push
	Capabilities []string `toml:"capabilities,omitempty"`

	// CACert are the public key certificates for TLS
	// Accepted types
	// - string - Single file with certificate(s)
	// - []string - Multiple files with certificates
	CACert interface{} `toml:"ca,omitempty,omitempty"`

	// Client keypair(s) for TLS with client authentication
	// Accepted types
	// - string - Single file with public and private keys
	// - []string - Multiple files with public and private keys
	// - [][2]string - Multiple keypairs with public and private keys in separate files
	Client interface{} `toml:"client,omitempty"`

	// SkipVerify skips verification of the server's certificate chain
	// and host name. This should only be used for testing or in
	// combination with other methods of verifying connections.
	SkipVerify *bool `toml:"skip_verify,omitempty"`

	// Header are additional header files to send to the server
	Header map[string]interface{} `toml:"header,omitempty"`

	// OverridePath indicates the API root endpoint is defined in the URL
	// path rather than by the API specification.
	// This may be used with non-compliant OCI registries to override the
	// API root endpoint.
	OverridePath bool `toml:"override_path,omitempty"`

	// TODO: Credentials: helper? name? username? alternate domain? token?
}

type HostFile struct {
	// Server specifies the default server. When `host` is
	// also specified, those hosts are tried first.
	Server string `toml:"server"`
	// HostConfigs store the per-host configuration
	HostConfigs map[string]HostFileConfig `toml:"host"`
}
