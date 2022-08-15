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
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

const (
	longDescription = `
  Uninstall kubeclipper Platform from deploy-config.yaml or cmd flags.

  Uninstall all kubeclipper plug-ins.
  You can choose to uninstall it directly or using a configuration file.`
	cleanExample = `
  # Uninstall the entire kubeclipper platform use default deploy config(~/.kc/deploy-config.yaml).
  kcctl clean --all
  kcctl clean -A

  # Mock uninstall,without -A flag will only do preCheck and config check.
  kcctl clean
  kcctl clean --config ~/.kc/deploy-config.yaml


  # Specify the deploy-config.yaml path
  kcctl clean --config ~/.kc/deploy-config.yaml -A

  Please read 'kcctl clean -h' get more clean flags`
)

type CleanOptions struct {
	options.IOStreams
	deployConfig *options.DeployConfig
	cleanAll     bool
	allNodes     []string
}

func NewCleanOptions(streams options.IOStreams) *CleanOptions {
	return &CleanOptions{
		IOStreams:    streams,
		deployConfig: options.NewDeployOptions(),
	}
}

func NewCmdClean(streams options.IOStreams) *cobra.Command {
	o := NewCleanOptions(streams)
	cmd := &cobra.Command{
		Use:                   "clean [(--config [<configFilePath>])] [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Uninstall kubeclipper platform",
		Long:                  longDescription,
		Example:               cleanExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			o.ValidateArgs()
			if !o.preCheck() {
				return
			}
			o.RunClean()
			fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
		},
	}
	cmd.Flags().StringVarP(&o.deployConfig.Config, "config", "c", options.DefaultDeployConfigPath, "Path to the deploy config file to use for clean.")
	cmd.Flags().BoolVarP(&o.cleanAll, "all", "A", o.cleanAll, "clean all components for kubeclipper")
	return cmd
}

func (c *CleanOptions) Complete() error {
	c.allNodes = sets.NewString().
		Insert(c.deployConfig.ServerIPs...).
		Insert(c.deployConfig.Agents.ListIP()...).
		List()
	return c.deployConfig.Complete()
}

func (c *CleanOptions) ValidateArgs() {
	// TODO: add validate
}

func (c *CleanOptions) preCheck() bool {
	return sudo.PreCheck("sudo", c.deployConfig.SSHConfig, c.IOStreams, c.allNodes)
}

func (c *CleanOptions) RunClean() {
	if c.cleanAll {
		c.cleanKcAgent()
		c.cleanKcServer()
		c.cleanKcConsole()
		c.cleanBinaries()
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
			logger.V(2).Error(err)
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
			logger.V(2).Error(err)
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
			logger.V(2).Error(err)
		}
	}
}

func (c *CleanOptions) cleanBinaries() {
	if len(c.allNodes) == 0 {
		logger.V(2).Info("no kubeclipper node need to be cleaned")
		return
	}

	cmdList := []string{
		"rm -rf /usr/local/bin/kube* && rm -rf /usr/local/bin/etcd* && rm -rf /usr/local/bin/kcctl && rm -rf /usr/local/bin/caddy",
	}

	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(c.deployConfig.SSHConfig, c.allNodes, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.V(2).Error(err)
		}
	}
}
