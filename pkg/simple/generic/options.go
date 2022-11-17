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

package generic

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

type ServerRunOptions struct {
	BindAddress   string `json:"bindAddress" yaml:"bindAddress"`
	InsecurePort  int    `json:"insecurePort" yaml:"insecurePort"`
	SecurePort    int    `json:"securePort" yaml:"securePort"`
	CACertFile    string `json:"caCertFile" yaml:"caCertFile"`
	TLSCertFile   string `json:"tlsCertFile" yaml:"tlsCertFile"`
	TLSPrivateKey string `json:"tlsPrivateKey" yaml:"tlsPrivateKey"`
}

func NewServerRunOptions() *ServerRunOptions {
	s := ServerRunOptions{
		BindAddress:   "0.0.0.0",
		InsecurePort:  8080,
		SecurePort:    0,
		TLSCertFile:   "",
		TLSPrivateKey: "",
	}
	return &s
}

func (s *ServerRunOptions) Validate() []error {
	var errs []error

	if s.SecurePort == 0 && s.InsecurePort == 0 {
		errs = append(errs, fmt.Errorf("insecure and secure port can not be disabled at the same time"))
	}

	if netutil.IsValidPort(s.SecurePort) {
		if s.TLSCertFile == "" {
			errs = append(errs, fmt.Errorf("tls cert file is empty while secure serving"))
		} else {
			if _, err := os.Stat(s.TLSCertFile); err != nil {
				errs = append(errs, err)
			}
		}

		if s.TLSPrivateKey == "" {
			errs = append(errs, fmt.Errorf("tls private key file is empty while secure serving"))
		} else {
			if _, err := os.Stat(s.TLSPrivateKey); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func (s *ServerRunOptions) AddFlags(fs *pflag.FlagSet, c *ServerRunOptions) {

	fs.StringVar(&s.BindAddress, "bind-address", c.BindAddress, "server bind address")
	fs.IntVar(&s.InsecurePort, "insecure-port", c.InsecurePort, "insecure port number")
	fs.IntVar(&s.SecurePort, "secure-port", s.SecurePort, "secure port number")
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", c.TLSCertFile, "tls cert file")
	fs.StringVar(&s.TLSPrivateKey, "tls-private-key", c.TLSPrivateKey, "tls private key")
}
