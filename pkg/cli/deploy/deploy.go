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
	"slices"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
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
	deployExamplePkg              = constatns.KubeClipperReleaseBaseURL + "/v1.4.0/kc-amd64.tar.gz"
	kcServerClientIdentity        = "system:kc-server"
	serviceHealthCheckTimeout     = 5 * time.Second
	authenticationJWTSecretLength = 24
	authenticationJWTSecretDigits = 5
	longDescription               = `
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
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3 \
    --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `

  # Deploy env with many agent node in same region.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:192.168.10.123,192.168.10.124 \
    --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `

  # Deploy env with many agent node in different region.
  kcctl deploy --server 192.168.234.3 \
    --agent us-west-1:1.1.1.1,1.1.1.2 --agent us-west-2:1.1.1.3 \
    --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `

  # Deploy env with many agent node which has orderly ip.
  # this will add 10 agent,1.1.1.1, 1.1.1.2, ... 1.1.1.10.
  kcctl deploy --server 192.168.234.3 --agent us-west-1:1.1.1.1-1.1.1.10 \
    --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `
  
  # Deploy env with many agent nodes and specify ip detect method for these nodes
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3,192.168.234.4 \
    --ip-detect=interface=eth0 --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `

  # Deploy env with many agent nodes and specify node ip detect method for these nodes, used for routing between nodes in the kubernetes cluster
  kcctl deploy --server 192.168.234.3 --agent 192.168.234.3,192.168.234.4 \
    --node-ip-detect=interface=eth1 --pk-file ~/.ssh/id_rsa \
    --pkg ` + deployExamplePkg + `

  # Deploy from config.
  kcctl deploy --deploy-config deploy-config.yaml
  # Deploy and config fip to agent node.
  kcctl deploy --server 172.20.149.198 --agent us-west-1:10.0.0.10 --agent us-west-2:20.0.0.11 --fip 10.0.0.10:172.20.149.199 --fip 20.0.0.11:172.20.149.200

  Please read 'kcctl deploy -h' get more deploy flags`
	defaultPkg = constatns.KubeClipperReleaseBaseURL + "/%s/kc-%s.tar.gz"
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

func (d *DeployOptions) nodeRole(ip string) string {
	isServer := slices.Contains(d.deployConfig.ServerIPs, ip)
	_, isAgent := d.deployConfig.Agents[ip]
	switch {
	case isServer && isAgent:
		return "server+agent"
	case isServer:
		return "server"
	case isAgent:
		return "agent"
	default:
		return ""
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
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.ValidateArgs(); err != nil {
				return err
			}
			o.preRun()
			if err := o.preCheck(); err != nil {
				return &deployPrecheckError{cause: err}
			}
			return o.RunDeploy()
		},
		Args: cobra.NoArgs,
	}

	cmd.Flags().StringArrayVar(&o.agents, "agent", o.agents, "Kc agent region and ips.")
	cmd.Flags().StringArrayVar(&o.fips, "float-ip", o.fips, "Kc agent ip and float ip.")
	auth := o.deployConfig.AuthenticationOpts
	flags := cmd.Flags()
	flags.IntVar(&auth.AuthenticateRateLimiterMaxTries, "authenticate-rate-limiter-max-retries", auth.AuthenticateRateLimiterMaxTries,
		"maximum number of retry times within the valid period")
	flags.DurationVar(&auth.AuthenticateRateLimiterDuration, "authenticate-rate-limiter-duration", auth.AuthenticateRateLimiterDuration,
		"specifies the lock duration of the user")
	flags.DurationVar(&auth.LoginHistoryRetentionPeriod, "login-history-retention-period", auth.LoginHistoryRetentionPeriod,
		"login-history-retention-period defines how long login history should be kept.")
	flags.IntVar(&auth.LoginHistoryMaximumEntries, "login-history-maximum-entries", auth.LoginHistoryMaximumEntries,
		"login-history-maximum-entries defines how many entries of login history should be kept.")
	flags.StringVar(&auth.InitialPassword, "initial-password", auth.InitialPassword, "admin user password")
	o.deployConfig.AddFlags(cmd.Flags())
	o.deployConfig.AuditOpts.AddFlags(cmd.Flags())

	cmd.AddCommand(NewCmdDeployConfig(o))

	return cmd
}

func (d *DeployOptions) Complete() error {
	if err := d.deployConfig.Complete(); err != nil {
		return err
	}
	if err := d.generateAuthenticationJWTSecret(); err != nil {
		return err
	}
	if d.deployConfig.Pkg == "" {
		v := os.Getenv("KC_VERSION")
		var ok bool
		if v == "" {
			v, ok = strutil.ParseGitDescribeInfo(version.Get().GitVersion)
			if !ok {
				v = "v1.6.0"
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
		agents, err := BuildAgent(d.agents, d.fips, d.deployConfig.DefaultRegion)
		if err != nil {
			return err
		}
		d.deployConfig.Agents = agents
	}

	d.allNodes = sets.NewString().
		Insert(d.deployConfig.ServerIPs...).
		Insert(d.deployConfig.Agents.ListIP()...).
		List()

	if d.deployConfig.NodeIPDetect == "" {
		logger.Infof("node-ip-detect inherits from ip-detect: %s", d.deployConfig.IPDetect)
		d.deployConfig.NodeIPDetect = d.deployConfig.IPDetect
	}

	if d.aio {
		logger.Infof("run in aio mode.")
	}

	return nil
}

func (d *DeployOptions) generateAuthenticationJWTSecret() error {
	secret, err := password.Generate(authenticationJWTSecretLength, authenticationJWTSecretDigits, 0, false, true)
	if err != nil {
		return fmt.Errorf("generate authentication JWT secret: %w", err)
	}
	d.deployConfig.AuthenticationOpts.JwtSecret = secret
	return nil
}

func (d *DeployOptions) ValidateArgs() error {
	if !d.deployConfig.TLS {
		return fmt.Errorf("operation v2 requires TLS because kc-agent communicates with kc-server over mTLS")
	}
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
	if d.deployConfig.TempDir == "" {
		d.deployConfig.TempDir = config.DefaultPkgPath
	}
	if !filepath.IsAbs(d.deployConfig.TempDir) {
		return fmt.Errorf("temp directory must be an absolute path")
	}
	if filepath.Clean(d.deployConfig.TempDir) == string(filepath.Separator) {
		return fmt.Errorf("temp directory must not be the filesystem root")
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
	return nil
}

func (d *DeployOptions) preRun() {
	for agent, metadata := range d.deployConfig.Agents {
		if metadata.AgentID == "" {
			metadata.AgentID = uuid.New().String()
			d.deployConfig.Agents[agent] = metadata
		}
	}
	for _, sip := range d.deployConfig.ServerIPs {
		hostname, err := sshutils.GetRemoteHostName(d.deployConfig.SSHConfig, sip)
		if err != nil {
			logger.Fatalf("get remote hostname failed,err:%v", err)
		}
		d.servers[sip] = hostname
	}
	d.dumpConfig()
}

type precheckFunc func(sshConfig *sshutils.SSH, host string) error

const (
	timeSyncPrecheckCommand = `for service in chrony chronyd ntp ntpd systemd-timesyncd; ` +
		`do systemctl is-active --quiet "$service" && exit 0; done; exit 10`
	timeSyncServiceMissingExitCode = 10
)

var (
	precheckKcEtcdFunc                = generateCommonPreCheckFunc("kc-etcd")
	precheckKcServerFunc              = generateCommonPreCheckFunc("kc-server")
	precheckKcAgentFunc               = generateCommonPreCheckFunc("kc-agent")
	precheckNtpFunc      precheckFunc = func(sshConfig *sshutils.SSH, host string) error {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, timeSyncPrecheckCommand)
		if ret.ExitCode == timeSyncServiceMissingExitCode {
			return fmt.Errorf("no supported time synchronization service is running (chrony, ntp, or systemd-timesyncd)")
		}
		return err
	}
)

func PrecheckPortFunc(port int, serviceName string) precheckFunc {
	return func(sshConfig *sshutils.SSH, host string) error {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, "ss -tlnp")
		if err != nil {
			ret, err = sshutils.SSHCmdWithSudo(sshConfig, host, "netstat -tlnp")
			if err != nil {
				return fmt.Errorf("check port %d failed: %w", port, err)
			}
		}
		output := ret.StdoutToString("")
		if strings.Contains(output, fmt.Sprintf(":%d ", port)) ||
			strings.Contains(output, fmt.Sprintf(":%d\t", port)) {
			return fmt.Errorf("port %d is already in use, required by %s", port, serviceName)
		}
		return nil
	}
}

func generateCommonPreCheckFunc(name string) precheckFunc {
	return func(sshConfig *sshutils.SSH, host string) error {
		command := fmt.Sprintf(
			"systemctl show --no-pager --property=LoadState --property=ActiveState "+
				"--property=SubState --value %s.service", name)
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, command)
		logger.V(2).Infof("exit code %d, err %v", ret.ExitCode, err)
		if err != nil {
			return err
		}
		state, err := parseSystemdUnitState(ret.Stdout)
		if err != nil {
			return fmt.Errorf("read %s.service state: %w", name, err)
		}
		if state.load != "not-found" {
			return &existingServiceError{name: name, state: state}
		}
		return nil
	}
}

type systemdUnitState struct {
	load   string
	active string
	sub    string
}

type existingServiceError struct {
	name  string
	state systemdUnitState
}

func (e *existingServiceError) Error() string {
	return fmt.Sprintf("%s.service already exists", e.name)
}

type precheckFailure struct {
	check string
	node  string
	cause error
}

func (e *precheckFailure) Error() string {
	if serviceErr, ok := e.cause.(*existingServiceError); ok {
		return fmt.Sprintf("check: %s\nnode: %s\nreason: %s.service already exists\nstate: load=%s, active=%s, sub=%s",
			e.check, e.node, serviceErr.name, serviceErr.state.load, serviceErr.state.active, serviceErr.state.sub)
	}
	return fmt.Sprintf("check: %s\nnode: %s\nreason: %v", e.check, e.node, e.cause)
}

func (e *precheckFailure) Unwrap() error { return e.cause }

type deployPrecheckError struct {
	cause error
}

func (e *deployPrecheckError) Error() string {
	message := "deploy precheck failed:\n  " + strings.ReplaceAll(e.cause.Error(), "\n", "\n  ")
	if failure, ok := e.cause.(*precheckFailure); ok {
		if _, ok := failure.cause.(*existingServiceError); ok {
			return message + "\nclean old environment before deploying"
		}
	}
	return message
}

func (e *deployPrecheckError) Unwrap() error { return e.cause }

func parseSystemdUnitState(output string) (systemdUnitState, error) {
	values := strings.Split(strings.TrimSpace(output), "\n")
	if len(values) != 3 {
		return systemdUnitState{}, fmt.Errorf("expected load, active, and sub state, got %q", strings.TrimSpace(output))
	}
	return systemdUnitState{
		load:   values[0],
		active: values[1],
		sub:    values[2],
	}, nil
}

func (d *DeployOptions) precheckService(name string, nodes []string, fn precheckFunc) error {
	logger.Infof("============>%s PRECHECK ...", name)
	errs := make([]error, len(nodes))
	wg := sync.WaitGroup{}
	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, host string) {
			defer wg.Done()
			if err := fn(d.deployConfig.SSHConfig, host); err != nil {
				errs[idx] = err
			}
		}(i, node)
	}
	wg.Wait()

	hasError := false
	for _, err := range errs {
		if err != nil {
			hasError = true
			break
		}
	}
	if !hasError {
		logger.Infof("============>%s PRECHECK OK!", name)
		return nil
	}

	groups := make(map[string][]string)
	var msgOrder []string
	for i, err := range errs {
		if err == nil {
			continue
		}
		msg := err.Error()
		host := nodes[i]
		role := d.nodeRole(host)
		ref := fmt.Sprintf("[%s@%s]", role, host)
		if role == "" {
			ref = fmt.Sprintf("[%s]", host)
		}
		if _, exists := groups[msg]; !exists {
			msgOrder = append(msgOrder, msg)
		}
		groups[msg] = append(groups[msg], ref)
	}
	for _, msg := range msgOrder {
		refs := groups[msg]
		logger.Warnf("%s:", msg)
		for _, ref := range refs {
			logger.Warnf("  - %s", ref)
		}
	}
	for i, err := range errs {
		if err == nil {
			continue
		}
		logger.Errorf("===========>%s PRECHECK FAILED!", name)
		return &precheckFailure{check: name, node: d.nodeReference(nodes, i), cause: err}
	}
	return fmt.Errorf("%s precheck failed", name)
}

func (d *DeployOptions) nodeReference(nodes []string, index int) string {
	if index < 0 || index >= len(nodes) {
		return "[unknown node]"
	}
	host := nodes[index]
	role := d.nodeRole(host)
	if role == "" {
		return fmt.Sprintf("[%s]", host)
	}
	return fmt.Sprintf("[%s@%s]", role, host)
}

func (d *DeployOptions) precheckTimeLag() error {
	logger.Infof("============>TIME-LAG PRECHECK ...")
	type timeLagEntry struct {
		lag float64
		err error
	}
	entries := make([]timeLagEntry, len(d.allNodes))
	wg := sync.WaitGroup{}
	now := time.Now()
	logger.Infof("BaseLine Time: %s", now.Format(time.RFC3339))
	for i, node := range d.allNodes {
		wg.Add(1)
		go func(idx int, host string) {
			defer wg.Done()
			ret, err := sshutils.SSHCmd(d.deployConfig.SSHConfig, host, "date +%s")
			if err != nil {
				logger.Errorf("get timestamp from %s failed: %s", host, err.Error())
				entries[idx] = timeLagEntry{err: fmt.Errorf("get timestamp: %w", err)}
				return
			}
			output := ret.StdoutToString("")
			ts, err := strconv.ParseInt(output, 10, 64)
			if err != nil {
				logger.Errorf("parse timestamp from %s failed: %s", host, err.Error())
				entries[idx] = timeLagEntry{err: fmt.Errorf("parse timestamp %q: %w", output, err)}
				return
			}
			t := time.Unix(ts, 0)
			diff := t.Sub(now).Seconds()
			logger.Infof("[%s] %v seconds", host, diff)
			if math.Abs(diff) > float64(5) {
				entries[idx] = timeLagEntry{lag: diff}
			}
		}(i, node)
	}
	wg.Wait()

	var failures []string
	for i, entry := range entries {
		if entry.err == nil && entry.lag == 0 {
			continue
		}
		role := d.nodeRole(d.allNodes[i])
		ref := fmt.Sprintf("[%s@%s]", role, d.allNodes[i])
		if role == "" {
			ref = fmt.Sprintf("[%s]", d.allNodes[i])
		}
		if entry.err != nil {
			failures = append(failures, fmt.Sprintf("%s %v", ref, entry.err))
			continue
		}
		failures = append(failures, fmt.Sprintf("%s time lag %.1fs exceeds 5s", ref, entry.lag))
	}
	if len(failures) == 0 {
		logger.Infof("all nodes time lag less then 5 seconds")
		logger.Infof("============>TIME-LAG PRECHECK OK!")
		return nil
	}
	logger.Warnf("time lag exceeds 5s threshold:")
	for _, failure := range failures {
		logger.Warnf("  - %s", failure)
	}
	logger.Errorf("===========>TIME-LAG PRECHECK FAILED!")
	return fmt.Errorf("TIME-LAG precheck failed: %s", strings.Join(failures, "; "))
}

func (d *DeployOptions) precheckPorts() error {
	toolCheck := func(sshConfig *sshutils.SSH, host string) error {
		_, err := sshutils.SSHCmdWithSudo(sshConfig, host, "which ss")
		if err != nil {
			_, err = sshutils.SSHCmdWithSudo(sshConfig, host, "which netstat")
			if err != nil {
				return fmt.Errorf("port precheck requires ss or netstat: %w", err)
			}
		}
		return nil
	}
	if err := d.precheckService("PORT-TOOL", d.allNodes, toolCheck); err != nil {
		return err
	}

	serverPorts := []struct {
		port int
		name string
	}{
		{d.deployConfig.EtcdConfig.ClientPort, "kc-etcd-client"},
		{d.deployConfig.EtcdConfig.PeerPort, "kc-etcd-peer"},
		{d.deployConfig.EtcdConfig.MetricsPort, "kc-etcd-metrics"},
		{d.deployConfig.ServerPort, "kc-server"},
		{d.deployConfig.StaticServerPort, "kc-server-static"},
		{d.deployConfig.ConsolePort, "kc-console"},
	}
	for _, p := range serverPorts {
		if err := d.precheckService(
			fmt.Sprintf("PORT-%d(%s)", p.port, p.name),
			d.deployConfig.ServerIPs,
			PrecheckPortFunc(p.port, p.name),
		); err != nil {
			return err
		}
	}
	// Only configured Agent nodes run kc-agent. A server-only node can use this
	// port for an unrelated service.
	return d.precheckService(
		"PORT(kc-agent-log)",
		d.deployConfig.Agents.ListIP(),
		func(sshConfig *sshutils.SSH, host string) error {
			logPort, err := d.deployConfig.Agents[host].LogPort()
			if err != nil {
				return err
			}
			return PrecheckPortFunc(logPort, "kc-agent-log")(sshConfig, host)
		},
	)
}

func (d *DeployOptions) preCheck() error {
	if err := d.precheckService("kc-etcd", d.deployConfig.ServerIPs, precheckKcEtcdFunc); err != nil {
		return err
	}
	if err := d.precheckService("kc-server", d.deployConfig.ServerIPs, precheckKcServerFunc); err != nil {
		return err
	}
	if err := d.precheckService("kc-agent", d.allNodes, precheckKcAgentFunc); err != nil {
		return err
	}
	if err := d.precheckTimeLag(); err != nil {
		return err
	}
	if err := d.precheckPorts(); err != nil {
		return err
	}
	if err := d.precheckService("NTP", d.allNodes, precheckNtpFunc); err != nil {
		return err
	}
	if err := sudo.PreCheckError("sudo", d.deployConfig.SSHConfig, d.IOStreams, d.allNodes); err != nil {
		return err
	}
	if err := sudo.MultiNICError(
		"ipDetect", d.deployConfig.SSHConfig, d.IOStreams,
		d.deployConfig.Agents.ListIP(), d.deployConfig.IPDetect,
	); err != nil {
		return err
	}
	return nil
}

func (d *DeployOptions) RunDeploy() error {
	if err := d.generateAndSendCerts(); err != nil {
		return err
	}
	logger.Infof("------ Send packages ------")
	d.sendPackage()
	logger.Infof("------ Install kc-etcd ------")
	d.deployEtcd()
	if err := d.waitEtcdReady(); err != nil {
		return err
	}
	logger.Infof("------ Install kc-server ------")
	d.deployKcServer()
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
	if err := writeLocalDeployConfig(d.deployConfig); err != nil {
		return fmt.Errorf("sync local deploy config: %w", err)
	}
	fmt.Printf("\033[1;40;36m%s\033[0m\n", options.Contact)
	return nil
}

func (d *DeployOptions) sendPackage() {
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s", filepath.Join(d.deployConfig.TempDir, "kc"),
		filepath.Join(d.deployConfig.TempDir, path.Base(d.deployConfig.Pkg)), d.deployConfig.TempDir)
	cp := sshutils.WrapSh(fmt.Sprintf("cp -rf %s /usr/local/bin/", filepath.Join(d.deployConfig.TempDir, "kc", "bin", "*")))
	mkdir := "mkdir -p /usr/lib/systemd/system"
	// rm -rf /root/kc && tar -xvf /root/kc/pkg/kc.tar -C ~/kc/pkg && /bin/bash -c 'cp -rf /root/kc/pkg/kc/bin/* /usr/local/bin/' && mkdir -p /usr/lib/systemd/system
	hook := sshutils.Combine([]string{tar, cp, mkdir})
	err := utils.SendPackageWithTempDir(
		d.deployConfig.SSHConfig, d.deployConfig.Pkg, d.allNodes, d.deployConfig.TempDir, nil, &hook, d.deployConfig.TempDir,
	)
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
	agentCerts := make(map[string]certutils.Config, len(d.deployConfig.Agents))

	kcctlCommonNameUsages := make(map[string][]x509.ExtKeyUsage)
	kcctlCommonNameUsages[options.AdminKcctlCert] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	kcctlCert := clientCertList(
		options.DefaultKcctlPKIPath,
		options.Ca,
		append(altNames, d.deployConfig.Agents.ListIP()...),
		[]string{user.KCCTL},
		kcctlCommonNameUsages,
	)
	certs = append(certs, kcctlCert...)
	for agentIP, metadata := range d.deployConfig.Agents {
		altNames := certutils.AltNames{DNSNames: map[string]string{metadata.AgentID: metadata.AgentID}, IPs: map[string]net.IP{}}
		if ip := net.ParseIP(agentIP); ip != nil {
			altNames.IPs[ip.String()] = ip
		}
		cert := certutils.Config{
			Path: filepath.Join(options.HomeDIR, options.DefaultPath, "pki", "agents", metadata.AgentID), BaseName: "agent",
			CAName: options.Ca, CommonName: "system:kc-agent:" + metadata.AgentID,
			Organization: []string{"system:kc-agents"}, Year: 100, AltNames: altNames,
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		}
		agentCerts[agentIP] = cert
		certs = append(certs, cert)
	}

	etcdCommonNameUsages := make(map[string][]x509.ExtKeyUsage)
	etcdCommonNameUsages[options.EtcdServer] = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdPeer] = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdKcClient] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	etcdCommonNameUsages[options.EtcdHealthCheck] = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	etcdCert := certList(options.DefaultEtcdPKIPath, options.Ca, append(altNames, d.deployConfig.ServerIPs...), etcdCommonNameUsages)
	certs = append(certs, etcdCert...)

	var kcCerts []certutils.Config
	if d.deployConfig.TLS {
		nameUsages := map[string][]x509.ExtKeyUsage{
			options.KCServer: {x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		}
		names := append(altNames, d.deployConfig.ServerIPs...)
		names = append(names, options.KCServerAltName)
		kcCerts = kcServerCertList(names, nameUsages)
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
	for agentIP := range agentCerts {
		cert := agentCerts[agentIP]
		if err := d.sendAgentIdentity(agentIP, &cert, &cas[0]); err != nil {
			return err
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

type etcdHealthClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Close() error
}

func (d *DeployOptions) waitEtcdReady() error {
	tlsConfig, err := d.etcdHealthTLSConfig()
	if err != nil {
		return fmt.Errorf("load etcd health check credentials: %w", err)
	}

	endpoints := d.etcdEndpoints()
	clients, err := newEtcdHealthClients(endpoints, tlsConfig)
	if err != nil {
		return fmt.Errorf("create etcd health check clients: %w", err)
	}
	defer closeEtcdHealthClients(clients)

	ctx, cancel := context.WithTimeout(context.Background(), d.deployConfig.KCServerHealthCheckTimeout)
	defer cancel()
	if err := retryFunc(ctx, 3*time.Second, "waitEtcdReady", "cluster", func(string) error {
		return checkEtcdEndpoints(ctx, clients, endpoints)
	}); err != nil {
		return fmt.Errorf("etcd cluster is not ready: %w", err)
	}
	return nil
}

func (d *DeployOptions) etcdEndpoints() []string {
	endpoints := make([]string, 0, len(d.deployConfig.ServerIPs))
	for _, host := range d.deployConfig.ServerIPs {
		endpoints = append(endpoints, net.JoinHostPort(host, strconv.Itoa(d.deployConfig.EtcdConfig.ClientPort)))
	}
	return endpoints
}

func (*DeployOptions) etcdHealthTLSConfig() (*tls.Config, error) {
	basePath := filepath.Join(options.HomeDIR, options.DefaultPath)
	clientCert, err := tls.LoadX509KeyPair(
		filepath.Join(basePath, options.DefaultEtcdPKIPath, options.EtcdHealthCheck+".crt"),
		filepath.Join(basePath, options.DefaultEtcdPKIPath, options.EtcdHealthCheck+".key"),
	)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(filepath.Join(basePath, options.DefaultCaPath, options.Ca+".crt"))
	if err != nil {
		return nil, err
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("parse etcd CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func newEtcdHealthClients(endpoints []string, tlsConfig *tls.Config) ([]etcdHealthClient, error) {
	clients := make([]etcdHealthClient, 0, len(endpoints))
	for _, endpoint := range endpoints {
		client, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{endpoint},
			DialTimeout: serviceHealthCheckTimeout,
			TLS:         tlsConfig,
		})
		if err != nil {
			closeEtcdHealthClients(clients)
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func closeEtcdHealthClients(clients []etcdHealthClient) {
	for _, client := range clients {
		_ = client.Close()
	}
}

func checkEtcdEndpoints(ctx context.Context, clients []etcdHealthClient, endpoints []string) error {
	if len(clients) != len(endpoints) {
		return fmt.Errorf("etcd health client count %d does not match endpoint count %d", len(clients), len(endpoints))
	}
	for i, endpoint := range endpoints {
		requestCtx, cancel := context.WithTimeout(ctx, serviceHealthCheckTimeout)
		_, err := clients[i].Get(requestCtx, "health")
		cancel()
		if err != nil {
			return fmt.Errorf("endpoint %s is unhealthy: %w", endpoint, err)
		}
	}
	return nil
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
	data["ServerCertPath"] = filepath.Join(
		options.DefaultKcServerConfigPath,
		options.DefaultEtcdPKIPath,
		fmt.Sprintf("%s.crt", options.EtcdServer),
	)
	data["DataDIR"] = d.deployConfig.EtcdConfig.DataDir
	data["PeerAddress"] = fmt.Sprintf("%s:%d", ip, d.deployConfig.EtcdConfig.PeerPort)
	data["InitialCluster"] = strings.Join(initialCluster, ",")
	data["ClusterToken"] = "kc-etcd-cluster"
	data["ServerCertKeyPath"] = filepath.Join(
		options.DefaultKcServerConfigPath,
		options.DefaultEtcdPKIPath,
		fmt.Sprintf("%s.key", options.EtcdServer),
	)
	if isFloatIP {
		// if user specify a float ip,we replace to listen 0.0.0.0
		data["PeerURLs"] = fmt.Sprintf("https://0.0.0.0:%d", d.deployConfig.EtcdConfig.PeerPort)
		data["ClientURLs"] = fmt.Sprintf("https://0.0.0.0:%d", d.deployConfig.EtcdConfig.ClientPort)
	} else {
		data["ClientURLs"] = fmt.Sprintf("https://127.0.0.1:%d,https://%s:%d", d.deployConfig.EtcdConfig.ClientPort, ip, d.deployConfig.EtcdConfig.ClientPort)
		data["PeerURLs"] = fmt.Sprintf("https://%s:%d", ip, d.deployConfig.EtcdConfig.PeerPort)
	}
	data["MetricsURLs"] = fmt.Sprintf("http://127.0.0.1:%d", d.deployConfig.EtcdConfig.MetricsPort)
	data["PeerCertPath"] = filepath.Join(
		options.DefaultKcServerConfigPath,
		options.DefaultEtcdPKIPath,
		fmt.Sprintf("%s.crt", options.EtcdPeer),
	)
	data["PeerCertKeyPath"] = filepath.Join(
		options.DefaultKcServerConfigPath,
		options.DefaultEtcdPKIPath,
		fmt.Sprintf("%s.key", options.EtcdPeer),
	)
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
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/resource/* %s/", d.deployConfig.TempDir, d.deployConfig.StaticServerPath)),
		sshutils.WrapSh(fmt.Sprintf("cp -rf %s/kc/bin/* %s/kc/", d.deployConfig.TempDir, d.deployConfig.StaticServerPath)),
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
	client := &http.Client{Transport: tr, Timeout: serviceHealthCheckTimeout}

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
	ticker := time.NewTicker(intervalTime)
	defer ticker.Stop()

	for {
		err := fn(host)
		if err == nil {
			return nil
		}
		logger.Infof("function '%s' running error: %s. about to enter retry", funcName, err.Error())

		select {
		case <-ctx.Done():
			logger.Warnf("retry function '%s' timeout...", funcName)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (d *DeployOptions) deployKcConsole() {
	data := d.getKcConsoleTemplateContent()

	cmdList := []string{
		fmt.Sprintf("mkdir -pv /etc/kc-console && cp -rf %s/kc/kc-console /etc/kc-console/dist", d.deployConfig.TempDir),
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

func (d *DeployOptions) sendAgentIdentity(agentIP string, cert, ca *certutils.Config) error {
	destination := filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultAgentPKIPath)
	sources := []string{
		path.Join(cert.Path, cert.BaseName+".crt"),
		path.Join(cert.Path, cert.BaseName+".key"),
		path.Join(ca.Path, ca.BaseName+".crt"),
	}
	for _, source := range sources {
		if err := utils.SendPackageV2WithTempDir(
			d.deployConfig.SSHConfig, source, []string{agentIP}, destination, nil, nil, d.deployConfig.TempDir,
		); err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) removeTempFile() {
	cmdList := []string{
		fmt.Sprintf("rm -rf %s/kc", d.deployConfig.TempDir),
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
		err := utils.SendPackageV2WithTempDir(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".key"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcServerConfigPath, pki), nil, nil, d.deployConfig.TempDir)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2WithTempDir(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcServerConfigPath, pki), nil, nil, d.deployConfig.TempDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) sendAgentCertAndKey(contents []certutils.Config, pki string) error {
	for _, content := range contents {
		err := utils.SendPackageV2WithTempDir(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".key"),
			d.deployConfig.Agents.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil, d.deployConfig.TempDir)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2WithTempDir(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.Agents.ListIP(),
			filepath.Join(options.DefaultKcAgentConfigPath, pki), nil, nil, d.deployConfig.TempDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DeployOptions) sendConsoleCert(contents []certutils.Config, pki string) error {
	for _, content := range contents {
		err := utils.SendPackageV2WithTempDir(d.deployConfig.SSHConfig,
			path.Join(content.Path, content.BaseName+".crt"),
			d.deployConfig.ServerIPs,
			filepath.Join(options.DefaultKcConsoleConfigPath, pki), nil, nil, d.deployConfig.TempDir)
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
				ClientCertificate: path.Join(
					homedir.HomeDir(),
					options.DefaultPath,
					options.DefaultKcctlPKIPath,
					options.AdminKcctlCert+".crt",
				),
				ClientKey: path.Join(
					homedir.HomeDir(),
					options.DefaultPath,
					options.DefaultKcctlPKIPath,
					options.AdminKcctlCert+".key",
				),
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
	afterHook := fmt.Sprintf(
		"mv %s %s/admin.conf",
		path.Join(options.DefaultKcServerConfigPath, options.DefaultConfig),
		options.DefaultKcServerConfigPath,
	)
	err := utils.SendPackageWithTempDir(d.deployConfig.SSHConfig,
		options.DefaultConfigPath,
		d.deployConfig.ServerIPs,
		options.DefaultKcServerConfigPath, nil, &afterHook, d.deployConfig.TempDir)
	return err
}

func createOrUpdateConfigMap(client *kc.Client, cm *v1.ConfigMap) {
	existing, err := client.DescribeConfigMap(context.TODO(), cm.Name)
	if err == nil && len(existing.Items) > 0 {
		existingCM := existing.Items[0]
		existingCM.Data = cm.Data
		if _, err = client.UpdateConfigMap(context.TODO(), &existingCM); err != nil {
			logger.Fatalf("update configmap %s failed: %v", cm.Name, err)
		}
		return
	}
	if _, err = client.CreateConfigMap(context.TODO(), cm); err != nil {
		logger.Fatalf("create configmap %s failed: %v", cm.Name, err)
	}
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
	createOrUpdateConfigMap(client, dc)
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
	createOrUpdateConfigMap(client, cacm)

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
	etcdhealthcheckkey, err := os.ReadFile(fmt.Sprintf("%s/kube-etcd-healthcheck-client.key", etcdPath))
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
	createOrUpdateConfigMap(client, etcdcm)

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

func kcServerCertList(altNames []string, commonNameUsage map[string][]x509.ExtKeyUsage) []certutils.Config {
	certs := certList(options.DefaultKCPKIPath, options.Ca, altNames, commonNameUsage)
	for i := range certs {
		if certs[i].BaseName == options.KCServer {
			certs[i].CommonName = kcServerClientIdentity
		}
	}
	return certs
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
