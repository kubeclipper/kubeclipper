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

package deploy

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"k8s.io/component-base/version"

	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/sethvargo/go-password/password"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	certutils "github.com/kubeclipper/kubeclipper/pkg/utils/certs"
)

const (
	longDescription = `
  Deploy Kubeclipper Platform from deploy-config.yaml or cmd flags.

  Kubeclipper Platform must have one kc-server node at lease, kc-server use etcd as db backend.
  So the number of kc-server nodes must be odd

  If you want to deploy kc-server and kc-agent on the same node, it is better to change etcd port configuration,
  in order to be able to deploy k8s on this node

  Now only support offline install, so the --pkg parameter must be valid`
	deployExample = `
  # Deploy All-In-One use local host, etcd port will be set automatically. (client-12379 | peer-12380 | metrics-12381)
  kcctl deploy

  # Deploy AIO env and change etcd port
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --passwd 'YOUR-SSH-PASSWORD' --etcd-port 12379 --etcd-peer-port 12380 --etcd-metric-port 12381

  # Deploy HA env
  kcctl deploy --server 192.168.234.3,192.168.234.4,192.168.234.5 --agent 192.168.234.3 --passwd 'YOUR-SSH-PASSWORD' --etcd-port 12379 --etcd-peer-port 12380 --etcd-metric-port 12381

  # Deploy env use SSH key instead of password
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --pk-file ~/.ssh/id_rsa --pkg kc-minimal.tar.gz

  # Deploy env use remove http/https resource server
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz

  # Deploy env with many agent node in same region.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:192.168.10.123,192.168.10.124  --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz

  # Deploy env with many agent node in different region.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:1.1.1.1,1.1.1.2 --agent us-west-2:1.1.1.3 --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz

  # Deploy env with many agent node which has orderly ip.
  # this will add 10 agent,1.1.1.1, 1.1.1.2, ... 1.1.1.10.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:1.1.1.1-1.1.1.10 --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz
  
  # Deploy env with many agent nodes and specify ip detect method for these nodes
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3,192.168.234.4 --ip-detect=interface=eth0 --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz

  # Deploy env with many agent nodes and specify node ip detect method for these nodes, used for routing between nodes in the kubernetes cluster
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3,192.168.234.4 --node-ip-detect=interface=eth1 --pk-file ~/.ssh/id_rsa --pkg https://oss.kubeclipper.io/release/v1.4.0/kc-amd64.tar.gz

  # Deploy from config.
  kcctl deploy --deploy-config deploy-config.yaml
  # Deploy and config fip to agent node.
  kcctl deploy --server 172.20.149.198 --agent us-west-1:10.0.0.10 --agent us-west-2:20.0.0.11 --fip 10.0.0.10:172.20.149.199 --fip 20.0.0.11:172.20.149.200

  Please read 'kcctl deploy -h' get more deploy flags`
	defaultPkg = "https://oss.kubeclipper.io/release/%s/kc-%s.tar.gz"
)

type DeployOptions struct {
	options.IOStreams
	deployConfig *options.DeployConfig
	allNodes     []string
	servers      map[string]string
	agents       []string // user input's agents,maybe with region,need to parse.
	fips         []string // ip:fip
	aio          bool
}

func NewDeployOptions(streams options.IOStreams) *DeployOptions {
	return &DeployOptions{
		IOStreams:    streams,
		deployConfig: options.NewDeployOptions(),
		servers:      make(map[string]string),
	}
}

func NewCmdDeploy(streams options.IOStreams) *cobra.Command {
	o := NewDeployOptions(streams)
	cmd := &cobra.Command{
		Use:                   "deploy (-c CONFIG | [flags])",
		DisableFlagsInUseLine: true,
		Short:                 "Deploy Kubeclipper platform",
		Long:                  longDescription,
		Example:               deployExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs())
			o.preRun()
			if !o.preCheck() {
				return
			}
			utils.CheckErr(o.RunDeploy())
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringArrayVar(&o.agents, "agent", o.agents, "Kc agent region and ips.")
	cmd.Flags().StringArrayVar(&o.fips, "float-ip", o.fips, "Kc agent ip and float ip.")
	cmd.Flags().IntVar(&o.deployConfig.AuthenticationOpts.AuthenticateRateLimiterMaxTries, "authenticate-rate-limiter-max-retries", o.deployConfig.AuthenticationOpts.AuthenticateRateLimiterMaxTries, "maximum number of retry times within the valid period")
	cmd.Flags().DurationVar(&o.deployConfig.AuthenticationOpts.AuthenticateRateLimiterDuration, "authenticate-rate-limiter-duration", o.deployConfig.AuthenticationOpts.AuthenticateRateLimiterDuration, "specifies the lock duration of the user")
	cmd.Flags().DurationVar(&o.deployConfig.AuthenticationOpts.LoginHistoryRetentionPeriod, "login-history-retention-period", o.deployConfig.AuthenticationOpts.LoginHistoryRetentionPeriod, "login-history-retention-period defines how long login history should be kept.")
	cmd.Flags().IntVar(&o.deployConfig.AuthenticationOpts.LoginHistoryMaximumEntries, "login-history-maximum-entries", o.deployConfig.AuthenticationOpts.LoginHistoryMaximumEntries, "login-history-maximum-entries defines how many entries of login history should be kept.")
	cmd.Flags().StringVar(&o.deployConfig.AuthenticationOpts.InitialPassword, "initial-password", o.deployConfig.AuthenticationOpts.InitialPassword, "admin user password")
	o.deployConfig.AddFlags(cmd.Flags())
	o.deployConfig.AuditOpts.AddFlags(cmd.Flags())

	cmd.AddCommand(NewCmdDeployConfig(o))

	return cmd
}

