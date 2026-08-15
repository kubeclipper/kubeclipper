/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const (
	apiTimeout      = 10 * time.Second
	nodeConcurrency = 5
)

type ExitError struct {
	code int
	err  error
}

func (e *ExitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *ExitError) Unwrap() error { return e.err }
func (e *ExitError) ExitCode() int { return e.code }

type Options struct {
	options.IOStreams
	configPath       string
	deployConfigPath string
	now              func() time.Time
}

type diagnosticState struct {
	client       *kc.Client
	platform     *platformstatus.PlatformStatus
	nodes        []corev1.Node
	nodesLoaded  bool
	deployConfig *options.DeployConfig
	remote       *remoteRunner
}

type deployConfigFile struct {
	SSH              sshConfigFile  `json:"ssh" yaml:"ssh"`
	Etcd             etcdConfigFile `json:"etcd" yaml:"etcd"`
	ServerIPs        []string       `json:"serverIPs" yaml:"serverIPs"`
	Agents           options.Agents `json:"agents" yaml:"agents"`
	ServerPort       int            `json:"serverPort" yaml:"serverPort"`
	TLS              bool           `json:"tls" yaml:"tls"`
	StaticServerPort int            `json:"staticServerPort" yaml:"staticServerPort"`
	StaticServerPath string         `json:"staticServerPath" yaml:"staticServerPath"`
}

type sshConfigFile struct {
	User       string `json:"user" yaml:"user"`
	Password   string `json:"password" yaml:"password"`
	Port       int    `json:"port" yaml:"port"`
	PkFile     string `json:"pkFile" yaml:"pkFile"`
	PrivateKey string `json:"privateKey" yaml:"privateKey"`
	PkPassword string `json:"pkPassword" yaml:"pkPassword"`
}

type etcdConfigFile struct {
	ClientPort  int    `json:"clientPort" yaml:"clientPort"`
	PeerPort    int    `json:"peerPort" yaml:"peerPort"`
	MetricsPort int    `json:"metricsPort" yaml:"metricsPort"`
	DataDir     string `json:"dataDir" yaml:"dataDir"`
}

func NewCmdDoctor(streams options.IOStreams) *cobra.Command {
	o := &Options{
		IOStreams:        streams,
		configPath:       options.DefaultConfigPath,
		deployConfigPath: options.DefaultDeployConfigPath,
		now:              time.Now,
	}
	cmd := &cobra.Command{
		Use:          "doctor",
		Short:        "Diagnose KubeClipper platform problems",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Root().SilenceErrors = true
			report := o.run(cmd.Context())
			if err := printReport(o.Out, report); err != nil {
				return &ExitError{code: 2, err: err}
			}
			if report.Status != platformstatus.Healthy {
				return &ExitError{code: 1}
			}
			return nil
		},
	}
	return cmd
}

func (o *Options) run(ctx context.Context) *Report {
	startedAt := o.now()
	state := &diagnosticState{}
	local := o.checkLocalAccess(ctx, state)
	components := []Component{
		local,
		checkKCServer(ctx, state),
		checkKCEtcd(ctx, state),
		checkKCAgents(ctx, state),
	}
	return newReport(startedAt, o.now().Sub(startedAt), components)
}

