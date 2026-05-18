/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func TestGetCurrentRegistry(t *testing.T) {
	tests := []struct {
		name string
		cfg  *RegistryConfig
		want *RegistryEntry
	}{
		{
			name: "empty config",
			cfg:  &RegistryConfig{},
			want: nil,
		},
		{
			name: "current not found in registries",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.2", Port: 5000},
				},
			},
			want: nil,
		},
		{
			name: "current found",
			cfg: &RegistryConfig{
				Current: "10.0.0.2",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
					{Node: "10.0.0.2", Port: 6666},
				},
			},
			want: &RegistryEntry{Node: "10.0.0.2", Port: 6666},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCurrentRegistry(tt.cfg)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got.Node != tt.want.Node || got.Port != tt.want.Port {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestFindRegistry(t *testing.T) {
	cfg := &RegistryConfig{
		Registries: []RegistryEntry{
			{Node: "10.0.0.1", Port: 5000},
			{Node: "10.0.0.2", Port: 6666},
		},
	}
	if e := findRegistry(cfg, "10.0.0.2"); e == nil || e.Port != 6666 {
		t.Fatalf("expected port 6666, got %+v", e)
	}
	if e := findRegistry(cfg, "10.0.0.99"); e != nil {
		t.Fatalf("expected nil, got %+v", e)
	}
}

func TestAddOrUpdateRegistry(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *RegistryConfig
		entry    RegistryEntry
		wantLen  int
		wantNode string
	}{
		{
			name:     "add to empty",
			cfg:      &RegistryConfig{},
			entry:    RegistryEntry{Node: "10.0.0.1", Port: 5000},
			wantLen:  1,
			wantNode: "10.0.0.1",
		},
		{
			name: "add new to existing",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
				},
			},
			entry:    RegistryEntry{Node: "10.0.0.2", Port: 6666},
			wantLen:  2,
			wantNode: "10.0.0.2",
		},
		{
			name: "update existing",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
				},
			},
			entry:    RegistryEntry{Node: "10.0.0.1", Port: 6666},
			wantLen:  1,
			wantNode: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddOrUpdateRegistry(tt.cfg, tt.entry)
			if len(tt.cfg.Registries) != tt.wantLen {
				t.Fatalf("expected %d registries, got %d", tt.wantLen, len(tt.cfg.Registries))
			}
			if tt.cfg.Current != tt.wantNode {
				t.Fatalf("expected current=%s, got current=%s", tt.wantNode, tt.cfg.Current)
			}
			found := false
			for _, r := range tt.cfg.Registries {
				if r.Node == tt.entry.Node && r.Port == tt.entry.Port {
					found = true
				}
			}
			if !found {
				t.Fatalf("entry %v not found in registries", tt.entry)
			}
		})
	}
}

func TestRemoveRegistry(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *RegistryConfig
		removeNode string
		wantLen    int
		wantCurr   string
	}{
		{
			name:       "remove from empty",
			cfg:        &RegistryConfig{},
			removeNode: "10.0.0.1",
			wantLen:    0,
			wantCurr:   "",
		},
		{
			name: "remove current with others remaining",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
					{Node: "10.0.0.2", Port: 5000},
				},
			},
			removeNode: "10.0.0.1",
			wantLen:    1,
			wantCurr:   "10.0.0.2",
		},
		{
			name: "remove only registry",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
				},
			},
			removeNode: "10.0.0.1",
			wantLen:    0,
			wantCurr:   "",
		},
		{
			name: "remove non-current",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
					{Node: "10.0.0.2", Port: 5000},
				},
			},
			removeNode: "10.0.0.2",
			wantLen:    1,
			wantCurr:   "10.0.0.1",
		},
		{
			name: "remove nonexistent node",
			cfg: &RegistryConfig{
				Current: "10.0.0.1",
				Registries: []RegistryEntry{
					{Node: "10.0.0.1", Port: 5000},
				},
			},
			removeNode: "10.0.0.99",
			wantLen:    1,
			wantCurr:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RemoveRegistry(tt.cfg, tt.removeNode)
			if len(tt.cfg.Registries) != tt.wantLen {
				t.Fatalf("expected %d registries, got %d", tt.wantLen, len(tt.cfg.Registries))
			}
			if tt.cfg.Current != tt.wantCurr {
				t.Fatalf("expected current=%s, got current=%s", tt.wantCurr, tt.cfg.Current)
			}
		})
	}
}

