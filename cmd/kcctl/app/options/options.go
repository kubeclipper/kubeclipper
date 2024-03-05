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

package options

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/subosito/gotenv"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/auditing/option"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"

	"github.com/spf13/pflag"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
)

var (
	HomeDIR = homedir.HomeDir()
)

const (
	Contact = `
 _   __      _          _____ _ _
| | / /     | |        /  __ \ (_)
| |/ / _   _| |__   ___| /  \/ |_ _ __  _ __   ___ _ __
|    \| | | | '_ \ / _ \ |   | | | '_ \| '_ \ / _ \ '__|
| |\  \ |_| | |_) |  __/ \__/\ | | |_) | |_) |  __/ |
\_| \_/\__,_|_.__/ \___|\____/_|_| .__/| .__/ \___|_|
                                 | |   | |
                                 |_|   |_|
        repository: github.com/kubeclipper`
)

const (
	DefaultPath                = ".kc"
	DefaultDeployConfig        = "deploy-config.yaml"
	DefaultConfig              = "config"
	DefaultCaPath              = "pki"
	DefaultKcctlPKIPath        = "pki/kcctl"
	DefaultEtcdPKIPath         = "pki/etcd"
	DefaultNatsPKIPath         = "pki/nats"
	DefaultKCPKIPath           = "pki/kc"
	DefaultKcServerConfigPath  = "/etc/kubeclipper-server"
	DefaultKcAgentConfigPath   = "/etc/kubeclipper-agent"
	DefaultKcConsoleConfigPath = "/etc/kc-console"

	DefaultRegion = "default"

	// EtcdCa          = "etcd-ca"   //ca
	Ca               = "ca"
	EtcdPeer         = "etcd-peer" // peer
	EtcdServer       = "etcd"      // server
	EtcdKcClient     = "kc-server-etcd-client"
	EtcdHealthCheck  = "kube-etcd-healthcheck-client" // healthcheck-client
	NatsIOClient     = "kc-server-nats-client"
	NatsIOServer     = "kc-server-nats-server"
	KCServer         = "kc-server"
	NatsAltNameProxy = "proxy.kubeclipper.io" // add nats server SAN for agent proxy
	KCServerAltName  = "server.kubeclipper.io"

	AdminKcctlCert = "admin"
)

const IPDetectDescription = `
To eliminate node specific IP address configuration,the KubeClipper can be configuredto autodetect these IP addresses.
In many systems, there might be multiple physical interfaces on a host, or possibly multiple IP addresses configured 
on a physical interface.In these cases, there are multiple addresses to choose from and soautodetection of the correct 
address can be tricky.

The IP autodetection methods are provided to improve the selection of thecorrect address, by limiting the selection 
based on suitable criteria for your deployment.

The following sections describe the available IP autodetection methods.

1. first-found
The first-found option enumerates all interface IP addresses and returns the first valid IP address (based on IP version
and type of address) on the first valid interface. 
Certain known “local” interfaces are omitted, such as the docker bridge.The order that both the interfaces and the IP 
addresses are listed is system dependent.

This is the default detection method. 
However, since this method only makes a very simplified guess,it is recommended to either configure the node with a 
specific IP address,or to use one of the other detection methods.

2. interface=INTERFACE-REGEX
The interface method uses the supplied interface regular expression to enumerate matching interfaces and to return the 
first IP address on the first matching interface. 
The order that both the interfaces and the IP addresses are listed is system dependent.

Example with valid IP address on interface eth0, eth1, eth2 etc.:
interface=eth.*

3. cidr=CIDR
The cidr method will select any IP address from the node that falls within the given CIDRs.
Example:
cidr=10.0.1.0/24,10.0.2.0/24`

var AssumeYes bool

var (
	DefaultDeployConfigPath = filepath.Join(HomeDIR, DefaultPath, DefaultDeployConfig)
	DefaultConfigPath       = filepath.Join(HomeDIR, DefaultPath, DefaultConfig)
)

const (
	ResourceNode      = "node"
	ResourceCluster   = "cluster"
	ResourceUser      = "user"
	ResourceRole      = "role"
	ResourceConfigMap = "configmap"
	UpgradeKcctl      = "kcctl"
	UpgradeAgent      = "agent"
	UpgradeServer     = "server"
	UpgradeConsole    = "console"
	UpgradeAll        = "all"
)

type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

