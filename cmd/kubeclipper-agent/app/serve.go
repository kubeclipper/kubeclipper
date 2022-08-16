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

package app

import (
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/proxy"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/kubeclipper/kubeclipper/cmd/kubeclipper-agent/app/options"
	agentconfig "github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func newServeCommand(stopCh <-chan struct{}) *cobra.Command {
	s := options.NewAgentOptions()
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch a kubeclipper-agent",
		Long:  "TODO: add long description for kubeclipper-agent",
		RunE: func(c *cobra.Command, args []string) error {
			s, err := completionOptions(s)
			if err != nil {
				return err
			}
			if errs := s.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}
			return Run(s, stopCh)
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	return cmd
}

func completionOptions(s *options.AgentOptions) (*options.AgentOptions, error) {
	conf, err := agentconfig.TryLoadFromDisk()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// always use config file
			return nil, fmt.Errorf("config file not found, now only support config file to init server")
		}
		return nil, fmt.Errorf("error parsing configuration file %s", err)
	}
	proxy.ReplaceByProxy(conf)

	s = &options.AgentOptions{
		GenericServerRunOptions: s.GenericServerRunOptions,
		Config:                  conf,
	}
	if s.Config.AgentID == "" {
		s.Config.AgentID = uuid.New().String()
		agentconfig.SetConfig("agentID", s.Config.AgentID)
		if err := agentconfig.TrySaveToDisk(s.Config); err != nil {
			return nil, err
		}
	}
	logger.ApplyZapLoggerWithOptions(s.Config.LogOptions)
	downloader.SetOptions(s.Config.DownloaderOptions)
	return s, nil
}

func Run(s *options.AgentOptions, stopCh <-chan struct{}) error {
	server, err := s.NewServer(stopCh)
	if err != nil {
		return err
	}

	err = server.PrepareRun(stopCh)
	if err != nil {
		return err
	}

	return server.Run(stopCh)
}
