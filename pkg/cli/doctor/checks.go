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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	rpctypes "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	etcdRequestTimeout      = 10 * time.Second
	etcdEndpointHealthCheck = "endpoint-health"
)

type etcdClient interface {
	Get(context.Context, string, ...clientv3.OpOption) (*clientv3.GetResponse, error)
	MemberList(context.Context) (*clientv3.MemberListResponse, error)
	Status(context.Context, string) (*clientv3.StatusResponse, error)
	Close() error
}

type etcdClientFactory func([]string, *tls.Config) (etcdClient, error)

func checkKCServer(_ context.Context, state *diagnosticState) Component {
	component := Component{Name: "kc-server"}
	statusComponent := platformComponent(state.platform, "kc-server")
	for _, name := range []string{"api", "controller-manager", "static-resource"} {
		check := platformCheck(statusComponent, name)
		if check == nil {
			status := platformstatus.Unknown
			message := fmt.Sprintf("%s status is unavailable", name)
			if state.platform == nil {
				status = platformstatus.Skipped
				message = fmt.Sprintf("%s check skipped because platform status is unavailable", name)
			}
			component.Checks = append(component.Checks, Check{
				Name: name, Target: platformTarget, Status: status, Message: message,
			})
			continue
		}
		component.Checks = append(component.Checks, Check{
			Name: name, Target: platformTarget, Status: check.Status, Message: check.Message,
		})
	}

	if state.deployConfig == nil || state.remote == nil {
		component.Checks = append(component.Checks, Check{
			Name: "remote-instances", Status: platformstatus.Skipped,
			Message: "server SSH checks skipped because deployment configuration is unavailable",
		})
		component.Message = statusMessage(statusComponent, "server status unavailable")
		return component
	}

	servers := sortedStrings(state.deployConfig.ServerIPs)
	checks := runParallel(servers, func(host string) []Check {
		return checkServerNode(state, host)
	})
	component.Checks = append(component.Checks, checks...)
	attachServerFailureDetails(state, &component, servers)
	component.Message = serverSummary(servers, checks, statusComponent)
	return component
}

func checkServerNode(state *diagnosticState, host string) []Check {
	service := state.remote.service(host, "kc-server")
	checks := []Check{service}
	if service.Status == platformstatus.Unhealthy || service.Status == platformstatus.Unknown {
		return checks
	}

	scheme := "http"
	if state.deployConfig.TLS {
		scheme = "https"
	}
	checks = append(checks,
		state.remote.httpHealth(host, "api-health", fmt.Sprintf("%s://%s/healthz", scheme, endpoint(host, state.deployConfig.ServerPort))),
		state.remote.httpHealth(host, "static-resource-health",
			fmt.Sprintf("http://%s/healthz", endpoint(host, state.deployConfig.StaticServerPort))),
	)
	for i := range checks {
		if checks[i].Status != platformstatus.Healthy && len(checks[i].Logs) == 0 {
			checks[i].Logs = state.remote.journal(host, "kc-server")
			checks[i].Commands = appendUnique(checks[i].Commands, state.remote.serviceCommands(host, "kc-server")...)
		}
	}
	return checks
}

func checkKCEtcd(ctx context.Context, state *diagnosticState) Component {
	component := Component{Name: "kc-etcd"}
	statusComponent := platformComponent(state.platform, "kc-etcd")
	if statusComponent != nil {
		component.Checks = append(component.Checks, Check{
			Name: platformStatus, Target: platformTarget,
			Status: statusComponent.Status, Message: statusComponent.Message,
		})
	} else {
		component.Checks = append(component.Checks, Check{
			Name: platformStatus, Target: platformTarget,
			Status: platformstatus.Skipped, Message: "etcd platform status check skipped",
		})
	}
	if state.deployConfig == nil || state.remote == nil {
		component.Checks = append(component.Checks, Check{
			Name: "remote-members", Status: platformstatus.Skipped,
			Message: "etcd SSH checks skipped because deployment configuration is unavailable",
		})
		component.Message = statusMessage(statusComponent, "etcd status unavailable")
		return component
	}

	servers := sortedStrings(state.deployConfig.ServerIPs)
	serviceChecks := runParallel(servers, func(host string) []Check {
		return []Check{state.remote.service(host, "kc-etcd")}
	})
	component.Checks = append(component.Checks, serviceChecks...)
	healthyHost := ""
	for i := range serviceChecks {
		if serviceChecks[i].Status == platformstatus.Healthy {
			healthyHost = serviceChecks[i].Target
			break
		}
	}
	if healthyHost == "" {
		component.Checks = append(component.Checks, Check{
			Name: etcdEndpointHealthCheck, Status: platformstatus.Skipped,
			Message: "etcd endpoint checks skipped because no member service is available",
		})
	} else {
		component.Checks = append(component.Checks, checkEtcdCluster(ctx, state.deployConfig, state.apiConfig))
		attachEtcdFailureDetails(state, &component, healthyHost)
	}
	component.Message = etcdSummary(servers, component.Checks, statusComponent)
	return component
}