func TestNewEntryFromOptions(t *testing.T) {
	o := &RegistryOptions{
		Node:         "10.0.0.1",
		RegistryPort: 6666,
		SSHConfig: &sshutils.SSH{
			User:       "root",
			PkFile:     "/path/to/key",
			PkPassword: "secret",
		},
	}

	entry := newEntryFromOptions(o)
	if entry.Node != "10.0.0.1" {
		t.Fatalf("expected node=10.0.0.1, got %s", entry.Node)
	}
	if entry.Port != 6666 {
		t.Fatalf("expected port=6666, got %d", entry.Port)
	}
	if entry.SSH.User != "root" {
		t.Fatalf("expected ssh.user=root, got %s", entry.SSH.User)
	}
	if entry.SSH.PkFile != "/path/to/key" {
		t.Fatalf("expected ssh.pkFile=/path/to/key, got %s", entry.SSH.PkFile)
	}
	if entry.SSH.PkPasswd != "secret" {
		t.Fatalf("expected ssh.pkPasswd=secret, got %s", entry.SSH.PkPasswd)
	}
}

func TestApplyEntryToOptions(t *testing.T) {
	tests := []struct {
		name            string
		o               *RegistryOptions
		entry           *RegistryEntry
		nodeChanged     bool
		portChanged     bool
		sshUserChanged  bool
		pkFileChanged   bool
		pkPasswdChanged bool
		wantNode        string
		wantPort        int
		wantSSHUser     string
		wantPkFile      string
		wantPkPasswd    string
	}{
		{
			name: "fill all from entry",
			o: &RegistryOptions{
				Node:         "",
				RegistryPort: 0,
				SSHConfig:    sshutils.NewSSH(),
			},
			entry: &RegistryEntry{
				Node: "10.0.0.1",
				Port: 6666,
				SSH: RegistrySSH{
					User:     "root",
					PkFile:   "/key",
					PkPasswd: "pw",
				},
			},
			nodeChanged:     false,
			portChanged:     false,
			sshUserChanged:  false,
			pkFileChanged:   false,
			pkPasswdChanged: false,
			wantNode:        "10.0.0.1",
			wantPort:        6666,
			wantSSHUser:     "root",
			wantPkFile:      "/key",
			wantPkPasswd:    "pw",
		},
		{
			name: "cli flags override config",
			o: &RegistryOptions{
				Node:         "10.0.0.2",
				RegistryPort: 5000,
				SSHConfig: &sshutils.SSH{
					User:       "ubuntu",
					PkFile:     "/other",
					PkPassword: "otherpw",
				},
			},
			entry: &RegistryEntry{
				Node: "10.0.0.1",
				Port: 6666,
				SSH: RegistrySSH{
					User:     "root",
					PkFile:   "/key",
					PkPasswd: "pw",
				},
			},
			nodeChanged:     true,
			portChanged:     true,
			sshUserChanged:  true,
			pkFileChanged:   true,
			pkPasswdChanged: true,
			wantNode:        "10.0.0.2",
			wantPort:        5000,
			wantSSHUser:     "ubuntu",
			wantPkFile:      "/other",
			wantPkPasswd:    "otherpw",
		},
		{
			name: "partial override - node from config, port from cli",
			o: &RegistryOptions{
				Node:         "",
				RegistryPort: 5000,
				SSHConfig:    sshutils.NewSSH(),
			},
			entry: &RegistryEntry{
				Node: "10.0.0.1",
				Port: 6666,
			},
			nodeChanged:  false,
			portChanged:  true,
			wantNode:     "10.0.0.1",
			wantPort:     5000,
			wantSSHUser:  sshutils.NewSSH().User,
			wantPkFile:   "",
			wantPkPasswd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyEntryToOptions(tt.o, tt.entry, tt.nodeChanged, tt.portChanged, tt.sshUserChanged, tt.pkFileChanged, tt.pkPasswdChanged)
			if tt.o.Node != tt.wantNode {
				t.Errorf("node: expected %s, got %s", tt.wantNode, tt.o.Node)
			}
			if tt.o.RegistryPort != tt.wantPort {
				t.Errorf("port: expected %d, got %d", tt.wantPort, tt.o.RegistryPort)
			}
			if tt.o.SSHConfig != nil {
				if tt.o.SSHConfig.User != tt.wantSSHUser {
					t.Errorf("ssh user: expected %s, got %s", tt.wantSSHUser, tt.o.SSHConfig.User)
				}
				if tt.o.SSHConfig.PkFile != tt.wantPkFile {
					t.Errorf("pkFile: expected %s, got %s", tt.wantPkFile, tt.o.SSHConfig.PkFile)
				}
				if tt.o.SSHConfig.PkPassword != tt.wantPkPasswd {
					t.Errorf("pkPasswd: expected %s, got %s", tt.wantPkPasswd, tt.o.SSHConfig.PkPassword)
				}
			}
		})
	}
}

