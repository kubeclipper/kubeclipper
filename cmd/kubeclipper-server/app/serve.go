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

	"k8s.io/klog/v2"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/kubeclipper/kubeclipper/cmd/kubeclipper-server/app/options"
	serverconfig "github.com/kubeclipper/kubeclipper/pkg/server/config"
)

func newServeCommand(stopCh <-chan struct{}) *cobra.Command {
	s := options.NewServerOptions()
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch a kubeclipper-server",
		Long:  "TODO: add long description for kubeclipper-server",
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

func Run(s *options.ServerOptions, stopCh <-chan struct{}) error {
	apiserver, err := s.NewAPIServer(stopCh)
	if err != nil {
		return err
	}

	err = apiserver.PrepareRun(stopCh)
	if err != nil {
		return err
	}

	return apiserver.Run(stopCh)
}

func completionOptions(s *options.ServerOptions) (*options.ServerOptions, error) {
	conf, err := serverconfig.TryLoadFromDisk()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("config file not found, use cmd line...")
		} else {
			return nil, fmt.Errorf("error parsing configuration file %s", err)
		}
	} else {
		s = &options.ServerOptions{
			Config: conf,
		}
	}
	logger.ApplyZapLoggerWithOptions(s.Config.LogOptions)
	klog.SetLogger(zapr.NewLogger(logger.ZapLogger("klog")))
	return s, nil
}
