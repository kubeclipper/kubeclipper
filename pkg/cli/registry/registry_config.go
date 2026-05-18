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
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

const DefaultRegistryConfigFile = "registry-config.yaml"

type RegistrySSH struct {
	User     string `json:"user" yaml:"user"`
	PkFile   string `json:"pkFile,omitempty" yaml:"pkFile,omitempty"`
	PkPasswd string `json:"pkPasswd,omitempty" yaml:"pkPasswd,omitempty"` // maps to sshutils.SSH.PkPassword
}

type RegistryEntry struct {
	Node string      `json:"node" yaml:"node"`
	Port int         `json:"port" yaml:"port"`
	SSH  RegistrySSH `json:"ssh,omitempty" yaml:"ssh,omitempty"`
}

type RegistryConfig struct {
	Current    string          `json:"current,omitempty" yaml:"current,omitempty"`
	Registries []RegistryEntry `json:"registries,omitempty" yaml:"registries,omitempty"`
}

var registryConfigPath = func() string {
	return filepath.Join(homedir.HomeDir(), config.DefaultConfigPath, DefaultRegistryConfigFile)
}

func LoadRegistryConfig() (*RegistryConfig, error) {
	path := registryConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RegistryConfig{}, nil
		}
		return nil, fmt.Errorf("read registry config: %w", err)
	}
	cfg := &RegistryConfig{}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse registry config: %w", err)
	}
	return cfg, nil
}

func SaveRegistryConfig(cfg *RegistryConfig) error {
	path := registryConfigPath()
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create registry config dir: %w", err)
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal registry config: %w", err)
	}
	if err = utils.WriteToFile(path, data); err != nil {
		return fmt.Errorf("write registry config: %w", err)
	}
	return nil
}

func GetCurrentRegistry(cfg *RegistryConfig) *RegistryEntry {
	if cfg.Current == "" {
		return nil
	}
	for i := range cfg.Registries {
		if cfg.Registries[i].Node == cfg.Current {
			return &cfg.Registries[i]
		}
	}
	return nil
}

// findRegistry returns the registry entry matching the given node, or nil.
func findRegistry(cfg *RegistryConfig, node string) *RegistryEntry {
	for i := range cfg.Registries {
		if cfg.Registries[i].Node == node {
			return &cfg.Registries[i]
		}
	}
	return nil
}

func AddOrUpdateRegistry(cfg *RegistryConfig, entry RegistryEntry) {
	for i, r := range cfg.Registries {
		if r.Node == entry.Node {
			cfg.Registries[i] = entry
			cfg.Current = entry.Node
			return
		}
	}
	cfg.Registries = append(cfg.Registries, entry)
	cfg.Current = entry.Node
}

func RemoveRegistry(cfg *RegistryConfig, node string) {
	for i, r := range cfg.Registries {
		if r.Node == node {
			cfg.Registries = append(cfg.Registries[:i], cfg.Registries[i+1:]...)
			break
		}
	}
	if cfg.Current == node {
		cfg.Current = ""
		if len(cfg.Registries) > 0 {
			cfg.Current = cfg.Registries[0].Node
		}
	}
}

// newEntryFromOptions builds a RegistryEntry from the current RegistryOptions state.
func newEntryFromOptions(o *RegistryOptions) RegistryEntry {
	entry := RegistryEntry{
		Node: o.Node,
		Port: o.RegistryPort,
	}
	if o.SSHConfig != nil {
		entry.SSH = RegistrySSH{
			User:     o.SSHConfig.User,
			PkFile:   o.SSHConfig.PkFile,
			PkPasswd: o.SSHConfig.PkPassword,
		}
	}
	return entry
}

// applyEntryToOptions fills RegistryOptions fields from a RegistryEntry for any unset values.
// CLI-specified flags take precedence; config values only fill in gaps.
func applyEntryToOptions(o *RegistryOptions, entry *RegistryEntry, nodeChanged, portChanged, sshUserChanged, pkFileChanged, pkPasswdChanged bool) {
	if !nodeChanged && entry.Node != "" {
		o.Node = entry.Node
	}
	if !portChanged && entry.Port != 0 {
		o.RegistryPort = entry.Port
	}
	if o.SSHConfig != nil && entry.SSH.User != "" {
		if !sshUserChanged && o.SSHConfig.User == "" {
			o.SSHConfig.User = entry.SSH.User
		}
		if !pkFileChanged && o.SSHConfig.PkFile == "" && entry.SSH.PkFile != "" {
			o.SSHConfig.PkFile = entry.SSH.PkFile
		}
		if !pkPasswdChanged && o.SSHConfig.PkPassword == "" && entry.SSH.PkPasswd != "" {
			o.SSHConfig.PkPassword = entry.SSH.PkPasswd
		}
	}
}