type CliOptions struct {
	Config string
	cfg    *config.Config
}

func NewCliOptions() *CliOptions {
	return &CliOptions{
		Config: DefaultConfigPath,
		cfg:    config.New(),
	}
}

func (c *CliOptions) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Config, "config", c.Config, "Path to the config file to use for CLI requests.")
}

func (c *CliOptions) ToRawConfig() config.Config {
	if c.cfg == nil {
		return config.Config{}
	}
	return *c.cfg
}

func (c *CliOptions) Complete() error {
	var err error
	c.cfg, err = config.TryLoadFromFile(c.Config)
	if os.IsNotExist(err) {
		return fmt.Errorf("auth config %s not exist,please use kcctl login cmd to login first", c.Config)
	}
	return err
}

type Etcd struct {
	ClientPort  int    `json:"clientPort" yaml:"clientPort,omitempty"`
	PeerPort    int    `json:"peerPort" yaml:"peerPort,omitempty"`
	MetricsPort int    `json:"metricsPort" yaml:"metricsPort,omitempty"`
	DataDir     string `json:"dataDir" yaml:"dataDir,omitempty"`
}

type MQ struct {
	External    bool     `json:"external" yaml:"external,omitempty"`
	TLS         bool     `json:"tls" yaml:"tls,omitempty"`
	CA          string   `json:"ca" yaml:"ca,omitempty"`
	ClientCert  string   `json:"clientCert" yaml:"clientCert,omitempty"`
	ClientKey   string   `json:"clientKey" yaml:"clientKey,omitempty"`
	IPs         []string `json:"ips" yaml:"ips,omitempty"`
	Port        int      `json:"port" yaml:"port,omitempty"`
	ClusterPort int      `json:"clusterPort" yaml:"clusterPort,omitempty"`
	User        string   `json:"user" yaml:"user,omitempty"`
	Secret      string   `json:"secret" yaml:"secret,omitempty"`
}

type OpLog struct {
	Dir       string `json:"dir" yaml:"dir,omitempty"`
	Threshold int    `json:"threshold" yaml:"threshold,omitempty"`
}

type ImageProxy struct {
	KcImageRepoMirror string `json:"kcImageRepoMirror" yaml:"kcImageRepoMirror,omitempty"`
}

type Agents map[string]Metadata // key:ip

func (a Agents) ListIP() []string {
	list := make([]string, 0, len(a))
	for ip := range a {
		list = append(list, ip)
	}
	return list
}

func (a Agents) Exists(ip string) bool {
	_, ok := a[ip]
	return ok
}

func (a Agents) ExistsByID(id string) bool {
	return a.idToIP(id) != ""
}

func (a Agents) Delete(id string) {
	ip := a.idToIP(id)
	delete(a, ip)
}

func (a Agents) idToIP(id string) string {
	var ip string
	for k, v := range a {
		if v.AgentID == id {
			ip = k
		}
	}
	return ip
}

func (a Agents) Add(ip string, metadata Metadata) {
	a[ip] = metadata
}

// Metadata user custom node info,region will use by filter,use label,others use annotation.
type Metadata struct {
	AgentID string `json:"agentID" yaml:"agentID,omitempty"`
	Region  string `json:"region" yaml:"region,omitempty"`
	FloatIP string `json:"floatIP" yaml:"floatIP,omitempty"`

	// proxy server for proxy kc-server(mq and static server)
	ProxyServer string `json:"proxyServer" yaml:"proxyServer,omitempty"`
	// address for server to access k8s apiserver
	ProxyAPIServer string `json:"proxyAPIServer" yaml:"proxyAPIServer,omitempty"`
	// address for server to access node's ssh
	ProxySSH string `json:"proxySSH" yaml:"proxySSH,omitempty"`
}

