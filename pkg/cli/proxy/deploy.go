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

package proxy

import (
	"fmt"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	proxyConfig "github.com/kubeclipper/kubeclipper/pkg/proxy/config"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

const (
	longDescription = `
  Deploy kubeclipper proxy server.
  kubeclipper proxy as a transit server to proxy flow between kc-server and kc-agent.
  kc-agent can access kc-server by proxy server,kc-server can access kc-agent by proxy server too.`

	deployExample = `
  # Deploy proxy use default deploy config(~/.kc/deploy-config.yaml)
  # run proxy server on node 192.168.234.4, add 2 tunnels in 1 agent node.
  kcctl proxy --proxy 192.168.234.4 --tunnels 16443:192.168.234.5:6443 --tunnels 10022:192.168.234.5:22

  # Deploy proxy specify deploy config path.
  # run proxy server on node 192.168.234.4, add 4 tunnels in 2 agent node.
  kcctl proxy --deploy-config /root/.kc/deploy-config --proxy 192.168.234.4 --tunnels 16443:192.168.234.5:6443 --tunnels 10022:192.168.234.5:22 --tunnels 16443:192.168.234.6:6443 --tunnels 10022:192.168.234.6:22`
)

// kcctl proxy --pkg kc-test.tar.gz --proxy 192.168.20.212 --tunnels 16443:192.168.20.235:6443 --tunnels 10022:192.168.20.235:22 --v 5

type ProxyOptions struct {
	options.IOStreams
	deployConfig *options.DeployConfig
	config       *proxyConfig.Config
	pkg          string
	proxy        string
	tunnelStr    []string
	tunnels      []tunnel
}

func NewProxyOptions(streams options.IOStreams) *ProxyOptions {
	return &ProxyOptions{
		IOStreams:    streams,
		deployConfig: options.NewDeployOptions(),
	}
}

func NewCmdProxy(streams options.IOStreams) *cobra.Command {
	o := NewProxyOptions(streams)
	cmd := &cobra.Command{
		Use:                   "proxy [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Deploy Kubeclipper proxy server",
		Long:                  longDescription,
		Example:               deployExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs())

			utils.CheckErr(o.RunDeploy())
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringArrayVar(&o.tunnelStr, "tunnels", o.tunnelStr, "proxy tunnels.")
	cmd.Flags().StringVar(&o.proxy, "proxy", o.proxy, "kc proxy server")
	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "kcctl deploy config path")
	cmd.Flags().StringVar(&o.pkg, "pkg", o.deployConfig.Pkg, "kubeclipper package resource url (path or http url)")

	return cmd
}

func (d *ProxyOptions) Complete() error {
	var err error
	if err = d.deployConfig.Complete(); err != nil {
		return err
	}
	if d.pkg == "" {
		d.pkg = d.deployConfig.Pkg
	}
	d.tunnels, err = parseTunnel(d.tunnelStr)
	if err != nil {
		return err
	}
	logger.Info("tunnels:", d.tunnels)
	d.config = d.generateProxyConfig()
	logger.Info("c.config:", d.config)
	logger.Info("c.config len:", len(d.config.Config))
	return nil
}

func (d *ProxyOptions) ValidateArgs() error {
	if d.proxy == "" {
		return fmt.Errorf("--proxy must be specified")
	}

	for _, t := range d.tunnels {
		if !d.deployConfig.Agents.Exists(t.AgentIP) {
			return fmt.Errorf(`[%s] not in agent list`, t.AgentIP)
		}
	}
	return nil
}

func (d *ProxyOptions) RunDeploy() error {
	d.sendPackage()
	d.deployProxy()
	d.removeTempFile()
	fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
	return nil
}

func (d *ProxyOptions) checkProxy(name string) error {
	ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, d.proxy, fmt.Sprintf("systemctl --all --type service | grep -Fq %s", name))
	logger.V(2).Infof("exit code %d, err %v", ret.ExitCode, err)
	if err != nil {
		return err
	}
	if ret.ExitCode == 0 {
		err = fmt.Errorf("%s service exist, please clean old environment", name)
	}
	return err
}

func (d *ProxyOptions) sendPackage() {
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s", filepath.Join(config.DefaultPkgPath, "kc"),
		filepath.Join(config.DefaultPkgPath, path.Base(d.pkg)), config.DefaultPkgPath)
	cp := sshutils.WrapSh(fmt.Sprintf("cp -rf %s /usr/local/bin/", filepath.Join(config.DefaultPkgPath, "kc/bin/kubeclipper-proxy")))
	// rm -rf /root/kc && tar -xvf /root/kc/pkg/kc.tar -C ~/kc/pkg && /bin/bash -c 'cp -rf /root/kc/pkg/kc/bin/* /usr/local/bin/'
	hook := sshutils.Combine([]string{tar, cp})
	err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.pkg, []string{d.proxy}, config.DefaultPkgPath, nil, &hook)
	if err != nil {
		logger.Fatalf("sendPackage err:%s", err.Error())
	}
}

func (d *ProxyOptions) deployProxy() {
	configData, err := yaml.Marshal(d.config)
	if err != nil {
		logger.Fatalf("[deploy proxy] marshal proxy config failed due to %s", err.Error())
	}
	logger.Info("config data:", string(configData))

	cmdList := []string{
		sshutils.WrapEcho(config.KcProxyService, "/usr/lib/systemd/system/kc-proxy.service"),
		"mkdir -pv /etc/kubeclipper-proxy",
		sshutils.WrapEcho(string(configData), "/etc/kubeclipper-proxy/kubeclipper-proxy.yaml"),
		"systemctl daemon-reload && systemctl enable kc-proxy --now",
	}
	for _, cmd := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, d.proxy, cmd)
		if err != nil {
			logger.Fatalf("[%s]deploy etcd failed due to %s", d.proxy, err.Error())
		}
		if err = ret.Error(); err != nil {
			logger.Fatalf("[%s]deploy etcd failed due to %s", d.proxy, err.Error())
		}
	}
}

func (d *ProxyOptions) removeTempFile() {
	cmd := fmt.Sprintf("rm -rf %s/kc", config.DefaultPkgPath)
	ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, d.proxy, cmd)
	if err != nil {
		logger.Error("remote tmp file err:", err.Error())
	}
	if err = ret.Error(); err != nil {
		logger.Error("remote tmp file err:", err.Error())
	}
}
