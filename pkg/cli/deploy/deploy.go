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
	"crypto"
	"crypto/x509"
	"fmt"
	"math"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/sethvargo/go-password/password"
	"gopkg.in/yaml.v2"

	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"

	"github.com/kubeclipper/kubeclipper/pkg/cli/join"

	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
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
  kcctl deploy --pk-file ~/.ssh/id_rsa

  # Deploy AIO env and change etcd port
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --passwd 'YOUR-SSH-PASSWORD' --pkg kc-minimal.tar.gz --etcd-port 12379 --etcd-peer-port 12380 --etcd-metric-port 12381

  # Deploy HA env
  kcctl deploy --server 192.168.234.3,192.168.234.4,192.168.234.5 --agent 192.168.234.3 --passwd 'YOUR-SSH-PASSWORD' --pkg kc-minimal.tar.gz --etcd-port 12379 --etcd-peer-port 12380 --etcd-metric-port 12381

  # Deploy env use SSH key instead of password
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --pk-file ~/.ssh/id_rsa --pkg kc-minimal.tar.gz

  # Deploy env use remove http/https resource server
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 --pk-file ~/.ssh/id_rsa --pkg https://github.com/kubeclipper/kubeclipper/release/kc-minimal.tar.gz

  # Deploy env with many agent node in same region.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:192.168.10.123,192.168.10.124  --pk-file ~/.ssh/id_rsa --pkg https://github.com/kubeclipper/kubeclipper/release/kc-minimal.tar.gz

  # Deploy env with many agent node in different region.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:1.1.1.1,1.1.1.2 --agent us-west-2:1.1.1.3 --pk-file ~/.ssh/id_rsa --pkg https://github.com/kubeclipper/kubeclipper/release/kc-minimal.tar.gz

  # Deploy env with many agent node which has orderly ip.
  # this will add 10 agent,1.1.1.1, 1.1.1.2, ... 1.1.1.10.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:1.1.1.1-1.1.1.10 --pk-file ~/.ssh/id_rsa --pkg https://github.com/kubeclipper/kubeclipper/release/kc-minimal.tar.gz

  # Deploy from config.
  kcctl deploy --deploy-config deploy-config.yaml

  Please read 'kcctl deploy -h' get more deploy flags`
	defaultPkg              = "https://oss.kubeclipper.io/release/kc-latest.tar.gz"
	allInOneEtcdClientPort  = 12379
	allInOneEtcdPeerPort    = 12380
	allInOneEtcdMetricsPort = 12381
)

type DeployOptions struct {
	options.IOStreams
	deployConfig *options.DeployConfig
	allNodes     []string
	servers      map[string]string
	agents       []string // user input's agents,maybe with region,need to parse.
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
	o.deployConfig.AddFlags(cmd.Flags())

	cmd.AddCommand(NewCmdDeployConfig(o))

	return cmd
}

func (d *DeployOptions) Complete() error {
	var err error
	if err = d.deployConfig.Complete(); err != nil {
		return err
	}
	// if both the server and agent are empty, set the all-in-one environment
	if d.deployConfig.ServerIPs == nil && d.agents == nil {
		// TODO: Replace it with the official address
		if d.deployConfig.Pkg == "" {
			d.deployConfig.Pkg = defaultPkg
		}
		// set etcd port to avoid conflicts with k8s
		d.deployConfig.EtcdConfig.ClientPort = allInOneEtcdClientPort
		d.deployConfig.EtcdConfig.PeerPort = allInOneEtcdPeerPort
		d.deployConfig.EtcdConfig.MetricsPort = allInOneEtcdMetricsPort
		ip, err := netutil.GetDefaultIP(true, d.deployConfig.IPDetect)
		if err != nil {
			return err
		}
		// set the local host as the kc server and agent
		d.deployConfig.ServerIPs = []string{ip.String()}
		d.agents = []string{ip.String()}
	}

	if d.deployConfig.Config == "" { // use flags if not specify config file
		if d.deployConfig.AgentRegions, err = join.BuildAgentRegion(d.agents, d.deployConfig.DefaultRegion); err != nil {
			return err
		}
	}

	d.allNodes = sets.NewString().
		Insert(d.deployConfig.ServerIPs...).
		Insert(d.deployConfig.AgentRegions.ListIP()...).
		List()

	if !d.deployConfig.MQ.External {
		d.deployConfig.MQ.IPs = d.deployConfig.ServerIPs // internal mq use server ips as mq ips
		if d.deployConfig.MQ.TLS {                       // fill default tls file path
			d.deployConfig.MQ.CA = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultCaPath, fmt.Sprintf("%s.crt", options.Ca))
			d.deployConfig.MQ.ClientCert = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.crt", options.NatsIOClient))
			d.deployConfig.MQ.ClientKey = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.key", options.NatsIOClient))
		}
	}

	return nil
}

func (d *DeployOptions) ValidateArgs() error {
	if d.deployConfig.IPDetect != "" && !autodetection.CheckMethod(d.deployConfig.IPDetect) {
		return fmt.Errorf("invalid ip detect method,suppot [first-found,interface=xxx,cidr=xxx] now")
	}
	if d.deployConfig.Pkg == "" {
		return fmt.Errorf("--pkg must be specified")
	}
	if d.deployConfig.SSHConfig.PkFile == "" && d.deployConfig.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
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
		name := utils.GetRemoteHostName(d.deployConfig.SSHConfig, sip)
		if name == "" {
			logger.Fatal("get remote hostname failed")
		}
		d.servers[sip] = name
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
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, "systemctl --all --type service --state running | grep -Fq -e chronyd -e ntpd")
		if err != nil {
			return err
		}
		if ret.ExitCode != 0 {
			err = fmt.Errorf("chronyd or ntpd service not running, may cause service internal error")
		}
		return err
	}
)

func generateCommonPreCheckFunc(name string) precheckFunc {
	return func(sshConfig *sshutils.SSH, host string) error {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, fmt.Sprintf("systemctl --all --type service | grep -Fq %s", name))
		logger.V(2).Infof("exit code %d, err %v", ret.ExitCode, err)
		if err != nil {
			return err
		}
		if ret.ExitCode == 0 {
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
			output := d.deployConfig.SSHConfig.CmdToString(host, "date +%s", "")
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
	return true
}

func (d *DeployOptions) RunDeploy() error {
	if err := d.generateAndSendCerts(); err != nil {
		return err
	}
	d.sendPackage()
	d.deployEtcd()
	// TODO: add check etcd status instead of time.sleep
	time.Sleep(5 * time.Second)
	d.deployKcServer()
	time.Sleep(5 * time.Second)
	d.deployKcAgent()
	d.deployKcConsole()
	d.removeTempFile()
	fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
	return nil
}

// check node kc-etcd/kc-server/kc-agent service already exists
func (d *DeployOptions) check() []error {
	var errs []error
	// check node kc-etcd/kc-server service already exists
	for node := range d.servers {
		res := d.deployConfig.SSHConfig.CmdToString(node, "systemctl status kc-etcd | grep Active && systemctl status kc-server | grep Active", "")
		if strings.Contains(res, "active (running)") {
			errs = append(errs, fmt.Errorf(`node %s kc-etcd/kc-server service already exists\n`, node))
		}
	}

	// check node kc-agent service already exists
	for _, node := range d.allNodes {
		res := d.deployConfig.SSHConfig.CmdToString(node, "systemctl status kc-agent | grep Active", "")
		if strings.Contains(res, "active (running)") {
			errs = append(errs, fmt.Errorf(`node %s kc-agent service already exists\n`, node))
		}
	}

	return errs
}

func (d *DeployOptions) sendPackage() {
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s", filepath.Join(config.DefaultPkgPath, "kc"),
		filepath.Join(config.DefaultPkgPath, path.Base(d.deployConfig.Pkg)), config.DefaultPkgPath)
	cp := sshutils.WrapSh(fmt.Sprintf("cp -rf %s /usr/local/bin/", filepath.Join(config.DefaultPkgPath, "kc/bin/*")))
	// rm -rf /root/kc && tar -xvf /root/kc/pkg/kc.tar -C ~/kc/pkg && /bin/bash -c 'cp -rf /root/kc/pkg/kc/bin/* /usr/local/bin/'
	hook := sshutils.Combine([]string{tar, cp})
	err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.Pkg, d.allNodes, config.DefaultPkgPath, nil, &hook)
	if err != nil {
		logger.Fatalf("sendPackage err:%s", err.Error())
	}
}

func (d DeployOptions) generateAndSendCerts() error {
	var altNames []string
	for _, name := range d.servers {
		altNames = append(altNames, name)
	}
	cas := caList()
	certs := make([]certutils.Config, 0)

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
		natsCert = certList(options.DefaultNatsPKIPath, options.Ca, append(altNames, d.deployConfig.ServerIPs...), natsCommonNameUsages)
		certs = append(certs, natsCert...)
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
				d.deployConfig.AgentRegions.ListIP(), filepath.Dir(d.deployConfig.MQ.CA), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientCert,
				d.deployConfig.AgentRegions.ListIP(), filepath.Dir(d.deployConfig.MQ.ClientCert), nil, nil); err != nil {
				return err
			}
			if err := utils.SendPackageV2(d.deployConfig.SSHConfig, d.deployConfig.MQ.ClientKey,
				d.deployConfig.AgentRegions.ListIP(), filepath.Dir(d.deployConfig.MQ.ClientKey), nil, nil); err != nil {
				return err
			}
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
	data["ClientURLs"] = fmt.Sprintf("https://127.0.0.1:%d,https://%s:%d", d.deployConfig.EtcdConfig.ClientPort, ip, d.deployConfig.EtcdConfig.ClientPort)
	data["MetricsURLs"] = fmt.Sprintf("http://127.0.0.1:%d", d.deployConfig.EtcdConfig.MetricsPort)
	data["PeerURLs"] = fmt.Sprintf("https://%s:%d", ip, d.deployConfig.EtcdConfig.PeerPort)
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
	for k := range d.servers {
		serverUpstream = serverUpstream + fmt.Sprintf(" http://%s:%d", k, d.deployConfig.ServerPort)
	}
	var data = make(map[string]interface{})
	data["ConsolePort"] = d.deployConfig.ConsolePort
	data["ServerUpstream"] = serverUpstream
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (d *DeployOptions) getKcServerConfigTemplateContent(ip string) string {
	tmpl, err := template.New("text").Parse(config.KcServerConfigTmpl)
	if err != nil {
		logger.Fatalf("template parse failed: %s", err.Error())
	}
	var mqServerEndpoints []string
	for _, v := range d.deployConfig.MQ.IPs {
		mqServerEndpoints = append(mqServerEndpoints, fmt.Sprintf("%s:%d", v, d.deployConfig.MQ.Port))
	}
	etcdEndpoints := []string{fmt.Sprintf("%s:%d", ip, d.deployConfig.EtcdConfig.ClientPort)}
	var data = make(map[string]interface{})
	data["ServerAddress"] = ip
	data["ServerPort"] = d.deployConfig.ServerPort
	// TODO: make auto generate
	data["JwtSecret"] = d.deployConfig.JWTSecret
	data["StaticServerPort"] = d.deployConfig.StaticServerPort
	data["StaticServerPath"] = d.deployConfig.StaticServerPath
	if d.deployConfig.Debug {
		data["LogLevel"] = "debug"
	} else {
		data["LogLevel"] = "info"
	}
	data["EtcdEndpoints"] = etcdEndpoints
	data["EtcdCaPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultCaPath, fmt.Sprintf("%s.crt", options.Ca))
	data["EtcdCertPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.crt", options.EtcdKcClient))
	data["EtcdKeyPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultEtcdPKIPath, fmt.Sprintf("%s.key", options.EtcdKcClient))

	data["MQExternal"] = d.deployConfig.MQ.External
	data["MQUser"] = d.deployConfig.MQ.User
	data["MQAuthToken"] = d.deployConfig.MQ.Secret
	data["MQServerEndpoints"] = mqServerEndpoints
	data["MQTLS"] = d.deployConfig.MQ.TLS
	if !d.deployConfig.MQ.External {
		data["MQServerAddress"] = ip
		data["MQServerPort"] = d.deployConfig.MQ.Port
		data["MQClusterPort"] = d.deployConfig.MQ.ClusterPort
		data["LeaderHost"] = fmt.Sprintf("%s:%d", d.deployConfig.ServerIPs[0], d.deployConfig.MQ.ClusterPort)
		if d.deployConfig.MQ.TLS {
			data["MQServerCertPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.crt", options.NatsIOServer))
			data["MQServerKeyPath"] = filepath.Join(options.DefaultKcServerConfigPath, options.DefaultNatsPKIPath, fmt.Sprintf("%s.key", options.NatsIOServer))
		}
	}

	if d.deployConfig.MQ.TLS {
		data["MQCaPath"] = d.deployConfig.MQ.CA
		data["MQClientCertPath"] = d.deployConfig.MQ.ClientCert
		data["MQClientKeyPath"] = d.deployConfig.MQ.ClientKey
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (d *DeployOptions) getKcAgentConfigTemplateContent(region string) string {
	tmpl, err := template.New("text").Parse(config.KcAgentConfigTmpl)
	if err != nil {
		logger.Fatalf("template parse failed: %s", err.Error())
	}
	var mqServerEndpoints []string
	for _, v := range d.deployConfig.MQ.IPs {
		mqServerEndpoints = append(mqServerEndpoints, fmt.Sprintf("%s:%d", v, d.deployConfig.MQ.Port))
	}

	var data = make(map[string]interface{})
	data["AgentID"] = uuid.New().String()
	data["Region"] = region
	data["IPDetect"] = d.deployConfig.IPDetect
	data["StaticServerAddress"] = fmt.Sprintf("http://%s:%d", d.deployConfig.ServerIPs[0], d.deployConfig.StaticServerPort)
	if d.deployConfig.Debug {
		data["LogLevel"] = "debug"
	} else {
		data["LogLevel"] = "info"
	}
	data["MQServerEndpoints"] = mqServerEndpoints
	data["MQAuthToken"] = d.deployConfig.MQ.Secret
	data["MQExternal"] = d.deployConfig.MQ.External
	data["MQUser"] = d.deployConfig.MQ.User
	data["MQAuthToken"] = d.deployConfig.MQ.Secret
	data["MQTLS"] = d.deployConfig.MQ.TLS
	if d.deployConfig.MQ.TLS {
		data["MQCaPath"] = filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultCaPath, filepath.Base(d.deployConfig.MQ.CA))
		data["MQClientCertPath"] = filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultNatsPKIPath, filepath.Base(d.deployConfig.MQ.ClientCert))
		data["MQClientKeyPath"] = filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultNatsPKIPath, filepath.Base(d.deployConfig.MQ.ClientKey))
	}
	data["OpLogDir"] = d.deployConfig.OpLog.Dir
	data["OpLogThreshold"] = d.deployConfig.OpLog.Threshold
	data["KcImageRepoMirror"] = d.deployConfig.ImageProxy.KcImageRepoMirror
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (d *DeployOptions) deployKcServer() {
	cmdList := []string{
		"mkdir -pv /etc/kubeclipper-server",
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/configs/*.json /etc/kubeclipper-server/", config.DefaultPkgPath)),
		sshutils.WrapEcho(config.KcServerService, "/usr/lib/systemd/system/kc-server.service"),
		fmt.Sprintf("mkdir -pv %s ", d.deployConfig.StaticServerPath),
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/resource/* %s/", config.DefaultPkgPath, d.deployConfig.StaticServerPath)),
	}
	for _, cmd := range cmdList {
		err := sshutils.CmdBatchWithSudo(d.deployConfig.SSHConfig, d.deployConfig.ServerIPs, cmd, sshutils.DefaultWalk)
		if err != nil {
			logger.Fatalf("deploy kc server failed due to %s", err.Error())
		}
	}

	for _, host := range d.deployConfig.ServerIPs {
		data := d.getKcServerConfigTemplateContent(host)
		cmd := sshutils.WrapEcho(data, "/etc/kubeclipper-server/kubeclipper-server.yaml") +
			"&& systemctl daemon-reload && systemctl enable kc-server --now"
		ret, err := sshutils.SSHCmdWithSudo(d.deployConfig.SSHConfig, host, cmd)
		if err != nil {
			logger.Fatalf("[%s]deploy kc server failed due to %s", host, err.Error())
		}
		if err = ret.Error(); err != nil {
			logger.Fatalf("[%s]deploy kc server failed due to %s", host, err.Error())
		}
		// TODO: check server healthz endpoint instead of time sleep
		time.Sleep(1 * time.Second)
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
	for region, agents := range d.deployConfig.AgentRegions {
		for _, agent := range agents {
			agentConfig := d.getKcAgentConfigTemplateContent(region)
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
	_, err := cmdutil.RunCmd(false, "mkdir", "-p", filepath.Dir(options.DefaultDeployConfigPath))
	if err != nil {
		logger.Fatalf("dump config failed due to %s", err.Error())
	}
	b, err := yaml.Marshal(d.deployConfig)
	if err != nil {
		logger.Fatalf("dump config failed due to %s", err.Error())
	}
	if err = utils.WriteToFile(options.DefaultDeployConfigPath, b); err != nil {
		logger.Fatalf("dump config to %s failed due to %s", options.DefaultDeployConfigPath, err.Error())
	}
	logger.V(2).Info("dump config to %s", options.DefaultDeployConfigPath)
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
			d.deployConfig.AgentRegions.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.AgentRegions.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil)
		if err != nil {
			return err
		}
	}
	return nil
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