func (d *DeployOptions) Complete() error {
	var err error
	if err = d.deployConfig.Complete(); err != nil {
		return err
	}
	if d.deployConfig.Pkg == "" {
		v := os.Getenv("KC_VERSION")
		var ok bool
		if v == "" {
			v, ok = strutil.ParseGitDescribeInfo(version.Get().GitVersion)
			if !ok {
				v = "v1.5.0"
			}
		}
		d.deployConfig.Pkg = fmt.Sprintf(defaultPkg, v, runtime.GOARCH)
	}

	// if both the server and agent are empty, set the all-in-one environment
	if d.deployConfig.ServerIPs == nil && d.agents == nil {
		d.aio = true

		ip, err := netutil.GetDefaultIP(true, d.deployConfig.IPDetect)
		if err != nil {
			return err
		}
		// set the local host as the kc server and agent
		d.deployConfig.ServerIPs = []string{ip.String()}
		d.agents = []string{ip.String()}
	}

	// if specify config，ignore flags.
	if d.deployConfig.Config == "" {
		if d.deployConfig.Agents, err = BuildAgent(d.agents, d.fips, d.deployConfig.DefaultRegion); err != nil {
			return err
		}
	}

	d.allNodes = sets.NewString().
		Insert(d.deployConfig.ServerIPs...).
		Insert(d.deployConfig.Agents.ListIP()...).
		List()

	if !d.deployConfig.MQ.External {
		d.deployConfig.MQ.IPs = d.deployConfig.ServerIPs // internal mq use server ips as mq ips
		if d.deployConfig.MQ.TLS {                       // fill default tls file path
			d.deployConfig.MQ.CA = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultCaPath, fmt.Sprintf("%s.crt", options.Ca))
			d.deployConfig.MQ.ClientCert = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.crt", options.NatsIOClient))
			d.deployConfig.MQ.ClientKey = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.key", options.NatsIOClient))
		}
	}

	if d.deployConfig.NodeIPDetect == "" {
		logger.Infof("node-ip-detect inherits from ip-detect: %s", d.deployConfig.IPDetect)
		d.deployConfig.NodeIPDetect = d.deployConfig.IPDetect
	}

	if d.aio {
		logger.Infof("run in aio mode.")
	}

	return nil
}

func (d *DeployOptions) ValidateArgs() error {
	if errs := d.deployConfig.AuditOpts.Validate(); len(errs) != 0 {
		return fmt.Errorf("%d errors in audit occured: %v", len(errs), errs)
	}
	if errs := d.deployConfig.AuthenticationOpts.Validate(); len(errs) != 0 {
		return fmt.Errorf("%d errors in AuthenticationOpts occured: %v", len(errs), errs)
	}

	if d.deployConfig.IPDetect != "" && !autodetection.CheckMethod(d.deployConfig.IPDetect) {
		return fmt.Errorf("invalid ip detect method,suppot [first-found,interface=xxx,cidr=xxx] now")
	}
	if d.deployConfig.NodeIPDetect != "" && !autodetection.CheckMethod(d.deployConfig.NodeIPDetect) {
		return fmt.Errorf("invalid node ip detect method,suppot [first-found,interface=xxx,cidr=xxx] now")
	}
	if d.deployConfig.Pkg == "" {
		return fmt.Errorf("--pkg must be specified")
	}
	if !d.aio && d.deployConfig.SSHConfig.PkFile == "" && d.deployConfig.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
	}
	if d.deployConfig.SSHConfig.Port <= 0 {
		return fmt.Errorf("ssh connection port must be a positive number")
	}
	if len(d.deployConfig.ServerIPs) == 0 {
		return fmt.Errorf("must specify at least one server")
	}
	if len(d.deployConfig.ServerIPs)%2 == 0 {
		return fmt.Errorf("the number of servers must be odd")
	}
	if d.deployConfig.MQ.External {
		if len(d.deployConfig.MQ.IPs) == 0 {
			return fmt.Errorf("the ips of the external mq cannot be empty")
		}
		if d.deployConfig.MQ.Port == 0 {
			return fmt.Errorf("the port of the external mq cannot be empty")
		}
		if d.deployConfig.MQ.TLS {
			if d.deployConfig.MQ.CA == "" || d.deployConfig.MQ.ClientCert == "" || d.deployConfig.MQ.ClientKey == "" {
				return fmt.Errorf("mq tls: the mq-external-ca/mq-external-cert/mq-external-key of the external mq cannot be empty")
			}
			if !(filepath.IsAbs(d.deployConfig.MQ.CA) || filepath.IsAbs(d.deployConfig.MQ.ClientCert) || filepath.IsAbs(d.deployConfig.MQ.ClientKey)) {
				return fmt.Errorf("mq tls: ca/cert/key file must be an absolute path")
			}
		}
	}
	return nil
}