type DeployConfig struct {
	Config                     string                         `json:"-" yaml:"-"`
	SSHConfig                  *sshutils.SSH                  `json:"ssh" yaml:"ssh,omitempty"`
	EtcdConfig                 *Etcd                          `json:"etcd" yaml:"etcd,omitempty"`
	ServerIPs                  []string                       `json:"serverIPs" yaml:"serverIPs,omitempty"`
	Agents                     Agents                         `json:"agents" yaml:"agents,omitempty"`
	Proxys                     []string                       `json:"proxys" yaml:"proxys,omitempty"`
	IPDetect                   string                         `json:"ipDetect" yaml:"ipDetect,omitempty"`
	NodeIPDetect               string                         `json:"nodeIPDetect" yaml:"nodeIPDetect,omitempty"`
	Debug                      bool                           `json:"debug" yaml:"debug,omitempty"`
	DefaultRegion              string                         `json:"defaultRegion" yaml:"defaultRegion,omitempty"`
	ServerPort                 int                            `json:"serverPort" yaml:"serverPort,omitempty"`
	TLS                        bool                           `json:"tls" yaml:"tls,omitempty"`
	StaticServerPort           int                            `json:"staticServerPort" yaml:"staticServerPort,omitempty"`
	StaticServerPath           string                         `json:"staticServerPath" yaml:"staticServerPath,omitempty"`
	Pkg                        string                         `json:"pkg" yaml:"pkg,omitempty"`
	ConsolePort                int                            `json:"consolePort" yaml:"consolePort,omitempty"`
	JWTSecret                  string                         `json:"jwtSecret" yaml:"jwtSecret,omitempty"`
	AuditOpts                  *option.AuditOptions           `json:"audit" yaml:"audit,omitempty"`
	MQ                         *MQ                            `json:"mq" yaml:"mq,omitempty"`
	OpLog                      *OpLog                         `json:"opLog" yaml:"opLog,omitempty"`
	ImageProxy                 *ImageProxy                    `json:"imageProxy" yaml:"imageProxy,omitempty"`
	AuthenticationOpts         *options.AuthenticationOptions `json:"authentication" yaml:"authentication,omitempty"`
	KCServerHealthCheckTimeout time.Duration                  `json:"kcServerHealthCheckTimeout" yaml:"kcServerHealthCheckTimeout,omitempty"`
}

type AgentRegions map[string][]string // key: region, value: ips
type FIPs map[string]string           // key: ip, value: fip

func (a AgentRegions) ListIP() []string {
	set := sets.NewString()
	for _, v := range a {
		set.Insert(v...)
	}
	return set.List()
}

func (a AgentRegions) Add(region, ip string) {
	if a.Exists(ip) {
		return
	}
	a[region] = append(a[region], ip)
}

func (a AgentRegions) Delete(ip string) {
	for region := range a {
		a[region] = sliceutil.RemoveString(a[region], func(item string) bool {
			return item == ip
		})
		if len(a[region]) == 0 {
			delete(a, region)
		}
	}
}

func (a AgentRegions) Exists(ip string) bool {
	for _, agents := range a {
		if sliceutil.HasString(agents, ip) {
			return true
		}
	}
	return false
}

func NewDeployOptions() *DeployConfig {
	return &DeployConfig{
		IPDetect:  autodetection.MethodFirst,
		SSHConfig: sshutils.NewSSH(),
		EtcdConfig: &Etcd{
			ClientPort:  12379,
			PeerPort:    12380,
			MetricsPort: 12381,
			DataDir:     "/var/lib/kc-etcd",
		},
		Debug:            false,
		DefaultRegion:    "default",
		ServerPort:       8080,
		TLS:              true,
		StaticServerPort: 8081,
		StaticServerPath: "/opt/kubeclipper-server/resource",
		AuditOpts:        option.NewAuditOptions(),
		MQ: &MQ{
			User:        "admin",
			TLS:         true,
			Port:        9889,
			ClusterPort: 9890,
		},
		ConsolePort: 80,
		OpLog: &OpLog{
			Dir:       "/var/log/kc-agent",
			Threshold: 1048576,
		},
		ImageProxy: &ImageProxy{
			KcImageRepoMirror: getRepoMirror(),
		},
		AuthenticationOpts:         options.NewAuthenticateOptions(),
		Agents:                     make(Agents),
		KCServerHealthCheckTimeout: time.Second * 30,
	}
}

func (c *DeployConfig) MergeDeployOptions() {
	d := NewDeployOptions()

	if c.OpLog == nil {
		c.OpLog = d.OpLog
	}

}

/*
NOTE: deploy cmd use local deploy-config,others cmd use online deploy-config.
and clean cmd use online deploy-config first,if can't get online deploy-config,
then use local deploy-config as a downgrade.
*/

