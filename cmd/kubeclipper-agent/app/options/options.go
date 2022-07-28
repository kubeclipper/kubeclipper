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
	return errors
}

func (s *AgentOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	s.GenericServerRunOptions.AddFlags(fs, s.GenericServerRunOptions)
	s.LogOptions.AddFlags(fss.FlagSet("log"))
	s.MQOptions.AddFlags(fss.FlagSet("mq"))
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
