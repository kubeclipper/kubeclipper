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
	"crypto/x509"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func TestPrecheckServiceDoesNotIgnoreFailureWithAssumeYes(t *testing.T) {
	originalAssumeYes := options.AssumeYes
	options.AssumeYes = true
	t.Cleanup(func() { options.AssumeYes = originalAssumeYes })

	d := NewDeployOptions(options.IOStreams{})
	if d.precheckService("test", []string{"192.0.2.1"}, func(*sshutils.SSH, string) error {
		return errors.New("precheck failed")
	}) {
		t.Fatal("precheckService() succeeded with --assumeyes after a precheck failure")
	}
}

func TestTimeSyncPrecheckSupportsSystemdTimesyncd(t *testing.T) {
	if !strings.Contains(timeSyncPrecheckCommand, "systemd-timesyncd") {
		t.Fatalf("time synchronization precheck must support systemd-timesyncd: %q", timeSyncPrecheckCommand)
	}
}

func TestKcServerCertificateUsesAuthenticatedServerIdentity(t *testing.T) {
	certs := kcServerCertList([]string{options.KCServerAltName}, map[string][]x509.ExtKeyUsage{
		options.KCServer: {x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	})
	if len(certs) != 1 {
		t.Fatalf("expected one kc-server certificate, got %d", len(certs))
	}
	if certs[0].BaseName != options.KCServer {
		t.Fatalf("certificate file basename = %q, want %q", certs[0].BaseName, options.KCServer)
	}
	if certs[0].CommonName != kcServerClientIdentity {
		t.Fatalf("certificate common name = %q, want %q", certs[0].CommonName, kcServerClientIdentity)
	}
}

func TestDeployOptions_getEtcdTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	for _, s := range d.deployConfig.ServerIPs {
		t.Log(d.getEtcdTemplateContent(s))
	}
}

func TestDeployOptions_getKcServerConfigTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	for _, s := range d.deployConfig.ServerIPs {
		t.Log(d.deployConfig.GetKcServerConfigTemplateContent(s))
	}
}

func TestDeployOptions_getKcAgentConfigTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}
	metadata := options.Metadata{
		Region:  d.deployConfig.DefaultRegion,
		FloatIP: "1.1.1.1",
	}
	for range d.deployConfig.ServerIPs {
		metadata.AgentID = uuid.New().String()
		t.Log(d.deployConfig.GetKcAgentConfigTemplateContent(metadata))
	}
}

func TestDeployOptions_getKcConsoleTemplateContent(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.168.234.3", "192.168.234.4", "192.168.234.5"}
	d.servers = map[string]string{
		"192.168.234.3": "master1",
		"192.168.234.4": "master2",
		"192.168.234.5": "master3",
	}

	t.Log(d.getKcConsoleTemplateContent())
}

func TestDeployOptions_nodeRole(t *testing.T) {
	tests := []struct {
		name      string
		serverIPs []string
		agentIPs  []string
		queryIP   string
		wantRole  string
	}{
		{
			name:      "server only",
			serverIPs: []string{"10.0.0.1", "10.0.0.2"},
			agentIPs:  []string{"10.0.0.3"},
			queryIP:   "10.0.0.1",
			wantRole:  "server",
		},
		{
			name:      "agent only",
			serverIPs: []string{"10.0.0.1"},
			agentIPs:  []string{"10.0.0.2", "10.0.0.3"},
			queryIP:   "10.0.0.2",
			wantRole:  "agent",
		},
		{
			name:      "AIO node is server+agent",
			serverIPs: []string{"10.0.0.1"},
			agentIPs:  []string{"10.0.0.1", "10.0.0.2"},
			queryIP:   "10.0.0.1",
			wantRole:  "server+agent",
		},
		{
			name:      "unknown IP returns empty",
			serverIPs: []string{"10.0.0.1"},
			agentIPs:  []string{"10.0.0.2"},
			queryIP:   "10.0.0.99",
			wantRole:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeployOptions(options.IOStreams{})
			d.deployConfig.ServerIPs = tt.serverIPs
			d.deployConfig.Agents = make(options.Agents)
			for _, ip := range tt.agentIPs {
				d.deployConfig.Agents[ip] = options.Metadata{}
			}
			got := d.nodeRole(tt.queryIP)
			if got != tt.wantRole {
				t.Errorf("nodeRole(%q) = %q, want %q", tt.queryIP, got, tt.wantRole)
			}
		})
	}
}