func (c *DeployConfig) Complete() error {
	if c.Config == "" {
		return nil
	}

	if !utils.FileExist(c.Config) {
		return fmt.Errorf("%s is not exist", c.Config)
	}
	data, err := os.ReadFile(c.Config)
	if err != nil {
		return err
	}
	bytes, err := Omitempty(data)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, c)
	if err != nil {
		return err
	}
	// fill default region
	for ip := range c.Agents {
		metadata := c.Agents[ip]
		if metadata.Region == "" {
			if c.DefaultRegion == "" {
				return fmt.Errorf("one of region or defaultRegion must specify")
			}
			metadata.Region = c.DefaultRegion
			c.Agents[ip] = metadata
		}
	}

	return nil
}

// Omitempty use unmarshal+marshal to omit empty field.
func Omitempty(data []byte) ([]byte, error) {
	d := new(DeployConfig)
	err := yaml.Unmarshal(data, d)
	if err != nil {
		return nil, err
	}
	marshal, err := yaml.Marshal(d)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (c *DeployConfig) Write() error {
	path := c.Config
	if c.Config == "" {
		path = DefaultDeployConfigPath
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("dump config failed due to %s", err.Error())
	}
	if err = utils.WriteToFile(path, b); err != nil {
		return fmt.Errorf("dump config to %s failed due to %s", path, err.Error())
	}
	return nil
}

// getRepoMirror env variables have a higher priority than .env file
func getRepoMirror() string {
	if mirror := os.Getenv("KC_IMAGE_REPO_MIRROR"); mirror != "" {
		return mirror
	}
	_ = gotenv.Load("/etc/kc/kc.env")
	return os.Getenv("KC_IMAGE_REPO_MIRROR")
}

func (c *DeployConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.Config, "deploy-config", "c", c.Config, "Path to the config file to use for Deploy.")
	flags.StringVar(&c.IPDetect, "ip-detect", c.IPDetect, fmt.Sprintf("Kc agent node ip detect method. Used to route between nodes. \n%s", IPDetectDescription))
	flags.StringVar(&c.NodeIPDetect, "node-ip-detect", c.NodeIPDetect, fmt.Sprintf("Kc agent node ip detect method. Used for routing between nodes in the kubernetes cluster. If not specified, ip-detect is inherited. \n%s", IPDetectDescription))
	flags.BoolVar(&c.Debug, "debug", c.Debug, "Deploy kc use debug mode")
	flags.StringVarP(&c.DefaultRegion, "region", "r", c.DefaultRegion, "Kc agent default region")
	flags.BoolVar(&c.TLS, "tls", c.TLS, "Kc api server  use tls mode")
	flags.IntVar(&c.ServerPort, "server-port", c.ServerPort, "Kc server port")
	flags.IntVar(&c.StaticServerPort, "static-server-port", c.StaticServerPort, "Kc static server port")
	flags.StringVar(&c.StaticServerPath, "static-server-path", c.StaticServerPath, "Kc static server path(absolute path")
	flags.BoolVar(&c.MQ.External, "mq-external", c.MQ.External, "Kc external mq")
	flags.BoolVar(&c.MQ.TLS, "mq-tls", c.MQ.TLS, "Kc external mq client and built-in mq client/server use tls mode. built-in mq client/server cert automatic generation")
	flags.StringVar(&c.MQ.CA, "mq-ca", c.MQ.CA, "Kc external mq client ca file path(absolute path)")
	flags.StringVar(&c.MQ.ClientCert, "mq-cert", c.MQ.ClientCert, "Kc external mq client cert file path(absolute path)")
	flags.StringVar(&c.MQ.ClientKey, "mq-key", c.MQ.ClientKey, "Kc external mq client key file path(absolute path)")
	flags.StringSliceVar(&c.MQ.IPs, "mq-ips", c.MQ.IPs, "external mq ips.")
	flags.IntVar(&c.MQ.Port, "mq-port", c.MQ.Port, "Kc built-in mq or external mq port")
	flags.StringVar(&c.MQ.User, "mq-user", c.MQ.User, "external mq user")
	flags.StringVar(&c.MQ.Secret, "mq-secret", c.MQ.Secret, "external mq user secret")
	flags.IntVar(&c.MQ.ClusterPort, "mq-cluster-port", c.MQ.ClusterPort, "Kc mq cluster port")
	flags.StringSliceVar(&c.ServerIPs, "server", c.ServerIPs, "Kc server ips")
	flags.IntVar(&c.EtcdConfig.ClientPort, "etcd-port", c.EtcdConfig.ClientPort, "Etcd port")
	flags.IntVar(&c.EtcdConfig.PeerPort, "etcd-peer-port", c.EtcdConfig.PeerPort, "Etcd peer port")
	flags.IntVar(&c.EtcdConfig.MetricsPort, "etcd-metric-port", c.EtcdConfig.MetricsPort, "Etcd metric port")
	flags.StringVar(&c.EtcdConfig.DataDir, "etcd-data-dir", c.EtcdConfig.DataDir, "Etcd data dir(absolute path)")
	flags.StringVar(&c.Pkg, "pkg", c.Pkg, "Package resource url (path or http url)")
	flags.IntVar(&c.ConsolePort, "console-port", c.ConsolePort, "kc console port")
	flags.StringVar(&c.OpLog.Dir, "oplog-dir", c.OpLog.Dir, "kc agent operation log dir")
	flags.IntVar(&c.OpLog.Threshold, "oplog-threshold", c.OpLog.Threshold, "kc agent operation log single threshold")
	flags.StringVar(&c.ImageProxy.KcImageRepoMirror, "kc-image-repo-mirror", c.ImageProxy.KcImageRepoMirror, "K8s image repository mirror")
	flags.DurationVar(&c.KCServerHealthCheckTimeout, "kc-server-health-check-timeout", c.KCServerHealthCheckTimeout, "kc server health check timeout, default is 30s")

	AddFlagsToSSH(c.SSHConfig, flags)
}

