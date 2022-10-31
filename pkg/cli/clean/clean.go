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

package clean

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/deploy"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

const (
	longDescription = `
  Uninstall kubeclipper Platform .

  Uninstall all kubeclipper plug-ins.`
	cleanExample = `
  # Uninstall the entire kubeclipper platform.
  kcctl clean --all
  kcctl clean -A

  # Mock uninstall,without -A flag will only do preCheck and config check.
  kcctl clean

  # Uninstall the entire kubeclipper platform,use specify the auth config.
  kcctl clean -A --config ~/.kc/config

  # Uninstall the entire kubeclipper platform,use local deploy config when kc-server is not health.
  kcctl clean -A -f

  Please read 'kcctl clean -h' get more clean flags`
)

type CleanOptions struct {
	options.IOStreams
	cliOpts      *options.CliOptions
	deployConfig *options.DeployConfig
	client       *kc.Client
	cleanAll     bool
	force        bool

	allNodes []string
}

func NewCleanOptions(streams options.IOStreams) *CleanOptions {
	return &CleanOptions{
		IOStreams:    streams,
		cliOpts:      options.NewCliOptions(),
		deployConfig: options.NewDeployOptions(),
	}
}

func NewCmdClean(streams options.IOStreams) *cobra.Command {
	o := NewCleanOptions(streams)
	cmd := &cobra.Command{
		Use:                   "clean [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Uninstall kubeclipper platform",
		Long:                  longDescription,
		Example:               cleanExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			if !o.preCheck() {
				return
			}
			o.RunClean()
			fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
		},
	}
	o.cliOpts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVarP(&o.cleanAll, "all", "A", o.cleanAll, "clean all components for kubeclipper")
	cmd.Flags().BoolVarP(&o.force, "force", "f", o.force, "force use local deploy config to clean kubeclipper when kc-server not health")
	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "path to the deploy config file to use for clean,just work with force flag.")
	return cmd
}

func (c *CleanOptions) Complete() error {

	if c.force {
		if err := c.deployConfig.Complete(); err != nil {
			return errors.WithMessage(err, "get local deploy-config failed")
		}
	} else {
		// config Complete
		var err error
		if err = c.cliOpts.Complete(); err != nil {
			return err
		}
		c.client, err = kc.FromConfig(c.cliOpts.ToRawConfig())
		if err != nil {
			return err
		}

		// deploy-config Complete
		c.deployConfig, err = deploy.GetDeployConfig(context.Background(), c.client, false)
		if err != nil {
			return errors.WithMessage(err, "get online deploy-config failed")
		}
	}

	c.allNodes = sets.NewString().
		Insert(c.deployConfig.ServerIPs...).
		Insert(c.deployConfig.Agents.ListIP()...).
		List()
	return nil
}

func (c *CleanOptions) preCheck() bool {
	return sudo.PreCheck("sudo", c.deployConfig.SSHConfig, c.IOStreams, c.allNodes)
}

func (c *CleanOptions) RunClean() {
	if c.cleanAll {
		c.cleanKcAgent()
		c.cleanKcProxy()
		c.cleanKcServer()
		c.cleanKcConsole()
		c.cleanBinaries()
		c.cleanKcEnv()
		c.cleanKcConfig()
	}
	logger.Info("clean successful")
}

func (c *CleanOptions) cleanKcAgent() {
	if len(c.deployConfig.Agents.ListIP()) == 0 {
		logger.V(2).Info("no kubeclipper agent need to be cleaned")
		return
	}

	cmdList := []string{
		"systemctl disable kc-agent --now",
		"rm -rf /usr/lib/systemd/system/kc-agent.service",
		"rm -rf /etc/kubeclipper-agent",
		fmt.Sprintf("rm -rf %s", c.deployConfig.OpLog.Dir),
		"systemctl reset-failed kc-agent || true",
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.deployConfig.Agents.ListIP(), cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc agent failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanKcProxy() {
	if len(c.deployConfig.Proxys) == 0 {
		logger.V(2).Info("no kubeclipper proxy need to be cleaned")
		return
	}

	cmdList := []string{
		"systemctl disable kc-proxy --now",
		"rm -rf /usr/lib/systemd/system/kc-proxy.service",
		"rm -rf /etc/kubeclipper-proxy",
		"systemctl reset-failed kc-proxy || true",
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.deployConfig.Proxys, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc proxy failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanKcServer() {
	if len(c.deployConfig.ServerIPs) == 0 {
		logger.V(2).Info("no kubeclipper server need to be cleaned")
		return
	}

	cmdList := []string{
		"systemctl disable kc-server --now",
		"rm -rf /usr/lib/systemd/system/kc-server.service",
		"systemctl disable kc-etcd --now",
		"rm -rf /usr/lib/systemd/system/kc-etcd.service",
		"rm -rf /etc/kubeclipper-server",
		fmt.Sprintf("rm -rf %s", c.deployConfig.EtcdConfig.DataDir),
		fmt.Sprintf("rm -rf %s", c.deployConfig.StaticServerPath),
		"systemctl reset-failed kc-etcd || true",
		"systemctl reset-failed kc-server || true",
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc server failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanKcConsole() {
	if len(c.deployConfig.ServerIPs) == 0 {
		logger.V(2).Info("no kubeclipper console need to be cleaned")
		return
	}

	cmdList := []string{
		"systemctl disable kc-console --now",
		"rm -rf /usr/lib/systemd/system/kc-console.service",
		"rm -rf /etc/kc-console",
		"systemctl reset-failed kc-console || true",
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc console failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanBinaries() {
	if len(c.allNodes) == 0 {
		logger.V(2).Info("no kubeclipper node need to be cleaned")
		return
	}

	cmdList := []string{
		"rm -rf /usr/local/bin/kubeclipper* && rm -rf /usr/local/bin/etcd*  && rm -rf /usr/local/bin/caddy",
	}

	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.allNodes, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc binary failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanKcEnv() {
	if len(c.deployConfig.ServerIPs) == 0 {
		logger.V(2).Info("no kubeclipper console need to be cleaned")
		return
	}

	cmdList := []string{
		"rm -rf /etc/kc/kc.env",
	}

	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Warn("clean kc env failed,reason: ", err)
		}
	}
}

func (c *CleanOptions) cleanKcConfig() {
	if err := sshutils.Cmd("rm", "-rf", filepath.Dir(options.DefaultDeployConfigPath)); err != nil {
		logger.Warn("clean kc config failed,reason: ", err)
	}
}