func TestSaveAndLoadRegistryConfig(t *testing.T) {
	dir := t.TempDir()
	origPath := registryConfigPath
	registryConfigPath = func() string { return filepath.Join(dir, "registry-config.yaml") }
	defer func() { registryConfigPath = origPath }()

	cfg := &RegistryConfig{
		Current: "10.0.0.1",
		Registries: []RegistryEntry{
			{
				Node: "10.0.0.1",
				Port: 5000,
				SSH: RegistrySSH{
					User:   "root",
					PkFile: "/home/user/.ssh/id_rsa",
				},
			},
			{
				Node: "10.0.0.2",
				Port: 6666,
			},
		},
	}

	if err := SaveRegistryConfig(cfg); err != nil {
		t.Fatalf("SaveRegistryConfig failed: %v", err)
	}

	loaded, err := LoadRegistryConfig()
	if err != nil {
		t.Fatalf("LoadRegistryConfig failed: %v", err)
	}

	if loaded.Current != cfg.Current {
		t.Fatalf("current: expected %s, got %s", cfg.Current, loaded.Current)
	}
	if len(loaded.Registries) != len(cfg.Registries) {
		t.Fatalf("registries count: expected %d, got %d", len(cfg.Registries), len(loaded.Registries))
	}
	for i, want := range cfg.Registries {
		got := loaded.Registries[i]
		if got.Node != want.Node || got.Port != want.Port {
			t.Fatalf("registry[%d]: expected %+v, got %+v", i, want, got)
		}
		if got.SSH.User != want.SSH.User || got.SSH.PkFile != want.SSH.PkFile {
			t.Fatalf("registry[%d].SSH: expected %+v, got %+v", i, want.SSH, got.SSH)
		}
	}
}

func TestLoadRegistryConfigNotExist(t *testing.T) {
	dir := t.TempDir()
	origPath := registryConfigPath
	registryConfigPath = func() string { return filepath.Join(dir, "nonexistent.yaml") }
	defer func() { registryConfigPath = origPath }()

	cfg, err := LoadRegistryConfig()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Current != "" {
		t.Fatalf("expected empty current, got %s", cfg.Current)
	}
	if len(cfg.Registries) != 0 {
		t.Fatalf("expected empty registries, got %d", len(cfg.Registries))
	}
}

func TestLoadRegistryConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("{{invalid yaml"), 0644)

	origPath := registryConfigPath
	registryConfigPath = func() string { return path }
	defer func() { registryConfigPath = origPath }()

	_, err := LoadRegistryConfig()
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestRoundTripEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	origPath := registryConfigPath
	registryConfigPath = func() string { return filepath.Join(dir, "registry-config.yaml") }
	defer func() { registryConfigPath = origPath }()

	cfg := &RegistryConfig{}
	if err := SaveRegistryConfig(cfg); err != nil {
		t.Fatalf("SaveRegistryConfig failed: %v", err)
	}

	loaded, err := LoadRegistryConfig()
	if err != nil {
		t.Fatalf("LoadRegistryConfig failed: %v", err)
	}
	if loaded.Current != "" {
		t.Fatalf("expected empty current, got %s", loaded.Current)
	}
	if len(loaded.Registries) != 0 {
		t.Fatalf("expected empty registries, got %d", len(loaded.Registries))
	}
}