func checkEtcdCluster(ctx context.Context, deployConfig *options.DeployConfig, apiConfig *config.Config) Check {
	tlsConfig, err := etcdTLSConfig(apiConfig)
	if err != nil {
		return Check{
			Name: etcdEndpointHealthCheck, Target: "cluster", Status: platformstatus.Unknown,
			Message: "etcd client credentials are unavailable", Evidence: []string{sanitize(err.Error())},
		}
	}
	return checkEtcdClusterWithClient(ctx, etcdInitialEndpoints(deployConfig), tlsConfig, newEtcdClient)
}

// checkEtcdClusterWithClient follows `etcdctl endpoint health --cluster` and
// `etcdctl endpoint status --cluster`: discover member client URLs first, then
// probe each endpoint and identify the current leader.
func checkEtcdClusterWithClient(ctx context.Context, initialEndpoints []string, tlsConfig *tls.Config, newClient etcdClientFactory) Check {
	check := Check{Name: etcdEndpointHealthCheck, Target: "cluster"}
	if len(initialEndpoints) == 0 {
		check.Status = platformstatus.Unknown
		check.Message = "etcd endpoints are unavailable"
		return check
	}
	if newClient == nil {
		check.Status = platformstatus.Unknown
		check.Message = "etcd client is unavailable"
		return check
	}

	clusterClient, err := newClient(initialEndpoints, tlsConfig)
	if err != nil {
		check.Status = platformstatus.Unknown
		check.Message = "cannot create etcd client"
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	defer clusterClient.Close()

	endpoints, err := discoverEtcdMemberEndpoints(ctx, clusterClient)
	if err != nil {
		check.Status = platformstatus.Unhealthy
		check.Message = "etcd cluster member discovery failed"
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	if len(endpoints) == 0 {
		check.Status = platformstatus.Unhealthy
		check.Message = "etcd cluster has no advertised client endpoints"
		return check
	}

	if evidence := unhealthyEtcdEndpointEvidence(ctx, endpoints, tlsConfig, newClient); len(evidence) != 0 {
		check.Status = platformstatus.Unhealthy
		check.Message = "etcd cluster health check failed"
		check.Evidence = append(check.Evidence, evidence...)
		return check
	}
	return checkEtcdMemberStatus(ctx, clusterClient, endpoints, &check)
}

func discoverEtcdMemberEndpoints(ctx context.Context, client etcdClient) ([]string, error) {
	membersCtx, cancel := context.WithTimeout(ctx, etcdRequestTimeout)
	defer cancel()
	members, err := client.MemberList(membersCtx)
	if err != nil {
		return nil, err
	}
	return etcdMemberEndpoints(members), nil
}

func unhealthyEtcdEndpointEvidence(ctx context.Context, endpoints []string, tlsConfig *tls.Config, newClient etcdClientFactory) []string {
	healthChecks := runParallel(endpoints, func(endpoint string) []Check {
		return []Check{checkEtcdEndpointHealth(ctx, endpoint, tlsConfig, newClient)}
	})
	var evidence []string
	for i := range healthChecks {
		if healthChecks[i].Status == platformstatus.Healthy {
			continue
		}
		evidence = append(evidence, healthChecks[i].Target+": "+healthChecks[i].Message)
		evidence = append(evidence, healthChecks[i].Evidence...)
	}
	return evidence
}

func checkEtcdMemberStatus(ctx context.Context, client etcdClient, endpoints []string, check *Check) Check {
	var leader string
	statusAvailable := true
	for _, endpoint := range endpoints {
		statusCtx, cancel := context.WithTimeout(ctx, etcdRequestTimeout)
		response, statusErr := client.Status(statusCtx, endpoint)
		cancel()
		if statusErr != nil {
			statusAvailable = false
			check.Evidence = append(check.Evidence, endpoint+": "+sanitize(statusErr.Error()))
			continue
		}
		if response != nil && response.Header != nil && response.Header.MemberId == response.Leader && response.Leader != 0 {
			leader = endpoint
		}
	}
	if !statusAvailable {
		check.Status = platformstatus.Degraded
		check.Message = "etcd endpoints are healthy but member status is unavailable"
		return *check
	}

	check.Status = platformstatus.Healthy
	check.Message = "etcd cluster endpoints are healthy"
	if leader != "" {
		check.Evidence = append(check.Evidence, "leader="+leader)
	}
	return *check
}

func checkEtcdEndpointHealth(ctx context.Context, endpoint string, tlsConfig *tls.Config, newClient etcdClientFactory) Check {
	check := Check{Name: etcdEndpointHealthCheck, Target: endpoint}
	client, err := newClient([]string{endpoint}, tlsConfig)
	if err != nil {
		check.Status = platformstatus.Unhealthy
		check.Message = "cannot create etcd endpoint client"
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	defer client.Close()

	healthCtx, cancel := context.WithTimeout(ctx, etcdRequestTimeout)
	_, err = client.Get(healthCtx, "health")
	cancel()
	if err != nil && !errors.Is(err, rpctypes.ErrPermissionDenied) {
		check.Status = platformstatus.Unhealthy
		check.Message = "etcd endpoint health check failed"
		check.Evidence = []string{sanitize(err.Error())}
		return check
	}
	check.Status = platformstatus.Healthy
	check.Message = "etcd endpoint is healthy"
	return check
}

func newEtcdClient(endpoints []string, tlsConfig *tls.Config) (etcdClient, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: etcdRequestTimeout,
		TLS:         tlsConfig,
	})
}

func etcdInitialEndpoints(deployConfig *options.DeployConfig) []string {
	if deployConfig == nil || deployConfig.EtcdConfig == nil {
		return nil
	}
	endpoints := make([]string, 0, len(deployConfig.ServerIPs))
	for _, host := range sortedStrings(deployConfig.ServerIPs) {
		endpoints = append(endpoints, "https://"+endpoint(host, deployConfig.EtcdConfig.ClientPort))
	}
	return endpoints
}

func etcdMemberEndpoints(members *clientv3.MemberListResponse) []string {
	seen := make(map[string]struct{})
	var endpoints []string
	if members == nil {
		return endpoints
	}
	for _, member := range members.Members {
		if member == nil {
			continue
		}
		for _, endpoint := range member.ClientURLs {
			endpoint = strings.TrimSpace(endpoint)
			if endpoint == "" {
				continue
			}
			if _, found := seen[endpoint]; found {
				continue
			}
			seen[endpoint] = struct{}{}
			endpoints = append(endpoints, endpoint)
		}
	}
	return sortedStrings(endpoints)
}

func etcdTLSConfig(apiConfig *config.Config) (*tls.Config, error) {
	if apiConfig == nil {
		return nil, errors.New("API configuration is unavailable")
	}
	currentContext, found := apiConfig.Contexts[apiConfig.CurrentContext]
	if !found || currentContext == nil {
		return nil, fmt.Errorf("current context %q is not configured", apiConfig.CurrentContext)
	}
	server, found := apiConfig.Servers[currentContext.Server]
	if !found || server == nil {
		return nil, fmt.Errorf("server %q is not configured", currentContext.Server)
	}
	authInfo, found := apiConfig.AuthInfos[currentContext.AuthInfo]
	if !found || authInfo == nil {
		return nil, fmt.Errorf("user %q is not configured", currentContext.AuthInfo)
	}

	caData, err := configData(server.CertificateAuthorityData, server.CertificateAuthority, "certificate authority")
	if err != nil {
		return nil, err
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caData) {
		return nil, errors.New("parse certificate authority")
	}

	clientCert, err := clientCertificate(authInfo)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func configData(data []byte, path, name string) ([]byte, error) {
	if len(data) != 0 {
		return data, nil
	}
	if path == "" {
		return nil, fmt.Errorf("%s is unavailable", name)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return contents, nil
}

func clientCertificate(authInfo *config.AuthInfo) (tls.Certificate, error) {
	switch {
	case len(authInfo.ClientCertificateData) != 0 && len(authInfo.ClientKeyData) != 0:
		return tls.X509KeyPair(authInfo.ClientCertificateData, authInfo.ClientKeyData)
	case authInfo.ClientCertificate != "" && authInfo.ClientKey != "":
		return tls.LoadX509KeyPair(authInfo.ClientCertificate, authInfo.ClientKey)
	default:
		return tls.Certificate{}, errors.New("client certificate and key are unavailable")
	}
}

type agentTarget struct {
	ID       string
	IP       string
	Node     *corev1.Node
	Deployed bool
	Disabled bool
}

func checkKCAgents(_ context.Context, state *diagnosticState) Component {
	component := Component{Name: "kc-agent"}
	statusComponent := platformComponent(state.platform, "kc-agent")
	if statusComponent != nil {
		component.Checks = append(component.Checks, Check{
			Name: platformStatus, Target: platformTarget,
			Status: statusComponent.Status, Message: statusComponent.Message,
		})
	} else {
		component.Checks = append(component.Checks, Check{
			Name: platformStatus, Target: platformTarget,
			Status: platformstatus.Skipped, Message: "agent platform status check skipped",
		})
	}

	targets := mergeAgentTargets(state.deployConfig, state.nodes)
	if len(targets) == 0 {
		component.Checks = append(component.Checks, Check{
			Name: "agents", Status: platformstatus.Skipped, Message: "no agents discovered",
		})
		component.Message = "no agents discovered"
		return component
	}
	checks := runParallel(targets, func(target agentTarget) []Check {
		return checkAgent(state, target)
	})
	component.Checks = append(component.Checks, checks...)
	component.Message = agentSummary(targets, checks, statusComponent)
	return component
}

func checkAgent(state *diagnosticState, target agentTarget) []Check {
	label := targetLabel(target)
	if target.Disabled {
		return []Check{{Name: "disabled", Target: label, Status: platformstatus.Skipped, Message: fmt.Sprintf("agent %s is disabled", label)}}
	}
	var checks []Check
	checks = append(checks, agentHeartbeat(target, state.nodesLoaded))
	if !target.Deployed {
		status := platformstatus.Degraded
		if state.deployConfig == nil {
			status = platformstatus.Skipped
		}
		checks = append(checks, Check{
			Name: "remote-service", Target: label, Status: status,
			Message: fmt.Sprintf("SSH check skipped for %s because it is absent from deployment configuration", label),
		})
		return checks
	}
	if state.remote == nil {
		checks = append(checks, Check{Name: "remote-service", Target: label, Status: platformstatus.Skipped, Message: "agent SSH check skipped"})
		return checks
	}
	service := state.remote.service(target.IP, "kc-agent")
	service.Target = label
	checks = append(checks, service)
	if service.Status == platformstatus.Unhealthy || service.Status == platformstatus.Unknown {
		return checks
	}
	return checks
}

func agentHeartbeat(target agentTarget, inventoryLoaded bool) Check {
	label := targetLabel(target)
	if target.Node == nil {
		if !inventoryLoaded {
			return Check{
				Name: heartbeatCheck, Target: label, Status: platformstatus.Skipped,
				Message: fmt.Sprintf("agent %s heartbeat check skipped because API inventory is unavailable", label),
			}
		}
		return Check{
			Name: heartbeatCheck, Target: label, Status: platformstatus.Unhealthy,
			Message: fmt.Sprintf("agent %s is not registered", label),
		}
	}
	condition := nodeReadyCondition(target.Node)
	if condition == nil {
		return Check{
			Name: heartbeatCheck, Target: label, Status: platformstatus.Unknown,
			Message: fmt.Sprintf("agent %s has never reported readiness", label),
		}
	}
	check := Check{Name: heartbeatCheck, Target: label}
	check.Evidence = []string{
		"NodeReady=" + string(condition.Status),
		"LastHeartbeat=" + condition.LastHeartbeatTime.Local().Format(time.RFC3339),
	}
	if condition.Reason != "" {
		check.Evidence = append(check.Evidence, "Reason="+condition.Reason)
	}
	switch condition.Status {
	case corev1.ConditionTrue:
		check.Status = platformstatus.Healthy
		check.Message = fmt.Sprintf("agent %s heartbeat is current", label)
	case corev1.ConditionFalse:
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("agent %s reports not ready", label)
	default:
		check.Status = platformstatus.Unhealthy
		check.Message = fmt.Sprintf("agent %s heartbeat is stale", label)
	}
	return check
}

func nodeReadyCondition(node *corev1.Node) *corev1.NodeCondition {
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == corev1.NodeReady {
			return &node.Status.Conditions[i]
		}
	}
	return nil
}

func mergeAgentTargets(deployConfig *options.DeployConfig, nodes []corev1.Node) []agentTarget {
	byID := make(map[string]*corev1.Node, len(nodes))
	byIP := make(map[string]*corev1.Node, len(nodes))
	for i := range nodes {
		node := &nodes[i]
		byID[node.Name] = node
		if node.Status.Ipv4DefaultIP != "" {
			byIP[node.Status.Ipv4DefaultIP] = node
		}
	}
	var targets []agentTarget
	used := make(map[string]struct{})
	if deployConfig != nil {
		for ip, metadata := range deployConfig.Agents {
			node := byID[metadata.AgentID]
			if node == nil {
				node = byIP[ip]
			}
			target := agentTarget{ID: metadata.AgentID, IP: ip, Node: node, Deployed: true}
			if target.ID == "" && node != nil {
				target.ID = node.Name
			}
			if node != nil {
				_, target.Disabled = node.Labels[common.LabelNodeDisable]
				used[node.Name] = struct{}{}
			}
			targets = append(targets, target)
		}
	}
	for i := range nodes {
		node := &nodes[i]
		if _, found := used[node.Name]; found {
			continue
		}
		_, disabled := node.Labels[common.LabelNodeDisable]
		targets = append(targets, agentTarget{
			ID: node.Name, IP: node.Status.Ipv4DefaultIP, Node: node, Disabled: disabled,
		})
	}
	return targets
}

func targetLabel(target agentTarget) string {
	switch {
	case target.ID != "" && target.IP != "":
		return fmt.Sprintf("%s (%s)", target.ID, target.IP)
	case target.ID != "":
		return target.ID
	default:
		return target.IP
	}
}

func statusMessage(component *platformstatus.Component, fallback string) string {
	if component == nil || component.Message == "" {
		return fallback
	}
	return component.Message
}

func attachServerFailureDetails(state *diagnosticState, component *Component, servers []string) {
	for i := range component.Checks {
		check := &component.Checks[i]
		if check.Target != platformTarget || check.Status == platformstatus.Healthy || check.Status == platformstatus.Skipped {
			continue
		}
		for _, host := range servers {
			check.Logs = append(check.Logs, state.remote.journal(host, "kc-server")...)
			check.Commands = appendUnique(check.Commands, state.remote.serviceCommands(host, "kc-server")...)
		}
		switch check.Name {
		case "controller-manager":
			attachServerTimes(state.remote, check, servers)
		case "static-resource":
			attachStaticResourceEvidence(state, check, servers)
		}
		if len(check.Logs) > shownLogs {
			check.Logs = check.Logs[len(check.Logs)-shownLogs:]
		}
	}
}

func attachEtcdFailureDetails(state *diagnosticState, component *Component, host string) {
	for i := range component.Checks {
		check := &component.Checks[i]
		if check.Name != etcdEndpointHealthCheck || check.Status == platformstatus.Healthy || check.Status == platformstatus.Skipped {
			continue
		}
		if state.deployConfig.EtcdConfig != nil && state.deployConfig.EtcdConfig.DataDir != "" {
			diskCommand := fmt.Sprintf("timeout %d df -P %s", remoteCommandTimeout,
				shellQuote(state.deployConfig.EtcdConfig.DataDir))
			for _, evidence := range state.remote.capture(host, diskCommand) {
				check.Evidence = append(check.Evidence, host+" "+evidence)
			}
			check.Commands = appendUnique(check.Commands, state.remote.sshCommand(host, diskCommand))
		}
		check.Logs = append(check.Logs, state.remote.journal(host, "kc-etcd")...)
		check.Commands = appendUnique(check.Commands, state.remote.serviceCommands(host, "kc-etcd")...)
	}
}

func attachServerTimes(remote *remoteRunner, check *Check, servers []string) {
	for _, host := range servers {
		lines := remote.capture(host, fmt.Sprintf("timeout %d date +%%s", remoteCommandTimeout))
		if len(lines) == 1 {
			timestamp, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
			if err == nil {
				check.Evidence = append(check.Evidence,
					fmt.Sprintf("%s clock offset=%ds", host, timestamp-time.Now().Unix()))
				continue
			}
		}
		check.Evidence = append(check.Evidence, fmt.Sprintf("%s clock check failed", host))
	}
}

func attachStaticResourceEvidence(state *diagnosticState, check *Check, servers []string) {
	path := state.deployConfig.StaticServerPath
	if path == "" {
		return
	}
	quotedPath := shellQuote(path)
	script := fmt.Sprintf("if test -d %s; then echo directory=present; else echo directory=missing; fi; "+
		"if test -r %s; then echo readable=true; else echo readable=false; fi; df -P %s",
		quotedPath, quotedPath, quotedPath)
	command := fmt.Sprintf("timeout %d sh -c %s", remoteCommandTimeout, shellQuote(script))
	for _, host := range servers {
		for _, evidence := range state.remote.capture(host, command) {
			check.Evidence = append(check.Evidence, host+" "+evidence)
		}
		check.Commands = appendUnique(check.Commands, state.remote.sshCommand(host, "df -P "+quotedPath))
	}
}

func serverSummary(servers []string, checks []Check, status *platformstatus.Component) string {
	healthy := healthyServices(checks, "kc-server-service")
	if healthy == len(servers) && status != nil && status.Status == platformstatus.Healthy {
		return fmt.Sprintf("%d/%d instances healthy; all subsystems ready", healthy, len(servers))
	}
	return fmt.Sprintf("%d/%d instances running; %s", healthy, len(servers), statusMessage(status, "subsystem status unavailable"))
}

func etcdSummary(servers []string, checks []Check, status *platformstatus.Component) string {
	healthy := healthyServices(checks, "kc-etcd-service")
	clusterHealthy := hasHealthyCheck(checks, etcdEndpointHealthCheck)
	if healthy == len(servers) && clusterHealthy {
		return fmt.Sprintf("%d/%d members healthy", healthy, len(servers))
	}
	return fmt.Sprintf("%d/%d member services running; %s", healthy, len(servers), statusMessage(status, "cluster status unavailable"))
}

func hasHealthyCheck(checks []Check, name string) bool {
	for i := range checks {
		if checks[i].Name == name && checks[i].Status == platformstatus.Healthy {
			return true
		}
	}
	return false
}

func agentSummary(targets []agentTarget, checks []Check, status *platformstatus.Component) string {
	enabled := 0
	for _, target := range targets {
		if !target.Disabled {
			enabled++
		}
	}
	running, inspected := serviceCounts(checks, "kc-agent-service")
	if inspected == 0 {
		return statusMessage(status, "heartbeat status unavailable") + "; SSH checks skipped"
	}
	if status != nil && status.Status == platformstatus.Healthy && running == enabled && inspected == enabled {
		return fmt.Sprintf("%d/%d agents healthy", enabled, enabled)
	}
	hostState := fmt.Sprintf("%d/%d agent services running", running, inspected)
	if status == nil {
		return hostState + "; API heartbeat status unavailable"
	}
	if status.Status == platformstatus.Healthy && running < inspected {
		return hostState + "; API still reports " + status.Message
	}
	return hostState + "; API reports " + status.Message
}

func healthyServices(checks []Check, name string) int {
	healthy, _ := serviceCounts(checks, name)
	return healthy
}

func serviceCounts(checks []Check, name string) (healthy, total int) {
	for i := range checks {
		if checks[i].Name != name {
			continue
		}
		total++
		if checks[i].Status == platformstatus.Healthy {
			healthy++
		}
	}
	return
}

func endpoint(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if value == "" {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