func AddFlagsToSSH(ssh *sshutils.SSH, flags *pflag.FlagSet) {
	flags.StringVarP(&ssh.User, "user", "u", ssh.User, "Deploy ssh user")
	flags.StringVar(&ssh.Password, "passwd", ssh.Password, "Deploy ssh password")
	flags.IntVar(&ssh.Port, "ssh-port", 22, "ssh connection port of agent nodes")
	flags.StringVar(&ssh.PkFile, "pk-file", ssh.PkFile, "ssh pk file which used to remote access other agent nodes")
	flags.StringVar(&ssh.PkPassword, "pk-passwd", ssh.PkPassword, "the password of the ssh pk file which used to remote access other agent nodes")
}

func (c *DeployConfig) GetKcServerConfigTemplateContent(ip string) (string, error) {
	tmpl, err := template.New("text").Parse(config.KcServerConfigTmpl)
	if err != nil {
		return "", fmt.Errorf("template parse failed: %s", err.Error())
	}
	var mqServerEndpoints []string
	for _, v := range c.MQ.IPs {
		mqServerEndpoints = append(mqServerEndpoints, fmt.Sprintf("%s:%d", v, c.MQ.Port))
	}
	etcdEndpoints := []string{fmt.Sprintf("%s:%d", ip, c.EtcdConfig.ClientPort)}
	var data = make(map[string]interface{})
	data["ServerAddress"] = ip
	data["ServerPort"] = c.ServerPort
	if c.TLS {
		data["TLS"] = true
		data["TLSCertFile"] = filepath.Join(DefaultKcServerConfigPath, DefaultKCPKIPath, fmt.Sprintf("%s.crt", KCServer))
		data["TLSPrivateKey"] = filepath.Join(DefaultKcServerConfigPath, DefaultKCPKIPath, fmt.Sprintf("%s.key", KCServer))
		data["CACertFile"] = filepath.Join(DefaultKcServerConfigPath, DefaultCaPath, "ca.crt")
	}
	// TODO: make auto generate
	data["JwtSecret"] = c.JWTSecret
	data["InitialPassword"] = c.AuthenticationOpts.InitialPassword
	data["RetentionPeriod"] = c.AuditOpts.RetentionPeriod
	data["MaximumEntries"] = c.AuditOpts.MaximumEntries
	data["AuditLevel"] = c.AuditOpts.AuditLevel
	data["AuthenticateRateLimiterMaxTries"] = c.AuthenticationOpts.AuthenticateRateLimiterMaxTries
	data["AuthenticateRateLimiterDuration"] = c.AuthenticationOpts.AuthenticateRateLimiterDuration
	data["LoginHistoryMaximumEntries"] = c.AuthenticationOpts.LoginHistoryMaximumEntries
	data["LoginHistoryRetentionPeriod"] = c.AuthenticationOpts.LoginHistoryRetentionPeriod
	data["StaticServerPort"] = c.StaticServerPort
	data["StaticServerPath"] = c.StaticServerPath
	if c.Debug {
		data["LogLevel"] = "debug"
	} else {
		data["LogLevel"] = "info"
	}
	data["EtcdEndpoints"] = etcdEndpoints
	data["EtcdCaPath"] = filepath.Join(DefaultKcServerConfigPath, DefaultCaPath, fmt.Sprintf("%s.crt", Ca))
	data["EtcdCertPath"] = filepath.Join(DefaultKcServerConfigPath, DefaultEtcdPKIPath, fmt.Sprintf("%s.crt", EtcdKcClient))
	data["EtcdKeyPath"] = filepath.Join(DefaultKcServerConfigPath, DefaultEtcdPKIPath, fmt.Sprintf("%s.key", EtcdKcClient))

	data["MQExternal"] = c.MQ.External
	data["MQUser"] = c.MQ.User
	data["MQAuthToken"] = c.MQ.Secret
	data["MQServerEndpoints"] = mqServerEndpoints
	data["MQTLS"] = c.MQ.TLS
	if !c.MQ.External {
		isFloatIP, _ := sshutils.IsFloatIP(c.SSHConfig, ip)
		if isFloatIP {
			// if user specify a float ip,we replace to listen 0.0.0.0
			data["MQServerAddress"] = "0.0.0.0"
		} else {
			data["MQServerAddress"] = ip
		}
		data["MQServerPort"] = c.MQ.Port
		data["MQClusterPort"] = c.MQ.ClusterPort
		data["LeaderHost"] = fmt.Sprintf("%s:%d", c.ServerIPs[0], c.MQ.ClusterPort)
		if c.MQ.TLS {
			data["MQServerCertPath"] = filepath.Join(DefaultKcServerConfigPath, DefaultNatsPKIPath, fmt.Sprintf("%s.crt", NatsIOServer))
			data["MQServerKeyPath"] = filepath.Join(DefaultKcServerConfigPath, DefaultNatsPKIPath, fmt.Sprintf("%s.key", NatsIOServer))
		}
	}

	if c.MQ.TLS {
		data["MQCaPath"] = c.MQ.CA
		data["MQClientCertPath"] = c.MQ.ClientCert
		data["MQClientKeyPath"] = c.MQ.ClientKey
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("template execute failed: %s", err.Error())
	}
	return buffer.String(), nil
}