func (d *DeployOptions) preRun() {
	for _, sip := range d.deployConfig.ServerIPs {
		hostname, err := sshutils.GetRemoteHostName(d.deployConfig.SSHConfig, sip)
		if err != nil {
			logger.Fatalf("get remote hostname failed,err:%v", err)
		}
		d.servers[sip] = hostname
	}
	res, _ := password.Generate(24, 5, 0, false, true)
	d.deployConfig.JWTSecret = res
	if !d.deployConfig.MQ.External {
		res, _ = password.Generate(24, 0, 0, false, true)
		d.deployConfig.MQ.Secret = res
	}
	d.dumpConfig()
}

type precheckFunc func(sshConfig *sshutils.SSH, host string) error

var (
	precheckKcEtcdFunc                = generateCommonPreCheckFunc("kc-etcd")
	precheckKcServerFunc              = generateCommonPreCheckFunc("kc-server")
	precheckKcAgentFunc               = generateCommonPreCheckFunc("kc-agent")
	precheckNtpFunc      precheckFunc = func(sshConfig *sshutils.SSH, host string) error {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, "systemctl --all --type service --state running | grep -e chrony -e ntp|wc -l")
		if err != nil {
			return err
		}
		if ret.StdoutToString("") == "0" {
			err = fmt.Errorf("chronyd or ntpd service not running, may cause service internal error")
		}
		return err
	}
)

func generateCommonPreCheckFunc(name string) precheckFunc {
	return func(sshConfig *sshutils.SSH, host string) error {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, fmt.Sprintf("systemctl --all --type service | grep %s | wc -l", name))
		logger.V(2).Infof("exit code %d, err %v", ret.ExitCode, err)
		if err != nil {
			return err
		}
		if ret.StdoutToString("") != "0" {
			err = fmt.Errorf("%s service exist, please clean old environment", name)
		}
		return err
	}
}

func (d *DeployOptions) precheckService(name string, nodes []string, fn precheckFunc) bool {
	logger.Infof("============>%s PRECHECK ...", name)
	nodeErrChan := make(chan error, len(nodes))
	defer close(nodeErrChan)
	wg := sync.WaitGroup{}
	for _, node := range nodes {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			if err := fn(d.deployConfig.SSHConfig, host); err != nil {
				nodeErrChan <- err
			}
		}(node)
	}
	wg.Wait()
	if len(nodeErrChan) == 0 {
		logger.Infof("============>%s PRECHECK OK!", name)
		return true
	}
	for i := 0; i < len(nodeErrChan); i++ {
		logger.Warn(<-nodeErrChan)
	}
	logger.Errorf("===========>%s PRECHECK FAILED!", name)
	if options.AssumeYes {
		return true
	}
	_, _ = d.IOStreams.Out.Write([]byte("Ignore this error, still install? Please input (yes/no)"))
	return utils.AskForConfirmation()
}

func (d *DeployOptions) precheckTimeLag() bool {
	logger.Infof("============>TIME-LAG PRECHECK ...")
	nodeErrChan := make(chan struct{}, len(d.allNodes))
	defer close(nodeErrChan)
	wg := sync.WaitGroup{}
	now := time.Now()
	logger.Infof("BaseLine Time: %s", now.Format(time.RFC3339))
	for _, node := range d.allNodes {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			ret, err := sshutils.SSHCmd(d.deployConfig.SSHConfig, host, "date +%s")
			if err != nil {
				logger.Fatal("get timestamp failed", err)
			}
			output := ret.StdoutToString("")
			ts, err := strconv.ParseInt(output, 10, 64)
			if err != nil {
				logger.Fatal("get timestamp failed", err)
			}
			t := time.Unix(ts, 0)
			diff := t.Sub(now).Seconds()
			logger.Infof("[%s] %v seconds", host, diff)
			if math.Abs(diff) > float64(5) {
				nodeErrChan <- struct{}{}
			}
		}(node)
	}
	wg.Wait()
	if len(nodeErrChan) == 0 {
		logger.Infof("all nodes time lag less then 5 seconds")
		logger.Infof("============>TIME-LAG PRECHECK OK!")
		return true
	}
	logger.Errorf("===========>TIME-LAG PRECHECK FAILED!")
	if options.AssumeYes {
		return true
	}
	_, _ = d.IOStreams.Out.Write([]byte("Ignore this error, still install? Please input (yes/no)"))
	return utils.AskForConfirmation()
}

func (d *DeployOptions) preCheck() bool {
	if !d.precheckService("kc-etcd", d.deployConfig.ServerIPs, precheckKcEtcdFunc) {
		return false
	}
	if !d.precheckService("kc-server", d.deployConfig.ServerIPs, precheckKcServerFunc) {
		return false
	}
	if !d.precheckService("kc-agent", d.allNodes, precheckKcAgentFunc) {
		return false
	}
	if !d.precheckTimeLag() {
		return false
	}
	if !d.precheckService("NTP", d.allNodes, precheckNtpFunc) {
		return false
	}
	if !sudo.PreCheck("sudo", d.deployConfig.SSHConfig, d.IOStreams, d.allNodes) {
		return false
	}
	if !sudo.MultiNIC("ipDetect", d.deployConfig.SSHConfig, d.IOStreams, d.deployConfig.Agents.ListIP(), d.deployConfig.IPDetect) {
		return false
	}
	return true
}

