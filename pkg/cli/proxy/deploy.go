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
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/deploy"
	proxyConfig "github.com/kubeclipper/kubeclipper/pkg/proxy/config"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
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
  # Deploy proxy,run proxy server on node 192.168.234.4, add 2 tunnels in 1 agent node.
  # this means proxy 192.168.234.4's 16443/10022 port to 192.168.234.5's 6443/22 port
  kcctl proxy --pkg /tmp/kc.tar.gz --proxy 192.168.234.4 --tunnels 16443:192.168.234.5:6443 --tunnels 10022:192.168.234.5:22

  # Deploy proxy,run proxy server on node 192.168.234.4, add 4 tunnels in 2 agent node.
  kcctl proxy --pkg /tmp/kc.tar.gz --proxy 192.168.234.4 --tunnels 16443:192.168.234.5:6443 --tunnels 10022:192.168.234.5:22 --tunnels 26443:192.168.234.6:6443 --tunnels 20022:192.168.234.6:22`
)

type ProxyOptions struct {
	options.IOStreams
	deployConfig *options.DeployConfig
	config       *proxyConfig.Config
	cliOpts      *options.CliOptions
	client       *kc.Client
	pkg          string
	proxy        string
	tunnelStr    []string
	tunnels      []tunnel
}

func NewProxyOptions(streams options.IOStreams) *ProxyOptions {
	return &ProxyOptions{
		cliOpts:      options.NewCliOptions(),
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
	cmd.Flags().StringVar(&o.pkg, "pkg", o.deployConfig.Pkg, "kubeclipper package resource url (path or http url)")

	utils.CheckErr(cmd.MarkFlagRequired("pkg"))
	utils.CheckErr(cmd.MarkFlagRequired("proxy"))
	utils.CheckErr(cmd.MarkFlagRequired("tunnels"))
	return cmd
}

func (d *ProxyOptions) Complete() error {
	var err error
	if d.pkg == "" {
		d.pkg = d.deployConfig.Pkg
	}
	d.tunnels, err = parseTunnel(d.tunnelStr)
	if err != nil {
		return err
	}
	if err = d.cliOpts.Complete(); err != nil {
		return err
	}
	d.client, err = kc.FromConfig(d.cliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	d.deployConfig, err = deploy.GetDeployConfig(context.Background(), d.client, true)
	if err != nil {
		return errors.WithMessage(err, "get online deploy-config failed")
	}

	d.config = d.generateProxyConfig()
	return nil
}

func (d *ProxyOptions) ValidateArgs() error {
	if d.deployConfig.SSHConfig.PkFile == "" && d.deployConfig.SSHConfig.Password == "" {
		return fmt.Errorf("one of pkfile or password must be specify,please config it in %s", d.deployConfig.Config)
	}
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

	if !sliceutil.HasString(d.deployConfig.Proxys, d.proxy) {
		d.deployConfig.Proxys = append(d.deployConfig.Proxys, d.proxy)
	}
	if err := deploy.UpdateDeployConfig(context.Background(), d.client, d.deployConfig, true); err != nil {
		return errors.WithMessage(err, "update online deploy-config failed")
	}
	fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
	return nil
}

func (d *ProxyOptions) sendPackage() {
	err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.pkg, []string{d.proxy}, config.DefaultPkgPath, nil, nil)
	if err != nil {
		logger.Fatalf("sendPackage err:%s", err.Error())
	}
	// if has dir in tar package,add --strip-components 1 to tar cmd.
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s", filepath.Join(config.DefaultPkgPath, "kc"),
		filepath.Join(config.DefaultPkgPath, path.Base(d.pkg)), config.DefaultPkgPath)
	tarWithStrip := fmt.Sprintf("%s --strip-components 1", tar)
	script := `ok=$(tar -tf %s |grep /|wc -l); if [ $ok -eq 0 ]; then %s; else %s; fi;`
	realScript := sshutils.WrapSh(fmt.Sprintf(script, filepath.Join(config.DefaultPkgPath, path.Base(d.pkg)), tar, tarWithStrip))
	cp := sshutils.WrapSh(fmt.Sprintf("cp -rf %s /usr/local/bin/", filepath.Join(config.DefaultPkgPath, "kubeclipper-proxy")))
	cmdList := []string{
		realScript,
		cp,
	}
	for _, v := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, d.proxy, v)
		if err != nil {
			logger.Fatalf("run %s err:%s", v, err.Error())
		}
		if err = ret.Error(); err != nil {
			logger.Fatalf("run %s err:%s", v, err.Error())
		}
	}

}

func (d *ProxyOptions) deployProxy() {
	configData, err := yaml.Marshal(d.config)
	if err != nil {
		logger.Fatalf("[deploy proxy] marshal proxy config failed due to %s", err.Error())
	}

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
