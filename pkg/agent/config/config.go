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
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/simple/imageproxy"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"

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
	Region                    string              `json:"region,omitempty" yaml:"region"`
	IPDetect                  string              `json:"ipDetect,omitempty" yaml:"ipDetect"`
	RegisterNode              bool                `json:"registerNode,omitempty" yaml:"registerNode"`
	NodeStatusUpdateFrequency time.Duration       `json:"nodeStatusUpdateFrequency,omitempty" yaml:"nodeStatusUpdateFrequency"`
	DownloaderOptions         *downloader.Options `json:"downloader" yaml:"downloader" mapstructure:"downloader"`
	LogOptions                *logger.Options     `json:"log,omitempty" yaml:"log,omitempty" mapstructure:"log"`
	MQOptions                 *natsio.NatsOptions `json:"mq,omitempty" yaml:"mq,omitempty"  mapstructure:"mq"`
	OpLogOptions              *oplog.Options      `json:"oplog,omitempty" yaml:"oplog,omitempty" mapstructure:"oplog"`
	ImageProxyOptions         *imageproxy.Options `json:"imageProxy,omitempty" yaml:"imageProxy,omitempty" mapstructure:"imageProxy"`
}

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

// convertToMap simply converts config to map[string]bool
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
	viper.SetConfigName(defaultConfigurationName)
	viper.AddConfigPath(defaultConfigurationPath)
	// Load from current working directory, only used for debugging
	viper.AddConfigPath(".")
	// viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := New()

	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}
	logger.Errorf("agent config load success,ip-detect:%s\n", conf.IPDetect)
	return conf, nil
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