func (d *DeployOptions) RunDeploy() error {
	if err := d.generateAndSendCerts(); err != nil {
		return err
	}
	logger.Infof("------ Send packages ------")
	d.sendPackage()
	logger.Infof("------ Install kc-etcd ------")
	d.deployEtcd()
	// TODO: add check etcd status instead of time.sleep
	time.Sleep(5 * time.Second)
	logger.Infof("------ Install kc-server ------")
	d.deployKcServer()
	time.Sleep(5 * time.Second)
	logger.Infof("------ Install kc-agent ------")
	d.deployKcAgent()
	logger.Infof("------ Install kc-console ------")
	d.deployKcConsole()
	logger.Infof("------ Delete intermediate files ------")
	d.removeTempFile()
	logger.Infof("------ Dump configs ------")
	d.dumpConfig()
	logger.Infof("------ Upload configs ------")
	d.uploadConfig()
	fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
	return nil
}

func (d *DeployOptions) sendPackage() {
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s", filepath.Join(config.DefaultPkgPath, "kc"),
		filepath.Join(config.DefaultPkgPath, path.Base(d.deployConfig.Pkg)), config.DefaultPkgPath)
	cp := sshutils.WrapSh(fmt.Sprintf("cp -rf %s /usr/local/bin/", filepath.Join(config.DefaultPkgPath, "kc/bin/*")))
	mkdir := "mkdir -p /usr/lib/systemd/system"
	// rm -rf /root/kc && tar -xvf /root/kc/pkg/kc.tar -C ~/kc/pkg && /bin/bash -c 'cp -rf /root/kc/pkg/kc/bin/* /usr/local/bin/' && mkdir -p /usr/lib/systemd/system
	hook := sshutils.Combine([]string{tar, cp, mkdir})
	err := utils.SendPackage(d.deployConfig.SSHConfig, d.deployConfig.Pkg, d.allNodes, config.DefaultPkgPath, nil, &hook)
	if err != nil {
		logger.Fatalf("sendPackage err:%s", err.Error())
	}
}

func (d *DeployOptions) generateAndSendCerts() error {
	var altNames []string
	for _, name := range d.servers {
		altNames = append(altNames, name)
	}
	cas := caList()
	certs := make([]certutils.Config, 0)

	kcctlCommonNameUsages := make(map[string][]x509.ExtKeyUsage)
	kcctlCommonNameUsages[options.AdminKcctlCert] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	kcctlCert := clientCertList(options.DefaultKcctlPKIPath, options.Ca, append(altNames, d.deployConfig.Agents.ListIP()...), []string{user.KCCTL}, kcctlCommonNameUsages)
	certs = append(certs, kcctlCert...)

	etcdCommonNameUsages := make(map[string][]x509.ExtKeyUsage)
	etcdCommonNameUsages[options.EtcdServer] = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdPeer] = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdKcClient] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdHealthCheck] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	etcdCert := certList(options.DefaultEtcdPKIPath, options.Ca, append(altNames, d.deployConfig.ServerIPs...), etcdCommonNameUsages)
	certs = append(certs, etcdCert...)

	var natsCert []certutils.Config
	if !d.deployConfig.MQ.External && d.deployConfig.MQ.TLS {
		natsCommonNameUsages := make(map[string][]x509.ExtKeyUsage)
		natsCommonNameUsages[options.NatsIOClient] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		natsCommonNameUsages[options.NatsIOServer] = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		natsCert = certList(options.DefaultNatsPKIPath, options.Ca, append(append(altNames, d.deployConfig.ServerIPs...), options.NatsAltNameProxy), natsCommonNameUsages)
		certs = append(certs, natsCert...)
	}
	var kcCerts []certutils.Config
	if d.deployConfig.TLS {
		nameUsages := map[string][]x509.ExtKeyUsage{
			options.KCServer: {x509.ExtKeyUsageServerAuth},
		}
		names := append(altNames, d.deployConfig.ServerIPs...)
		names = append(names, options.KCServerAltName)
		kcCerts = certList(options.DefaultKCPKIPath, options.Ca, names, nameUsages)
		certs = append(certs, kcCerts...)
	}

	CACerts := map[string]*x509.Certificate{}
	CAKeys := map[string]crypto.Signer{}

	for _, ca := range cas {
		caCert, caKey, err := certutils.NewCaCertAndKey(ca)
		if err != nil {
			return err
		}
		CACerts[ca.CommonName] = caCert
		CAKeys[ca.CommonName] = caKey

		err = certutils.WriteCertAndKey(ca.Path, ca.BaseName, caCert, caKey)
		if err != nil {
			return err
		}
	}

	for _, cert := range certs {
		caCert, ok := CACerts[cert.CAName]
		if !ok {
			return fmt.Errorf("root ca cert not found %s", cert.CAName)
		}
		caKey, ok := CAKeys[cert.CAName]
		if !ok {
			return fmt.Errorf("root ca key not found %s", cert.CAName)
		}

		Cert, Key, err := certutils.NewCaCertAndKeyFromRoot(cert, caCert, caKey)
		if err != nil {
			return err
		}
		err = certutils.WriteCertAndKey(cert.Path, cert.BaseName, Cert, Key)
		if err != nil {
			return err
		}
	}

	if err := d.sendCertAndKey(cas, options.DefaultCaPath); err != nil {
		return err
	}
	//if err := d.sendClientCertAndKey(cas, true); err != nil {
	//	return err
	//}
	//if err := d.sendClientCertAndKey(kcctlCert, false); err != nil {
	//	return err
	//}

	if err := d.sendCertAndKey(etcdCert, options.DefaultEtcdPKIPath); err != nil {
		return err
	}

	if d.deployConfig.MQ.TLS {
		if !d.deployConfig.MQ.External {
			err := d.sendCertAndKey(natsCert, options.DefaultNatsPKIPath)
			if err != nil {
				return err
			}
			if err := d.sendAgentCertAndKey(cas, options.DefaultCaPath); err != nil {
				return err
			}
			err = d.sendAgentCertAndKey(natsCert, options.DefaultNatsPKIPath)
			if err != nil {
				return err
			}
		} else {
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.CA,
				d.deployConfig.ServerIPs, filepath.Dir(d.deployConfig.MQ.CA), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientCert,
				d.deployConfig.ServerIPs, filepath.Dir(d.deployConfig.MQ.ClientCert), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientKey,
				d.deployConfig.ServerIPs, filepath.Dir(d.deployConfig.MQ.ClientKey), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.CA,
				d.deployConfig.Agents.ListIP(), filepath.Dir(d.deployConfig.MQ.CA), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientCert,
				d.deployConfig.Agents.ListIP(), filepath.Dir(d.deployConfig.MQ.ClientCert), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientKey,
				d.deployConfig.Agents.ListIP(), filepath.Dir(d.deployConfig.MQ.ClientKey), nil, nil); err != nil {
				return err
			}
		}
	}
	if d.deployConfig.TLS {
		err := d.sendConsoleCert(cas, options.DefaultCaPath)
		if err != nil {
			return err
		}
		err = d.sendCertAndKey(kcCerts, options.DefaultKCPKIPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) deployEtcd() {
	for _, host := range d.deployConfig.ServerIPs {
		data := d.getEtcdTemplateContent(host)
		cmd := sshutils.WrapEcho(data, "/usr/lib/systemd/system/kc-etcd.service") +
			" && systemctl daemon-reload && systemctl enable kc-etcd --now"
		ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, host, cmd)
		if err != nil {
			logger.Fatalf("[%s]deploy etcd failed due to %s", host, err.Error())
		}
		if err = ret.Error(); err != nil {
			logger.Fatalf("[%s]deploy etcd failed due to %s", host, err.Error())
		}
	}
}

