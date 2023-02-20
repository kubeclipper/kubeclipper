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
	"reflect"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"

	"github.com/kubeclipper/kubeclipper/pkg/simple/imageproxy"

	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
)

const (
	// DefaultConfigurationName is the default name of configuration
	defaultConfigurationName = "kubeclipper-agent"

	// DefaultConfigurationPath the default location of the configuration file
	defaultConfigurationPath = "/etc/kubeclipper-agent"
)

// Config defines everything needed for apiserver to deal with external services
type Config struct {
	AgentID                   string              `json:"agentID,omitempty" yaml:"agentID"`
	Metadata                  options.Metadata    `json:"metadata,omitempty" yaml:"metadata"`
	IPDetect                  string              `json:"ipDetect,omitempty" yaml:"ipDetect"`
	NodeIPDetect              string              `json:"nodeIPDetect,omitempty" yaml:"nodeIPDetect"`
	RegisterNode              bool                `json:"registerNode,omitempty" yaml:"registerNode"`
	NodeStatusUpdateFrequency time.Duration       `json:"nodeStatusUpdateFrequency,omitempty" yaml:"nodeStatusUpdateFrequency"`
	DownloaderOptions         *downloader.Options `json:"downloader" yaml:"downloader" mapstructure:"downloader"`
	LogOptions                *logger.Options     `json:"log,omitempty" yaml:"log,omitempty" mapstructure:"log"`
	MQOptions                 *natsio.NatsOptions `json:"mq,omitempty" yaml:"mq,omitempty"  mapstructure:"mq"`
	OpLogOptions              *oplog.Options      `json:"oplog,omitempty" yaml:"oplog,omitempty" mapstructure:"oplog"`
	ImageProxyOptions         *imageproxy.Options `json:"imageProxy,omitempty" yaml:"imageProxy,omitempty" mapstructure:"imageProxy"`
}

var (
	config    *Config
	once      sync.Once
	configErr error
)

func New() *Config {
	return &Config{
		RegisterNode:              true,
		NodeStatusUpdateFrequency: 5 * time.Minute,
		LogOptions:                logger.NewLogOptions(),
		MQOptions:                 natsio.NewOptions(),
		DownloaderOptions:         downloader.NewOptions(),
		OpLogOptions:              oplog.NewOptions(),
		ImageProxyOptions:         imageproxy.NewOptions(),
	}
}

// ToMap convertToMap simply converts config to map[string]bool
// to hide sensitive information
func (conf *Config) ToMap() map[string]bool {
	conf.stripEmptyOptions()
	result := make(map[string]bool)

	if conf == nil {
		return result
	}

	c := reflect.Indirect(reflect.ValueOf(conf))

	for i := 0; i < c.NumField(); i++ {
		name := strings.Split(c.Type().Field(i).Tag.Get("json"), ",")[0]
		if strings.HasPrefix(name, "-") {
			continue
		}
		if c.Field(i).IsNil() {
			result[name] = false
		} else {
			result[name] = true
		}
	}

	return result
}

// Remove invalid options before serializing to json or yaml
func (conf *Config) stripEmptyOptions() {
	if conf.MQOptions != nil && len(conf.MQOptions.Client.ServerAddress) == 0 {
		conf.MQOptions = nil
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

func SetConfig(key string, value interface{}) {
	viper.Set(key, value)
}

func TrySaveToDisk(c *Config) error {
	filename := viper.ConfigFileUsed()
	flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	f, err := os.OpenFile(filename, flags, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = f.WriteString(string(b))
	if err != nil {
		return err
	}
	return nil
}