func (c *DeployConfig) GetKcAgentConfigTemplateContent(metadata Metadata) (string, error) {
	tmpl, err := template.New("text").Parse(config.KcAgentConfigTmpl)
	if err != nil {
		return "", fmt.Errorf("template parse failed: %s", err.Error())
	}
	var mqServerEndpoints []string
	for _, v := range c.MQ.IPs {
		mqServerEndpoints = append(mqServerEndpoints, fmt.Sprintf("%s:%d", v, c.MQ.Port))
	}

	var data = make(map[string]interface{})
	data["AgentID"] = metadata.AgentID
	data["Region"] = metadata.Region
	data["FloatIP"] = metadata.FloatIP
	data["IPDetect"] = c.IPDetect
	data["NodeIPDetect"] = c.NodeIPDetect
	data["StaticServerAddress"] = fmt.Sprintf("http://%s:%d", c.ServerIPs[0], c.StaticServerPort)
	if c.Debug {
		data["LogLevel"] = "debug"
	} else {
		data["LogLevel"] = "info"
	}
	data["MQServerEndpoints"] = mqServerEndpoints
	data["MQAuthToken"] = c.MQ.Secret
	data["MQExternal"] = c.MQ.External
	data["MQUser"] = c.MQ.User
	data["MQAuthToken"] = c.MQ.Secret
	data["MQTLS"] = c.MQ.TLS
	if c.MQ.TLS {
		data["MQCaPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultCaPath, filepath.Base(c.MQ.CA))
		data["MQClientCertPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath, filepath.Base(c.MQ.ClientCert))
		data["MQClientKeyPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath, filepath.Base(c.MQ.ClientKey))
	}
	data["OpLogDir"] = c.OpLog.Dir
	data["OpLogThreshold"] = c.OpLog.Threshold
	data["KcImageRepoMirror"] = c.ImageProxy.KcImageRepoMirror
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("template execute failed: %s", err.Error())
	}
	return buffer.String(), nil
}
