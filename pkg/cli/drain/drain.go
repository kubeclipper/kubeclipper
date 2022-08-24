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

package drain

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
  Drain the Kubeclipper service or agent node from the cluster.

  Now only support drain kc-agent node, so the --agent parameter must be valid.`
	drainExample = `
  # Drain kc-agent from kubeclipper cluster use default deploy-config(~/.kc/deploy-config.yaml) and config(~/.kc/config).
  kcctl drain --agent 192.168.10.19

  # Drain kc-agent from kubeclipper cluster specify deploy config and config.
  kcctl drain --agent 192.168.10.19 --deploy-config /root/.kc/deploy-config.yaml --config /root/.kc/config

  # Force drain kc-agent which is in used from kubeclipper cluster
  kcctl drain  --force --agent=192.168.10.123

  Please read 'kcctl drain -h' get more drain flags.`
)

type DrainOptions struct {
	options.IOStreams
	client       *kc.Client
	cliOpts      *options.CliOptions
	deployConfig *options.DeployConfig

	agents  []string
	servers []string
	force   bool
}

func NewDrainOptions(streams options.IOStreams) *DrainOptions {
	return &DrainOptions{
		IOStreams:    streams,
		deployConfig: options.NewDeployOptions(),
		cliOpts:      options.NewCliOptions(),
	}
}

func NewCmdDrain(streams options.IOStreams) *cobra.Command {
	o := NewDrainOptions(streams)
	cmd := &cobra.Command{
		Use:                   "drain (--agent <agentIps>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "drain kubeclipper server or agent",
		Long:                  longDescription,
		Example:               drainExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs())
			if !o.preCheck() {
				return
			}
			utils.CheckErr(o.RunDrainFunc())
		},
	}

	o.cliOpts.AddFlags(cmd.Flags())
	cmd.Flags().StringSliceVar(&o.agents, "agent", o.agents, "drain agent node ip.")
	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "kcctl deploy config path")
	cmd.Flags().BoolVarP(&o.force, "force", "F", o.force, "force delete in used node.")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listNode(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("agent"))
	return cmd
}

func (c *DrainOptions) listNode(toComplete string) []string {
	utils.CheckErr(c.Complete())

	q := query.New()
	if !c.force { // without force flag only can delete node which is not in used
		q.LabelSelector = fmt.Sprintf("!%s", common.LabelNodeRole)
	}
	nodes, err := c.client.ListNodes(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	set := sets.NewString()
	for _, node := range nodes.Items {
		if strings.HasPrefix(node.Status.Ipv4DefaultIP, toComplete) {
			set.Insert(node.Status.Ipv4DefaultIP)
		}
	}

	return set.List()
}

func (c *DrainOptions) Complete() error {
	var err error

	// deploy config Complete
	if err = c.deployConfig.Complete(); err != nil {
		return err
	}

	// config Complete
	if err = c.cliOpts.Complete(); err != nil {
		return err
	}
	c.client, err = kc.FromConfig(c.cliOpts.ToRawConfig())

	return err
}

func (c *DrainOptions) ValidateArgs() error {
	if c.cliOpts.Config == "" {
		return errors.New("config path cannot be empty")
	}
	if c.deployConfig.Config == "" {
		return errors.New("deploy config path cannot be empty")
	}
	if len(c.agents) == 0 {
		return errors.New("--agent is required")
	}
	return nil
}

func (c *DrainOptions) preCheck() bool {
	c.agents = sets.NewString(c.agents...).List()

	for _, agent := range c.agents {
		if !c.deployConfig.Agents.Exists(agent) {
			logger.Errorf("agent %s is not in agent nodes", agent)
			return false
		}
	}

	if c.force {
		_, _ = c.IOStreams.Out.Write([]byte("force delete node which is in used maybe cause data inconsistency." +
			"are you sure this node are not in used or your really want to force delete used node?  Please input (yes/no)"))
		return utils.AskForConfirmation()
	}

	return true
}

func (c *DrainOptions) RunDrainFunc() error {
	return c.RunDrainNode()
}

func (c *DrainOptions) RunDrainNode() error {
	if err := c.runDrainServerNode(); err != nil {
		return fmt.Errorf("drain server node failed: %s", err.Error())
	}
	if err := c.runDrainAgentNode(); err != nil {
		return fmt.Errorf("drain agent node failed: %s", err.Error())
	}

	return nil
}

func (c *DrainOptions) runDrainAgentNode() error {
	for _, node := range c.agents {
		nodeInfo, err := c.checkAgentNode(node)
		if err != nil {
			return err
		}
		if err = c.agentFilesAndData(nodeInfo); err != nil {
			return err
		}
	}
	logger.Info("agent node drain completed. show command: 'kcctl get node'")
	return nil
}

func (c *DrainOptions) checkAgentNode(ip string) (*v1.Node, error) {
	q := query.New()
	nodeList, err := c.client.ListNodes(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil, err
	}
	for _, node := range nodeList.Items {
		// check the node already exists in database
		if node.Status.Ipv4DefaultIP == ip {
			// check the node already is used
			if _, ok := node.Labels[common.LabelNodeRole]; !ok || c.force {
				return &node, nil
			}
			return nil, fmt.Errorf("the node could not be draind. reason: %s is used by the cluster", ip)
		}
	}

	return nil, fmt.Errorf("the node could not be draind. reason: %s does not exist", ip)
}

func (c *DrainOptions) agentFilesAndData(node *v1.Node) error {
	// 1. remove agent
	cmdList := []string{
		"systemctl disable kc-agent --now", // 	// disable agent service
		"rm -rf /usr/local/bin/kubeclipper-agent /etc/kubeclipper-agent /usr/lib/systemd/system/kc-agent.service " + c.checkOplogDir(), // remove agent files
	}

	for _, v := range cmdList {
		_, err := sshutils.SSHCmd(c.deployConfig.SSHConfig, node.Status.Ipv4DefaultIP, v)
		if err != nil {
			return errors.WithMessagef(err, "run cmd %s on %s failed", v, node.Status.Ipv4DefaultIP)
		}
	}

	// 2. delete from etcd
	err := c.client.DeleteNode(context.TODO(), node.Name)
	if err != nil {
		return errors.WithMessagef(err, "delete node %s failed", node.Name)
	}

	// 	3.rewrite deploy config
	c.deployConfig.Agents.Delete(node.Status.Ipv4DefaultIP)
	err = c.deployConfig.Write()
	if err != nil {
		return errors.WithMessage(err, "rewrite deploy config")
	}
	return nil
}

func (c *DrainOptions) runDrainServerNode() error {
	return nil
}

// checkOplogDir return oplog dir, avoid removing illegal folders
func (c *DrainOptions) checkOplogDir() string {
	if !filepath.IsAbs(c.deployConfig.OpLog.Dir) || c.deployConfig.OpLog.Dir == "/" {
		return ""
	}
	return c.deployConfig.OpLog.Dir
}