func (o *Options) checkLocalAccess(ctx context.Context, state *diagnosticState) Component {
	component := Component{Name: "kcctl"}
	deployConfig, err := loadDeployConfig(o.deployConfigPath)
	if err != nil {
		status := platformstatus.Degraded
		message := fmt.Sprintf("cannot load %s", o.deployConfigPath)
		if !errors.Is(err, os.ErrNotExist) {
			message = fmt.Sprintf("invalid deployment configuration: %v", err)
		}
		component.Checks = append(component.Checks, Check{
			Name: "deploy-config", Status: status, Message: message,
			Evidence: []string{"remote service, network and journal checks will be skipped"},
		})
	} else {
		state.deployConfig = deployConfig
		state.remote = newRemoteRunner(ctx, deployConfig.SSHConfig)
		component.Checks = append(component.Checks, Check{
			Name: "deploy-config", Status: platformstatus.Healthy, Message: "deployment configuration loaded",
		})
	}

	apiConfig, err := config.TryLoadFromFile(o.configPath)
	if err != nil {
		component.Checks = append(component.Checks, Check{
			Name: apiConfigCheck, Status: platformstatus.Unhealthy,
			Message: fmt.Sprintf("cannot load API configuration %s", o.configPath), Evidence: []string{sanitize(err.Error())},
		})
		component.Message = "API configuration is unavailable"
		return component
	}
	client, err := kc.FromConfigWithoutValidation(*apiConfig)
	if err != nil {
		component.Checks = append(component.Checks, Check{
			Name: apiConfigCheck, Status: platformstatus.Unhealthy,
			Message: "API configuration is invalid", Evidence: []string{sanitize(err.Error())},
		})
		component.Message = "API configuration is invalid"
		return component
	}
	state.client = client
	component.Checks = append(component.Checks, Check{
		Name: apiConfigCheck, Status: platformstatus.Healthy, Message: "API configuration loaded",
	})

	apiCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	healthErr := client.Healthz(apiCtx)
	if healthErr != nil {
		component.Checks = append(component.Checks, Check{
			Name: "api-health", Status: platformstatus.Unhealthy, Message: "KubeClipper API is unreachable",
			Evidence: []string{sanitize(healthErr.Error())},
		})
		component.Message = "API is unreachable"
		return component
	}
	component.Checks = append(component.Checks, Check{
		Name: "api-health", Status: platformstatus.Healthy, Message: "API liveness check passed",
	})

	statusCtx, statusCancel := context.WithTimeout(ctx, apiTimeout)
	defer statusCancel()
	platform, err := client.PlatformStatus(statusCtx)
	if err != nil {
		component.Checks = append(component.Checks, Check{
			Name: "api-authentication", Status: platformstatus.Unhealthy,
			Message: "cannot read platform status", Evidence: []string{sanitize(err.Error())},
		})
		component.Message = "API is reachable but status access failed"
		return component
	}
	state.platform = platform
	component.Checks = append(component.Checks, Check{
		Name: "api-authentication", Status: platformstatus.Healthy,
		Message: "API is reachable and authenticated",
	})

	nodesCtx, nodesCancel := context.WithTimeout(ctx, apiTimeout)
	defer nodesCancel()
	nodes, err := client.ListNodes(nodesCtx, kc.Queries(*query.New()))
	if err != nil {
		component.Checks = append(component.Checks, Check{
			Name: "node-inventory", Status: platformstatus.Degraded, Message: "cannot read agent inventory",
			Evidence: []string{sanitize(err.Error())},
		})
	} else {
		state.nodes = nodes.Items
		state.nodesLoaded = true
		component.Checks = append(component.Checks, Check{
			Name: "node-inventory", Status: platformstatus.Healthy, Message: "agent inventory loaded",
		})
	}
	component.Message = "connected to KubeClipper API"
	return component
}

func loadDeployConfig(path string) (*options.DeployConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defaults := options.NewDeployOptions()
	fileConfig := deployConfigFile{
		SSH: sshConfigFile{User: defaults.SSHConfig.User, Port: defaults.SSHConfig.Port},
		Etcd: etcdConfigFile{
			ClientPort: defaults.EtcdConfig.ClientPort, PeerPort: defaults.EtcdConfig.PeerPort,
			MetricsPort: defaults.EtcdConfig.MetricsPort, DataDir: defaults.EtcdConfig.DataDir,
		},
		Agents: defaults.Agents, ServerPort: defaults.ServerPort, TLS: defaults.TLS,
		StaticServerPort: defaults.StaticServerPort, StaticServerPath: defaults.StaticServerPath,
	}
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, err
	}
	deployConfig := defaults
	deployConfig.Config = path
	deployConfig.SSHConfig = &sshutils.SSH{
		User: fileConfig.SSH.User, Password: fileConfig.SSH.Password, Port: fileConfig.SSH.Port,
		PkFile: fileConfig.SSH.PkFile, PrivateKey: fileConfig.SSH.PrivateKey, PkPassword: fileConfig.SSH.PkPassword,
	}
	deployConfig.EtcdConfig = &options.Etcd{
		ClientPort: fileConfig.Etcd.ClientPort, PeerPort: fileConfig.Etcd.PeerPort,
		MetricsPort: fileConfig.Etcd.MetricsPort, DataDir: fileConfig.Etcd.DataDir,
	}
	deployConfig.ServerIPs = fileConfig.ServerIPs
	deployConfig.Agents = fileConfig.Agents
	deployConfig.ServerPort = fileConfig.ServerPort
	deployConfig.TLS = fileConfig.TLS
	deployConfig.StaticServerPort = fileConfig.StaticServerPort
	deployConfig.StaticServerPath = fileConfig.StaticServerPath
	if len(deployConfig.ServerIPs) == 0 {
		return nil, fmt.Errorf("serverIPs is empty")
	}
	return deployConfig, nil
}

func runParallel[T any](items []T, check func(T) []Check) []Check {
	semaphore := make(chan struct{}, nodeConcurrency)
	results := make(chan []Check, len(items))
	var wg sync.WaitGroup
	for i := range items {
		item := items[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results <- check(item)
		}()
	}
	wg.Wait()
	close(results)
	var checks []Check
	for result := range results {
		checks = append(checks, result...)
	}
	sortChecks(checks)
	return checks
}

func platformComponent(status *platformstatus.PlatformStatus, name string) *platformstatus.Component {
	if status == nil {
		return nil
	}
	for i := range status.Components {
		if status.Components[i].Name == name {
			return &status.Components[i]
		}
	}
	return nil
}

func platformCheck(component *platformstatus.Component, name string) *platformstatus.Check {
	if component == nil {
		return nil
	}
	for i := range component.Checks {
		if component.Checks[i].Name == name {
			return &component.Checks[i]
		}
	}
	return nil
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
