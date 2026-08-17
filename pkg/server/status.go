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

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/component-base/version"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
	nodeLeaseNamespace       = "node-lease"
	defaultNodeLeaseDuration = 240 * time.Second
	statusTimeout            = 8 * time.Second
	maxEtcdHealthBodyBytes   = 1024
	httpScheme               = "http"
	httpsScheme              = "https"
)

func (s *APIServer) PlatformStatus(ctx context.Context) *platformstatus.PlatformStatus {
	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()

	components := make([]platformstatus.Component, 3)
	var wg sync.WaitGroup
	wg.Add(len(components))
	go func() {
		defer wg.Done()
		components[0] = s.kcServerStatus(ctx)
	}()
	go func() {
		defer wg.Done()
		components[1] = s.kcEtcdStatus(ctx)
	}()
	go func() {
		defer wg.Done()
		components[2] = s.kcAgentStatus(ctx)
	}()
	wg.Wait()

	return platformstatus.New(components, startedAt, time.Since(startedAt))
}

func (s *APIServer) kcServerStatus(ctx context.Context) platformstatus.Component {
	startedAt := time.Now()
	checks := []platformstatus.Check{
		{
			Name:    "api",
			Status:  platformstatus.Healthy,
			Message: fmt.Sprintf("API ready, version %s", version.Get().GitVersion),
		},
		timedCheck("controller-manager", func() (platformstatus.Status, string) {
			if s.controllerManager == nil {
				return platformstatus.Unknown, "controller-manager status is unavailable"
			}
			if err := s.controllerManager.Health(ctx); err != nil {
				return platformstatus.Unhealthy, "active leader is not ready"
			}
			return platformstatus.Healthy, "active leader is ready"
		}),
		timedCheck("static-resource", func() (platformstatus.Status, string) {
			if s.staticResourceService == nil {
				return platformstatus.Unknown, "resource service status is unavailable"
			}
			if err := s.staticResourceService.Health(ctx); err != nil {
				return platformstatus.Unhealthy, "resource service is unavailable"
			}
			return platformstatus.Healthy, "resource service available"
		}),
	}

	component := platformstatus.Component{
		Name:           "kc-server",
		DurationMillis: time.Since(startedAt).Milliseconds(),
		Checks:         checks,
	}
	component.Status, component.Message = aggregateServerChecks(checks)
	return component
}

func aggregateServerChecks(checks []platformstatus.Check) (status platformstatus.Status, message string) {
	failed := make([]platformstatus.Check, 0, len(checks))
	status = platformstatus.Healthy
	for _, check := range checks {
		if check.Status == platformstatus.Healthy || check.Status == platformstatus.Skipped {
			continue
		}
		failed = append(failed, check)
		if check.Name == "static-resource" {
			if status == platformstatus.Healthy {
				status = platformstatus.Degraded
			}
		} else {
			status = platformstatus.Unhealthy
		}
	}
	switch len(failed) {
	case 0:
		return platformstatus.Healthy, "all server subsystems ready"
	case 1:
		return status, failed[0].Message
	default:
		return status, fmt.Sprintf("%d/%d subsystems unhealthy", len(failed), len(checks))
	}
}

func (s *APIServer) kcEtcdStatus(ctx context.Context) platformstatus.Component {
	startedAt := time.Now()
	component := platformstatus.Component{Name: "kc-etcd"}
	endpoints := s.Config.EtcdOptions.ServerList
	if len(endpoints) == 0 {
		component.Status = platformstatus.Unknown
		component.Message = "no endpoints configured"
		return component
	}

	client, err := s.etcdHealthClient()
	if err != nil {
		component.Status = platformstatus.Unhealthy
		component.Message = "health client configuration is invalid"
		return component
	}

	var wg sync.WaitGroup
	results := make(chan bool, len(endpoints))
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			healthURL, requestErr := etcdHealthURL(endpoint, s.etcdUsesTLS())
			if requestErr != nil {
				results <- false
				return
			}
			request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, http.NoBody)
			if requestErr != nil {
				results <- false
				return
			}
			results <- etcdEndpointHealthy(client, request)
		}()
	}
	wg.Wait()
	close(results)

	healthy := 0
	for result := range results {
		if result {
			healthy++
		}
	}
	component.DurationMillis = time.Since(startedAt).Milliseconds()
	component.Status, component.Message = aggregateEtcdEndpoints(healthy, len(endpoints))
	return component
}

func etcdEndpointHealthy(client *http.Client, request *http.Request) bool {
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false
	}
	var result struct {
		Health json.RawMessage `json:"health"`
	}
	if decodeErr := json.NewDecoder(io.LimitReader(response.Body, maxEtcdHealthBodyBytes)).Decode(&result); decodeErr != nil {
		return false
	}
	healthy, err := strconv.ParseBool(strings.Trim(string(result.Health), `"`))
	return err == nil && healthy
}