func (d *DeployOptions) getEtcdTemplateContent(ip string) string {
	tmpl, err := template.New("text").Parse(config.EtcdServiceTmpl)
	if err != nil {
		logger.Fatalf("template parse failed: %s", err.Error())
	}
	isFloatIP, _ := sshutils.IsFloatIP(d.deployConfig.SSHConfig, ip)
	var initialCluster []string
	for k, v := range d.servers {
		initialCluster = append(initialCluster, fmt.Sprintf("%s=https://%s:%d", v, k, d.deployConfig.EtcdConfig.PeerPort))
	}
	var data = make(map[string]interface{})
	data["NodeName"] = d.servers[ip]
	data["AdvertiseAddress"] = fmt.Sprintf("%s:%d", ip, d.deployConfig.EtcdConfig.ClientPort)
	data["ServerCertPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.crt", options.EtcdServer))
	data["DataDIR"] = d.deployConfig.EtcdConfig.DataDir
	data["PeerAddress"] = fmt.Sprintf("%s:%d", ip, d.deployConfig.EtcdConfig.PeerPort)
	data["InitialCluster"] = strings.Join(initialCluster, ",")
	data["ClusterToken"] = "kc-etcd-cluster"
	data["ServerCertKeyPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.key", options.EtcdServer))
	if isFloatIP {
		// if user specify a float ip,we replace to listen 0.0.0.0
		data["PeerURLs"] = fmt.Sprintf("https://0.0.0.0:%d", d.deployConfig.EtcdConfig.PeerPort)
		data["ClientURLs"] = fmt.Sprintf("https://0.0.0.0:%d", d.deployConfig.EtcdConfig.ClientPort)
	} else {
		data["ClientURLs"] = fmt.Sprintf("https://127.0.0.1:%d,https://%s:%d", d.deployConfig.EtcdConfig.ClientPort, ip, d.deployConfig.EtcdConfig.ClientPort)
		data["PeerURLs"] = fmt.Sprintf("https://%s:%d", ip, d.deployConfig.EtcdConfig.PeerPort)
	}
	data["MetricsURLs"] = fmt.Sprintf("http://127.0.0.1:%d", d.deployConfig.EtcdConfig.MetricsPort)
	data["PeerCertPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.crt", options.EtcdPeer))
	data["PeerCertKeyPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.key", options.EtcdPeer))
	data["CaPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultCaPath, fmt.Sprintf("%s.crt", options.Ca))
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (d *DeployOptions) getKcConsoleTemplateContent() string {
	tmpl, err := template.New("text").Parse(config.KcCaddyTmpl)
	if err != nil {
		logger.Fatalf("template parse failed: %s", err.Error())
	}
	var serverUpstream string
	scheme := "http"
	if d.deployConfig.TLS {
		scheme = "https"
	}
	for k := range d.servers {
		serverUpstream = serverUpstream + fmt.Sprintf(" %s://%s:%d",
			scheme, k, d.deployConfig.ServerPort)
	}
	var data = make(map[string]interface{})
	data["TLS"] = d.deployConfig.TLS
	data["TLSServerName"] = options.KCServerAltName
	data["CACert"] = filepath.Join(options.DefaultKcConsoleConfigPath, options.DefaultCaPath, "ca.crt")
	data["ConsolePort"] = d.deployConfig.ConsolePort
	data["ServerUpstream"] = serverUpstream
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (d *DeployOptions) deployKcServer() {
	cmdList := []string{
		"mkdir -pv /etc/kubeclipper-server",
		sshutils.WrapEcho(config.KcServerService, "/usr/lib/systemd/system/kc-server.service"),
		fmt.Sprintf("mkdir -pv %s/kc", d.deployConfig.StaticServerPath),
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/resource/* %s/", config.DefaultPkgPath, d.deployConfig.StaticServerPath)),
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/bin/* %s/kc/", config.DefaultPkgPath, d.deployConfig.StaticServerPath)),
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(d.deployConfig.SSHConfig, d.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Fatalf("deploy kc server failed due to %s", err.Error())
		}
	}

	for _, host := range d.deployConfig.ServerIPs {
		data, err := d.deployConfig.GetKcServerConfigTemplateContent(host)
		if err != nil {
			logger.Fatal(err)
		}
		cmd := sshutils.WrapEcho(data, "/etc/kubeclipper-server/kubeclipper-server.yaml") +
			"&& systemctl daemon-reload && systemctl enable kc-server --now"
		ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, host, cmd)
		if err != nil {
			logger.Fatalf("[%s]deploy kc server failed due to %s", host, err.Error())
		}
		if err = ret.Error(); err != nil {
			logger.Fatalf("[%s]deploy kc server failed due to %s", host, err.Error())
		}

		// wait kc-server start
		ctx, cancel := context.WithTimeout(context.Background(), d.deployConfig.KCServerHealthCheckTimeout)
		defer cancel()
		err = retryFunc(ctx, 3*time.Second, "waitServiceRunning", host, d.waitServerRunning)
		if err != nil {
			logger.Fatalf("kc server status is not ready: %s", err.Error())
		}
	}
}

func (d *DeployOptions) waitServerRunning(host string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	addr := fmt.Sprintf("http://%s:%v/healthz", host, d.deployConfig.ServerPort)
	if d.deployConfig.TLS {
		addr = fmt.Sprintf("https://%s:%v/healthz", host, d.deployConfig.ServerPort)
	}
	resp, err := client.Get(addr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) == "ok" {
		return nil
	}

	return fmt.Errorf("%s kc server status is unhealth", host)
}

func retryFunc(ctx context.Context, intervalTime time.Duration, funcName, host string, fn func(host string) error) error {
	for {
		select {
		case <-ctx.Done():
			logger.Warnf("retry function '%s' timeout...", funcName)
			return ctx.Err()
		case <-time.After(intervalTime):
			err := fn(host)
			if err == nil {
				return nil
			}
			logger.Infof("function '%s' running error: %s. about to enter retry", funcName, err.Error())
		}
	}
}

func (d *DeployOptions) deployKcConsole() {
	data := d.getKcConsoleTemplateContent()

	cmdList := []string{
		fmt.Sprintf("mkdir -pv /etc/kc-console && cp -rf %s/kc/kc-console /etc/kc-console/dist", config.DefaultPkgPath),
		sshutils.WrapEcho(config.KcConsoleServiceTmpl, "/usr/lib/systemd/system/kc-console.service"),
		sshutils.WrapEcho(data, "/etc/kc-console/Caddyfile") + " && systemctl daemon-reload && systemctl enable kc-console --now",
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(d.deployConfig.SSHConfig, d.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Fatalf("deploy kc console failed due to %s", err.Error())
		}
	}
}

func (d *DeployOptions) deployKcAgent() {
	for agent := range d.deployConfig.Agents {
		metadata := d.deployConfig.Agents[agent]
		metadata.AgentID = uuid.New().String()
		d.deployConfig.Agents[agent] = metadata
		agentConfig, err := d.deployConfig.GetKcAgentConfigTemplateContent(metadata)
		if err != nil {
			logger.Fatal(err)
		}
		cmdList := []string{
			sshutils.WrapEcho(config.KcAgentService, "/usr/lib/systemd/system/kc-agent.service"),
			"mkdir -pv /etc/kubeclipper-agent",
			sshutils.WrapEcho(agentConfig, "/etc/kubeclipper-agent/kubeclipper-agent.yaml"),
			"systemctl daemon-reload && systemctl enable kc-agent --now",
		}
		for _, cmd := range cmdList {
			ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, agent, cmd)
			if err != nil {
				logger.Fatalf("[%s]deploy kc agent failed due to %s", agent, err.Error())
			}
			if err = ret.Error(); err != nil {
				logger.Fatalf("[%s]deploy kc agent failed due to %s", agent, err.Error())
			}
		}
	}
}

func (d *DeployOptions) removeTempFile() {
	cmdList := []string{
		fmt.Sprintf("rm -rf %s/kc", config.DefaultPkgPath),
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(d.deployConfig.SSHConfig, d.allNodes, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Errorf("remove temp file filed due to %s", err.Error())
		}
	}
}

func (d *DeployOptions) dumpConfig() {
	if err := d.deployConfig.Write(); err != nil {
		logger.Fatal(err)
	}
	logger.V(2).Infof("dump config to %s", options.DefaultDeployConfigPath)
}

func (d *DeployOptions) sendCertAndKey(contents []certutils.Config, pki string) error {
	for _, content := range contents {
		err := utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".key"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcServerConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcServerConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) sendAgentCertAndKey(contents []certutils.Config, pki string) error {
	for _, content := range contents {
		err := utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".key"),
			d.deployConfig.Agents.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.Agents.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) sendConsoleCert(contents []certutils.Config, pki string) error {
	for _, content := range contents {
		err := utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcConsoleConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) uploadConfig() {
	scheme := "http"
	if d.deployConfig.TLS {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s:%d", scheme, d.deployConfig.ServerIPs[0],
		d.deployConfig.ServerPort)
	cfg := config.Config{
		Servers: map[string]*config.Server{
			"default-cert": {
				Server:               host,
				TLSServerName:        options.KCServerAltName,
				CertificateAuthority: path.Join(homedir.HomeDir(), options.DefaultPath, options.DefaultCaPath, options.Ca+".crt"),
			},
		},
		AuthInfos: map[string]*config.AuthInfo{
			"kcctl-admin": {
				ClientCertificate: path.Join(homedir.HomeDir(), options.DefaultPath, options.DefaultKcctlPKIPath, options.AdminKcctlCert+".crt"),
				ClientKey:         path.Join(homedir.HomeDir(), options.DefaultPath, options.DefaultKcctlPKIPath, options.AdminKcctlCert+".key"),
			},
		},
		CurrentContext: fmt.Sprintf("%s@default-cert", "kcctl-admin"),
		Contexts: map[string]*config.Context{
			fmt.Sprintf("%s@default-cert", "kcctl-admin"): {
				AuthInfo: "kcctl-admin",
				Server:   "default-cert",
			},
		},
	}
	c, err := kc.FromConfig(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	uploadDeployConfig(c, d.deployConfig)
	uploadCerts(c)
	if err = cfg.Dump(); err != nil {
		logger.Fatal(err)
	}
	if err = d.sendDefaultAdminConf(); err != nil {
		logger.Fatal(err)
	}
}

func (d *DeployOptions) sendDefaultAdminConf() error {
	afterHook := fmt.Sprintf("mv %s %s/admin.conf", path.Join(options.DefaultKcServerConfigPath, options.DefaultConfig), options.DefaultKcServerConfigPath)
	err := utils.SendPackage(d.deployConfig.SSHConfig,
		options.DefaultConfigPath,
		d.deployConfig.ServerIPs,
		options.DefaultKcServerConfigPath, nil, &afterHook)
	return err
}

func uploadDeployConfig(client *kc.Client, deployConfig *options.DeployConfig) {
	dcData, err := yaml.Marshal(deployConfig)
	if err != nil {
		logger.Fatal(err)
	}
	dc := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindConfigMap,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constatns.DeployConfigConfigMapName,
		},
		Data: map[string]string{
			constatns.DeployConfigConfigMapKey: string(dcData),
		},
	}
	_, err = client.CreateConfigMap(context.TODO(), dc)
	if err != nil {
		logger.Fatal(err)
	}
}

func uploadCerts(client *kc.Client) {
	caPath := filepath.Join(options.HomeDIR, options.DefaultPath, options.DefaultCaPath)
	cacrt, err := os.ReadFile(fmt.Sprintf("%s/ca.crt", caPath))
	if err != nil {
		logger.Fatal(err)
	}
	cakey, err := os.ReadFile(fmt.Sprintf("%s/ca.key", caPath))
	if err != nil {
		logger.Fatal(err)
	}
	cacm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindConfigMap,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constatns.KcCertsConfigMapName,
		},
		Data: map[string]string{
			"ca.crt": base64.StdEncoding.EncodeToString(cacrt),
			"ca.key": base64.StdEncoding.EncodeToString(cakey),
		},
	}
	_, err = client.CreateConfigMap(context.TODO(), cacm)
	if err != nil {
		logger.Fatal(err)
	}

	etcdPath := filepath.Join(options.HomeDIR, options.DefaultPath, options.DefaultEtcdPKIPath)
	etcdcrt, err := os.ReadFile(fmt.Sprintf("%s/etcd.crt", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdkey, err := os.ReadFile(fmt.Sprintf("%s/etcd.key", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdpeercrt, err := os.ReadFile(fmt.Sprintf("%s/etcd-peer.crt", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdpeerkey, err := os.ReadFile(fmt.Sprintf("%s/etcd-peer.key", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdclientcrt, err := os.ReadFile(fmt.Sprintf("%s/kc-server-etcd-client.crt", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdclientkey, err := os.ReadFile(fmt.Sprintf("%s/kc-server-etcd-client.key", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdhealthcheckcrt, err := os.ReadFile(fmt.Sprintf("%s/kube-etcd-healthcheck-client.crt", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdhealthcheckkey, err := os.ReadFile(fmt.Sprintf("%s/kube-etcd-healthcheck-client.crt", etcdPath))
	if err != nil {
		logger.Fatal(err)
	}
	etcdcm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindConfigMap,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constatns.KcEtcdCertsConfigMapName,
		},
		Data: map[string]string{
			"etcd.crt":                         base64.StdEncoding.EncodeToString(etcdcrt),
			"etcd.key":                         base64.StdEncoding.EncodeToString(etcdkey),
			"etcd-peer.crt":                    base64.StdEncoding.EncodeToString(etcdpeercrt),
			"etcd-peer.key":                    base64.StdEncoding.EncodeToString(etcdpeerkey),
			"kc-server-etcd-client.crt":        base64.StdEncoding.EncodeToString(etcdclientcrt),
			"kc-server-etcd-client.key":        base64.StdEncoding.EncodeToString(etcdclientkey),
			"kube-etcd-healthcheck-client.crt": base64.StdEncoding.EncodeToString(etcdhealthcheckcrt),
			"kube-etcd-healthcheck-client.key": base64.StdEncoding.EncodeToString(etcdhealthcheckkey),
		},
	}
	_, err = client.CreateConfigMap(context.TODO(), etcdcm)
	if err != nil {
		logger.Fatal(err)
	}

	natsPath := filepath.Join(options.HomeDIR, options.DefaultPath, options.DefaultNatsPKIPath)
	natsservercert, err := os.ReadFile(fmt.Sprintf("%s/kc-server-nats-server.crt", natsPath))
	if err != nil {
		logger.Fatal(err)
	}
	natsserverkey, err := os.ReadFile(fmt.Sprintf("%s/kc-server-nats-server.key", natsPath))
	if err != nil {
		logger.Fatal(err)
	}
	natsclientcert, err := os.ReadFile(fmt.Sprintf("%s/kc-server-nats-client.crt", natsPath))
	if err != nil {
		logger.Fatal(err)
	}
	natsclientkey, err := os.ReadFile(fmt.Sprintf("%s/kc-server-nats-client.key", natsPath))
	if err != nil {
		logger.Fatal(err)
	}
	natscm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindConfigMap,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: constatns.KcNatsCertsConfigMapName,
		},
		Data: map[string]string{
			"kc-server-nats-server.crt": base64.StdEncoding.EncodeToString(natsservercert),
			"kc-server-nats-server.key": base64.StdEncoding.EncodeToString(natsserverkey),
			"kc-server-nats-client.crt": base64.StdEncoding.EncodeToString(natsclientcert),
			"kc-server-nats-client.key": base64.StdEncoding.EncodeToString(natsclientkey),
		},
	}
	_, err = client.CreateConfigMap(context.TODO(), natscm)
	if err != nil {
		logger.Fatal(err)
	}
}

func caList() []certutils.Config {
	certPath := filepath.Join(options.HomeDIR, options.DefaultPath, options.DefaultCaPath)
	return []certutils.Config{
		{
			Path:         certPath,
			BaseName:     options.Ca,
			CommonName:   options.Ca,
			Organization: []string{"kubeclipper.io"},
			Year:         100,
			AltNames:     certutils.AltNames{},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		},
	}
}

func certList(pki, caName string, altNames []string, commonNameUsage map[string][]x509.ExtKeyUsage) []certutils.Config {
	certPath := filepath.Join(options.HomeDIR, options.DefaultPath, pki)
	alt := certutils.AltNames{
		DNSNames: map[string]string{
			"localhost": "localhost",
		},
		IPs: map[string]net.IP{
			"127.0.0.1":               net.IPv4(127, 0, 0, 1),
			net.IPv6loopback.String(): net.IPv6loopback,
		},
	}
	for _, altName := range altNames {
		if ip := net.ParseIP(altName); ip != nil {
			alt.IPs[ip.String()] = ip
			continue
		}
		alt.DNSNames[altName] = altName
	}
	logger.V(2).Infof("Etcd alt DNS : [%v], Etcd alt IPs: [%v]", alt.DNSNames, alt.IPs)
	certConfig := make([]certutils.Config, 0)
	for commonName, usages := range commonNameUsage {
		conf := certutils.Config{
			Path:         certPath,
			BaseName:     commonName,
			CAName:       caName,
			CommonName:   commonName,
			Organization: []string{"kubeclipper.io"},
			Year:         100,
			AltNames:     alt,
			Usages:       usages,
		}
		certConfig = append(certConfig, conf)
	}
	return certConfig
}

func clientCertList(pki, caName string, altNames, organization []string, commonNameUsage map[string][]x509.ExtKeyUsage) []certutils.Config {
	certPath := filepath.Join(options.HomeDIR, options.DefaultPath, pki)
	alt := certutils.AltNames{
		DNSNames: map[string]string{
			"localhost": "localhost",
		},
		IPs: map[string]net.IP{
			"127.0.0.1":               net.IPv4(127, 0, 0, 1),
			net.IPv6loopback.String(): net.IPv6loopback,
		},
	}
	for _, altName := range altNames {
		if ip := net.ParseIP(altName); ip != nil {
			alt.IPs[ip.String()] = ip
			continue
		}
		alt.DNSNames[altName] = altName
	}
	logger.V(2).Infof("client cert alt DNS : [%v], client cert alt IPs: [%v]", alt.DNSNames, alt.IPs)
	certConfig := make([]certutils.Config, 0)
	for commonName, usages := range commonNameUsage {
		conf := certutils.Config{
			Path:         certPath,
			BaseName:     commonName,
			CAName:       caName,
			CommonName:   commonName,   // e.g. kcctl,parse as username
			Organization: organization, // e.g. system:kcctl,parse as userGroup
			Year:         100,
			AltNames:     alt,
			Usages:       usages,
		}
		certConfig = append(certConfig, conf)
	}
	return certConfig
}
