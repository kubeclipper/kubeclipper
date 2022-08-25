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

package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	DefaultConfigPath = ".kc"
	DefaultPkgPath    = "/tmp"
)

type Context struct {
	AuthInfo string `json:"user" yaml:"user"`
	Server   string `json:"server" yaml:"server"`
}

type AuthInfo struct {
	Token string `json:"token,omitempty" yaml:"token,omitempty"`
}

type Server struct {
	Server                   string `json:"server" yaml:"server"`
	TLSServerName            string `json:"tls-server-name,omitempty" yaml:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify    bool   `json:"insecure-skip-tls-verify,omitempty" yaml:"insecure-skip-tls-verify,omitempty"`
	CertificateAuthority     string `json:"certificate-authority,omitempty" yaml:"certificate-authority,omitempty"`
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty" yaml:"certificate-authority-data,omitempty"`
}

type Config struct {
	Servers        map[string]*Server   `json:"servers" yaml:"servers"`
	AuthInfos      map[string]*AuthInfo `json:"users" yaml:"users"`
	CurrentContext string               `json:"current-context" yaml:"current-context"`
	Contexts       map[string]*Context  `json:"contexts" yaml:"contexts"`
}

func New() *Config {
	return &Config{}
}

func TryLoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := New()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
