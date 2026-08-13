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

package options

import (
	"fmt"
	"os"

	cliflag "k8s.io/component-base/cli/flag"

	"github.com/kubeclipper/kubeclipper/pkg/agent"
	agentconfig "github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/simple/generic"
)

type AgentOptions struct {
	GenericServerRunOptions *generic.ServerRunOptions
	*agentconfig.Config
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		GenericServerRunOptions: generic.NewServerRunOptions(),
		Config:                  agentconfig.New(),
	}
}

func (s *AgentOptions) Validate() []error {
	var errors []error
	errors = append(errors, s.GenericServerRunOptions.Validate()...)
	errors = append(errors, s.LogOptions.Validate()...)
	errors = append(errors, s.OpLogOptions.Validate()...)
	errors = append(errors, s.ImageProxyOptions.Validate()...)
	if s.APIServer == nil || s.APIServer.Endpoint == "" {
		errors = append(errors, fmt.Errorf("apiServer.endpoint is required"))
	} else {
		for name, path := range map[string]string{"apiServer.caFile": s.APIServer.CAFile, "apiServer.certFile": s.APIServer.CertFile, "apiServer.keyFile": s.APIServer.KeyFile} {
			if path == "" {
				errors = append(errors, fmt.Errorf("%s is required", name))
			} else if _, err := os.Stat(path); err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
			}
		}
	}
	return errors
}

func (s *AgentOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	s.GenericServerRunOptions.AddFlags(fs, s.GenericServerRunOptions)
	s.LogOptions.AddFlags(fss.FlagSet("log"))
	s.OpLogOptions.AddFlags(fss.FlagSet("oplog"))
	s.ImageProxyOptions.AddFlags(fss.FlagSet("imageProxy"))

	return fss
}

func (s *AgentOptions) NewServer(stopCh <-chan struct{}) (*agent.Server, error) {
	server := &agent.Server{
		Config: s.Config,
	}
	return server, nil
}
