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

package config

import (
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

const (
	// DefaultConfigurationName is the default name of configuration
	defaultConfigurationName = "kubeclipper-proxy"

	// DefaultConfigurationPath the default location of the configuration file
	defaultConfigurationPath = "/etc/kubeclipper-proxy"
)

type Config struct {
	ServerIP                string        `json:"serverIP,omitempty" yaml:"serverIP"`
	DefaultMQPort           int           `json:"defaultMQPort,omitempty" yaml:"defaultMQPort"`
	DefaultStaticServerPort int           `json:"defaultStaticServerPort,omitempty" yaml:"defaultStaticServerPort"`
	Config                  []AgentConfig `json:"config,omitempty" yaml:"config"`
}

type AgentConfig struct {
	Name    string   `json:"name,omitempty" yaml:"name"`
	Tunnels []Tunnel `json:"tunnels,omitempty" yaml:"tunnels"`
}

type Tunnel struct {
	LocalPort     int    `json:"localPort,omitempty" yaml:"localPort"`
	RemoteAddress string `json:"remoteAddress,omitempty" yaml:"remoteAddress"`
}

var (
	config    *Config
	once      sync.Once
	configErr error
)

func New() *Config {
	return &Config{
		DefaultMQPort:           9889,
		DefaultStaticServerPort: 8081,
	}
}

func TryLoadFromDisk() (*Config, error) {
	once.Do(func() {
		viper.SetConfigName(defaultConfigurationName)
		viper.AddConfigPath(defaultConfigurationPath)
		// Load from current working directory, only used for debugging
		viper.AddConfigPath(".")

		if configErr = viper.ReadInConfig(); configErr != nil {
			logger.Error("read config fail", zap.Error(configErr))
			return
		}

		config = New()
		if configErr = viper.Unmarshal(config); configErr != nil {
			logger.Error("unmarshal config fail", zap.Error(configErr))
			return
		}
	})

	if configErr != nil {
		return nil, configErr
	}
	return config, nil
}
