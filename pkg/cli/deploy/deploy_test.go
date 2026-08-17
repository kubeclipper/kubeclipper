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
	"context"
	"crypto/x509"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

type fakeEtcdHealthClient struct {
	getErr error
	keys   []string
}

func (c *fakeEtcdHealthClient) Get(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	c.keys = append(c.keys, key)
	return &clientv3.GetResponse{}, c.getErr
}

func (*fakeEtcdHealthClient) Close() error {
	return nil
}

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

func TestCheckEtcdEndpoints(t *testing.T) {
	endpoints := []string{"192.0.2.10:12379", "192.0.2.11:12379"}
	t.Run("all endpoints healthy", func(t *testing.T) {
		clients := []*fakeEtcdHealthClient{{}, {}}
		if err := checkEtcdEndpoints(context.Background(), []etcdHealthClient{clients[0], clients[1]}, endpoints); err != nil {
			t.Fatalf("checkEtcdEndpoints() error = %v", err)
		}
		for _, client := range clients {
			if !reflect.DeepEqual(client.keys, []string{"health"}) {
				t.Fatalf("health check keys = %v, want [health]", client.keys)
			}
		}
	})

	t.Run("endpoint unhealthy", func(t *testing.T) {
		clients := []etcdHealthClient{&fakeEtcdHealthClient{}, &fakeEtcdHealthClient{getErr: errors.New("connection refused")}}
		err := checkEtcdEndpoints(context.Background(), clients, endpoints)
		if err == nil || !strings.Contains(err.Error(), endpoints[1]) {
			t.Fatalf("checkEtcdEndpoints() error = %v, want endpoint %q", err, endpoints[1])
		}
	})
}

func TestDeployOptionsEtcdEndpoints(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.0.2.10", "192.0.2.11"}
	d.deployConfig.EtcdConfig.ClientPort = 22379

	if got, want := d.etcdEndpoints(), []string{"192.0.2.10:22379", "192.0.2.11:22379"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("etcdEndpoints() = %v, want %v", got, want)
	}
}

func TestDeployOptionsCompleteGeneratesAuthenticationJWTSecret(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.0.2.10"}
	d.deployConfig.AuthenticationOpts.JwtSecret = ""

	if err := d.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if d.deployConfig.AuthenticationOpts.JwtSecret == "" {
		t.Fatal("Complete() did not generate authentication JWT secret")
	}
}

func TestDeployOptionsValidateTempDir(t *testing.T) {
	for _, tt := range []struct {
		name    string
		tempDir string
		wantErr string
	}{
		{name: "absolute path", tempDir: "/var/lib/kubeclipper/tmp"},
		{name: "relative path", tempDir: "kubeclipper/tmp", wantErr: "absolute"},
		{name: "filesystem root", tempDir: "/", wantErr: "must not be the filesystem root"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDeployOptions(options.IOStreams{})
			d.aio = true
			d.deployConfig.Pkg = "/tmp/kc-amd64.tar.gz"
			d.deployConfig.ServerIPs = []string{"192.0.2.10"}
			d.deployConfig.TempDir = tt.tempDir

			err := d.ValidateArgs()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("ValidateArgs() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("ValidateArgs() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestKcServerConfigUsesAuthenticationJWTSecret(t *testing.T) {
	d := NewDeployOptions(options.IOStreams{})
	d.deployConfig.ServerIPs = []string{"192.0.2.10"}
	d.deployConfig.AuthenticationOpts.JwtSecret = "authentication-secret"

	content, err := d.deployConfig.GetKcServerConfigTemplateContent("192.0.2.10")
	if err != nil {
		t.Fatalf("GetKcServerConfigTemplateContent() error = %v", err)
	}
	if !strings.Contains(content, "jwtSecret: authentication-secret") {
		t.Fatalf("server config does not use authentication JWT secret:\n%s", content)
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
