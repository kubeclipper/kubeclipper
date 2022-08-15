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
	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/kubeclipper/kubeclipper/pkg/proxy"

	"github.com/kubeclipper/kubeclipper/cmd/kubeclipper-proxy/app/options"
)

func newServeCommand(stopCh <-chan struct{}) *cobra.Command {
	s := options.NewProxyOptions()
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch a kubeclipper-proxy",
		Long:  "TODO: add long description for kubeclipper-agent",
		RunE: func(c *cobra.Command, args []string) error {
			err := s.CompletionOptions()
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

	return cmd
}

func Run(s *options.ProxyOptions, stopCh <-chan struct{}) error {
	server, err := proxy.NewServer(s, stopCh)
	if err != nil {
		return err
	}
	err = server.PrepareRun()
	if err != nil {
		return err
	}

	return server.Run()
}