func (s *APIServer) etcdUsesTLS() bool {
	options := s.Config.EtcdOptions
	return options.TrustedCAFile != "" || options.CertFile != "" || options.KeyFile != ""
}

func etcdHealthURL(endpoint string, useTLS bool) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("etcd endpoint is empty")
	}
	if !strings.Contains(endpoint, "://") {
		scheme := httpScheme
		if useTLS {
			scheme = httpsScheme
		}
		endpoint = scheme + "://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse etcd endpoint: %w", err)
	}
	if (parsed.Scheme != httpScheme && parsed.Scheme != httpsScheme) || parsed.Host == "" {
		return "", fmt.Errorf("invalid etcd endpoint %q", endpoint)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/health"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s *APIServer) etcdHealthClient() (*http.Client, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if s.Config.EtcdOptions.TrustedCAFile != "" {
		caData, err := os.ReadFile(s.Config.EtcdOptions.TrustedCAFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("parse etcd CA certificate")
		}
		tlsConfig.RootCAs = pool
	}
	if s.Config.EtcdOptions.CertFile != "" || s.Config.EtcdOptions.KeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(s.Config.EtcdOptions.CertFile, s.Config.EtcdOptions.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}, nil
}

func (s *APIServer) kcAgentStatus(ctx context.Context) platformstatus.Component {
	startedAt := time.Now()
	component := platformstatus.Component{Name: "kc-agent"}
	if s.clusterOperator == nil || s.leaseOperator == nil {
		component.Status = platformstatus.Unknown
		component.Message = "agent status is unavailable"
		return component
	}

	nodes, err := s.clusterOperator.ListNodes(ctx, query.New())
	if err != nil {
		component.Status = platformstatus.Unknown
		component.Message = "unable to list registered agents"
		return component
	}
	leases, err := s.leaseOperator.ListLeases(ctx, query.New())
	if err != nil {
		component.Status = platformstatus.Unknown
		component.Message = "unable to read agent heartbeats"
		return component
	}

	total, running := countRunningAgents(nodes, leases, time.Now())

	component.DurationMillis = time.Since(startedAt).Milliseconds()
	component.Status, component.Message = aggregateCount(running, total, "agents running", true)
	return component
}

func countRunningAgents(nodes *corev1.NodeList, leases *coordinationv1.LeaseList, now time.Time) (total, running int) {
	leaseByNode := make(map[string]time.Time, len(leases.Items))
	for i := range leases.Items {
		item := &leases.Items[i]
		if item.Namespace != nodeLeaseNamespace || item.Spec.RenewTime == nil {
			continue
		}
		duration := defaultNodeLeaseDuration
		if item.Spec.LeaseDurationSeconds != nil {
			duration = time.Duration(*item.Spec.LeaseDurationSeconds) * time.Second
		}
		leaseByNode[item.Name] = item.Spec.RenewTime.Add(duration)
	}

	for i := range nodes.Items {
		node := &nodes.Items[i]
		if _, disabled := node.Labels[common.LabelNodeDisable]; disabled {
			continue
		}
		total++
		if expiresAt, found := leaseByNode[node.Name]; found && now.Before(expiresAt) {
			running++
		}
	}
	return total, running
}

func aggregateCount(
	healthy, total int,
	messageSuffix string,
	skipWhenEmpty bool,
) (status platformstatus.Status, message string) {
	if total == 0 {
		if skipWhenEmpty {
			return platformstatus.Skipped, "no agents registered"
		}
		return platformstatus.Unknown, "no endpoints configured"
	}
	message = fmt.Sprintf("%d/%d %s", healthy, total, messageSuffix)
	switch healthy {
	case total:
		return platformstatus.Healthy, message
	case 0:
		return platformstatus.Unhealthy, message
	default:
		return platformstatus.Degraded, message
	}
}

func aggregateEtcdEndpoints(healthy, total int) (status platformstatus.Status, message string) {
	if total == 0 {
		return platformstatus.Unknown, "no endpoints configured"
	}
	switch healthy {
	case total:
		return platformstatus.Healthy, "etcd is healthy"
	case 0:
		return platformstatus.Unhealthy, "etcd is unavailable"
	default:
		return platformstatus.Degraded, "some etcd endpoints are unavailable"
	}
}

func timedCheck(name string, check func() (platformstatus.Status, string)) platformstatus.Check {
	startedAt := time.Now()
	status, message := check()
	return platformstatus.Check{
		Name:           name,
		Status:         status,
		Message:        message,
		DurationMillis: time.Since(startedAt).Milliseconds(),
	}
}
