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

	"github.com/spf13/viper"

	"github.com/kubeclipper/kubeclipper/pkg/proxy/config"
)

type ProxyOptions struct {
	*config.Config
}

func NewProxyOptions() *ProxyOptions {
	return &ProxyOptions{}
}

func (s *ProxyOptions) CompletionOptions() error {
	conf, err := config.TryLoadFromDisk()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// always use config file
			return fmt.Errorf("config file not found, now only support config file to init server")
		}
		return fmt.Errorf("error parsing configuration file %s", err)
	}
	s.Config = conf
	return nil
}

func (s *ProxyOptions) Validate() []error {
	var errors []error
	if s.DefaultMQPort == 0 {
		errors = append(errors, fmt.Errorf("defaultMQPort can't be empty"))
	}
	if s.DefaultStaticServerPort == 0 {
		errors = append(errors, fmt.Errorf("defaultStaticServerPort can't be empty"))
	}

	if s.DefaultMQPort != 0 && s.DefaultMQPort == s.DefaultStaticServerPort {
		errors = append(errors, fmt.Errorf("defaultMQPort(%v) can't equal defaultStaticServerPort(%v)", s.DefaultMQPort, s.DefaultStaticServerPort))
	}
	return errors
}
