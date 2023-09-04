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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	pkgerr "github.com/pkg/errors"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
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
	// ClientCertificate is the path to a client cert file for TLS.
	ClientCertificate string `json:"client-certificate,omitempty"`
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	ClientCertificateData []byte `json:"client-certificate-data,omitempty"`
	// ClientKey is the path to a client key file for TLS.
	ClientKey string `json:"client-key,omitempty"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	ClientKeyData []byte `json:"client-key-data,omitempty" datapolicy:"security-key"`
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
	return &Config{
		Servers:        make(map[string]*Server),
		AuthInfos:      make(map[string]*AuthInfo),
		CurrentContext: "",
		Contexts:       make(map[string]*Context),
	}
}

func TryLoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := New()
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Merge two config,if conflict override old.
func (c *Config) Merge(newCfg *Config) {
	for k, server := range newCfg.Servers {
		c.Servers[k] = server
	}
	for k, auth := range newCfg.AuthInfos {
		c.AuthInfos[k] = auth
	}
	for k, context := range newCfg.Contexts {
		c.Contexts[k] = context
	}
	c.CurrentContext = newCfg.CurrentContext
}

func (c *Config) Dump() error {
	if err := c.Flatten(); err != nil {
		return pkgerr.WithMessage(err, "flatten failed")
	}
	cfgBytes, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	cfgBytes, err = yaml.JSONToYAML(cfgBytes)
	if err != nil {
		return err
	}
	fpath := filepath.Join(homedir.HomeDir(), DefaultConfigPath)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		if err := os.MkdirAll(fpath, os.ModeDir|0755); err != nil {
			return err
		}
	}
	f, err := os.Create(filepath.Join(fpath, "config"))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(cfgBytes)
	return err
}

// Flatten changes the config object into a selfContained config (useful for making secrets)
func (c *Config) Flatten() error {
	for key, server := range c.Servers {
		if err := flattenContent(&server.CertificateAuthority, &server.CertificateAuthorityData); err != nil {
			return err
		}

		c.Servers[key] = server
	}
	for key, authInfo := range c.AuthInfos {
		if err := flattenContent(&authInfo.ClientCertificate, &authInfo.ClientCertificateData); err != nil {
			return err
		}
		if err := flattenContent(&authInfo.ClientKey, &authInfo.ClientKeyData); err != nil {
			return err
		}

		c.AuthInfos[key] = authInfo
	}
	return nil
}

func flattenContent(path *string, contents *[]byte) error {
	if len(*path) != 0 {
		if len(*contents) > 0 {
			return errors.New("cannot have values for both path and contents")
		}

		var err error
		*contents, err = os.ReadFile(*path)
		if err != nil {
			return err
		}

		*path = ""
	}

	return nil
}
