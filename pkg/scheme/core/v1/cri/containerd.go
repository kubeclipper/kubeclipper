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
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
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

type ContainerdRunnable struct {
	Base
	RegistryConfigDir   string `json:"registryConfigDir"`
	LocalRegistry       string `json:"localRegistry"`
	KubeVersion         string `json:"kubeVersion"`
	PauseVersion        string `json:"pauseVersion"`
	PauseRegistry       string `json:"pauseRegistry"`
	EnableSystemdCgroup string `json:"enableSystemdCgroup"`

	installSteps   []v1.Step
	uninstallSteps []v1.Step
	upgradeSteps   []v1.Step
}

func (runnable *ContainerdRunnable) InitStep(ctx context.Context, cluster *v1.Cluster, nodes []v1.StepNode) error {
	metadata := component.GetExtraMetadata(ctx)
	runnable.Version = cluster.ContainerRuntime.Version
	runnable.Offline = metadata.Offline
	runnable.DataRootDir = strutil.StringDefaultIfEmpty(containerdDefaultConfigDir, cluster.ContainerRuntime.DataRootDir)
	runnable.LocalRegistry = metadata.LocalRegistry
	runnable.Registies = cluster.Status.Registries

	runnable.PauseVersion, runnable.PauseRegistry = runnable.matchPauseVersion(metadata.KubeVersion)
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

func (runnable *ContainerdRunnable) NewInstance() component.ObjectMeta {
	return &ContainerdRunnable{}
}

func (runnable ContainerdRunnable) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, criContainerd, runnable.Version, runtime.GOARCH, !runnable.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	runnable.EnableSystemdCgroup = "false"
	// check whether cgroup2 is used as the cgroup driver, if is it, enable containerd systemd cgroup
	res, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "cat /proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	if strings.Contains(res.StdOut(), "cgroup2") {
		runnable.EnableSystemdCgroup = "true"
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
	return nil, fmt.Errorf("ContainerdRunnable dose not support offlineUpgrade")
}

func (runnable *ContainerdRunnable) OnlineUpgrade(ctx context.Context, dryRun bool) ([]byte, error) {
	return nil, fmt.Errorf("ContainerdRunnable not supported onlineUpgrade")
}

func (runnable *ContainerdRunnable) matchPauseVersion(kubeVersion string) (string, string) {
	registry := "k8s.gcr.io"
	if kubeVersion == "" {
		return "", registry
	}
	kubeVersion = strings.ReplaceAll(kubeVersion, "v", "")
	kubeVersion = strings.ReplaceAll(kubeVersion, ".", "")

	kubeVersion = strings.Join(strings.Split(kubeVersion, "")[0:3], "")

	if v, _ := strconv.Atoi(kubeVersion); v >= 125 {
		registry = "registry.k8s.io"
	}
	return k8sMatchPauseVersion[kubeVersion], registry
}

func (runnable *ContainerdRunnable) setupContainerdConfig(ctx context.Context, dryRun bool) error {
	// local registry not filled and is in online mode, the default repo mirror proxy will be used
	if !runnable.Offline && runnable.LocalRegistry == "" {
		runnable.LocalRegistry = component.GetRepoMirror(ctx)
		logger.Info("render containerd config, the default repo mirror proxy will be used", zap.String("local_registry", runnable.LocalRegistry))
	}
	if runnable.RegistryConfigDir == "" {
		runnable.RegistryConfigDir = ContainerdDefaultRegistryConfigDir
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
	regCfgs := ToContainerdRegistryConfig(runnable.Registies)
	for _, cfg := range regCfgs {
		if err := cfg.renderConfigs(runnable.RegistryConfigDir); err != nil {
			return err
		}
	}
	return nil
}

type ContainerdRegistryConfigure struct {
	Registries map[string]*ContainerdRegistry `json:"registries,omitempty"`
	ConfigDir  string                         `json:"configDir"`
}

func (c *ContainerdRegistryConfigure) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	if opts.DryRun {
		return nil, nil
	}
	entries, err := os.ReadDir(c.ConfigDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read registry config dir:%s failed:%w", c.ConfigDir, err)
	}
	oldDirs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			oldDirs[entry.Name()] = struct{}{}
		}
	}
	for _, r := range c.Registries {
		err := r.renderConfigs(c.ConfigDir)
		if err != nil {
			return nil, fmt.Errorf("renderConfigs to %s failed:%w", c.ConfigDir, err)
		}
		delete(oldDirs, r.Server)
	}
	for d := range oldDirs {
		err = os.RemoveAll(filepath.Join(c.ConfigDir, d))
		if err != nil {
			logger.Errorf("clear old registry config dir: %s failed:%s", d, err)
		}
	}
	return nil, nil
}

func (c *ContainerdRegistryConfigure) Uninstall(_ context.Context, _ component.Options) ([]byte, error) {
	return nil, nil
}

func (c *ContainerdRegistryConfigure) NewInstance() component.ObjectMeta {
	return new(ContainerdRegistryConfigure)
}

const (
	CapabilityPull    = "pull"
	CapabilityPush    = "push"
	CapabilityResolve = "resolve"
)

type ContainerdHost struct {
	Scheme       string // http or https
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
				return fmt.Errorf("write ca file:%s failed:%w", caFile, err)
			}
		}
		hostConfig := HostFileConfig{
			Capabilities: host.Capabilities,
			SkipVerify:   skipVerify,
		}
		if caFile != "" {
			hostConfig.CACert = caFile
		}
		c.HostConfigs[fmt.Sprintf("%s://%s", host.Scheme, host.Host)] = hostConfig
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

func ToContainerdRegistryConfig(registries []v1.RegistrySpec) map[string]*ContainerdRegistry {
	cfgs := make(map[string]*ContainerdRegistry, len(registries))
	for _, r := range registries {
		cfg, ok := cfgs[r.Host]
		if !ok {
			cfg = &ContainerdRegistry{
				Server: r.Host,
			}
			cfgs[r.Host] = cfg
		}
		cfg.Hosts = append(cfg.Hosts, ContainerdHost{
			Scheme:       r.Scheme,
			Host:         r.Host,
			Capabilities: []string{CapabilityPull, CapabilityResolve},
			SkipVerify:   r.SkipVerify,
			CA:           []byte(r.CA),
		})
	}
	return cfgs
}
