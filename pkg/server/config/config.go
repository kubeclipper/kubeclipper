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
	"reflect"
	"strings"

	auditoptions "github.com/kubeclipper/kubeclipper/pkg/auditing/option"

	authoptions "github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"

	"github.com/kubeclipper/kubeclipper/pkg/simple/staticserver"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"github.com/kubeclipper/kubeclipper/pkg/simple/generic"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/etcd"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"

	"github.com/spf13/viper"
)

const (
	// DefaultConfigurationName is the default name of configuration
	DefaultConfigurationName = "kubeclipper-server"

	// DefaultConfigurationPath the default location of the configuration file
	DefaultConfigurationPath = "/etc/kubeclipper-server"
)

// Config defines everything needed for apiserver to deal with external services
type Config struct {
	GenericServerRunOptions *generic.ServerRunOptions          `json:"generic" yaml:"generic" mapstructure:"generic"`
	StaticServerOptions     *staticserver.Options              `json:"staticServer" yaml:"staticServer" mapstructure:"staticServer"`
	EtcdOptions             *etcd.Options                      `json:"etcd,omitempty" yaml:"etcd,omitempty" mapstructure:"etcd"`
	CacheOptions            *cache.Options                     `json:"cache,omitempty" yaml:"cache,omitempty" mapstructure:"cache"`
	MQOptions               *natsio.NatsOptions                `json:"mq,omitempty" yaml:"mq,omitempty"  mapstructure:"mq"`
	LogOptions              *logger.Options                    `json:"log,omitempty" yaml:"log,omitempty" mapstructure:"log"`
	AuthenticationOptions   *authoptions.AuthenticationOptions `json:"authentication,omitempty" yaml:"authentication,omitempty" mapstructure:"authentication"`
	AuditOptions            *auditoptions.AuditOptions         `json:"audit,omitempty" yaml:"audit,omitempty" mapstructure:"audit"`
}

func New() *Config {
	return &Config{
		GenericServerRunOptions: generic.NewServerRunOptions(),
		StaticServerOptions:     staticserver.NewOptions(),
		EtcdOptions:             etcd.NewEtcdOptions(),
		CacheOptions:            cache.NewEtcdOptions(),
		MQOptions:               natsio.NewOptions(),
		LogOptions:              logger.NewLogOptions(),
		AuthenticationOptions:   authoptions.NewAuthenticateOptions(),
		AuditOptions:            auditoptions.NewAuditOptions(),
	}
}

// ToMap simply converts config to map[string]bool
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

	types := bs.GetProviderFactoryType()
	for _, val := range types {
		result[val] = true
	}
	return result
}

// Remove invalid options before serializing to json or yaml
func (conf *Config) stripEmptyOptions() {
	if conf.EtcdOptions != nil && len(conf.EtcdOptions.ServerList) == 0 {
		conf.EtcdOptions = nil
	}
	if conf.MQOptions != nil && len(conf.MQOptions.Client.ServerAddress) == 0 {
		conf.MQOptions = nil
	}
}

func TryLoadFromDisk() (*Config, error) {
	viper.SetConfigName(DefaultConfigurationName)
	viper.AddConfigPath(DefaultConfigurationPath)
	// Load from current working directory, only used for debugging
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := New()

	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
