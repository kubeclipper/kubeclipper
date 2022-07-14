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

package cache

import (
	"fmt"

	"github.com/spf13/pflag"
)

const (
	ProviderEtcd   = "etcd"
	ProviderRedis  = "redis"
	ProviderMemory = "memory"
)

type Options struct {
	CacheProvider string        `json:"cacheProvider" yaml:"cacheProvider"`
	RedisOptions  *RedisOptions `json:"redis,omitempty" yaml:"redis,omitempty" mapstructure:"redis"`
}

func NewEtcdOptions() *Options {
	return &Options{
		CacheProvider: ProviderEtcd,
		RedisOptions:  NewRedisOptions(),
	}
}

func (s *Options) Validate() []error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.CacheProvider != ProviderEtcd &&
		s.CacheProvider != ProviderRedis &&
		s.CacheProvider != ProviderMemory {
		errs = append(errs, fmt.Errorf("not support cache provider:%s", s.CacheProvider))
	}
	errs = append(errs, s.RedisOptions.Validate()...)

	return errs
}

func (s *Options) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}
	fs.StringVar(&s.CacheProvider, "cache-provider", s.CacheProvider, "Cache underlying provider, must be one of 'etcd' or 'redis'.")
	s.RedisOptions.AddFlags(fs)
}
