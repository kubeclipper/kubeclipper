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

package backupstore

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/pflag"
)

type StoreType string

type DynamicOptions map[string]interface{}

func (o DynamicOptions) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(o)
	return data, err
}

type Options struct {
	Type     string         `json:"type" yaml:"type"`
	Provider DynamicOptions `json:"provider" yaml:"provider"`
}

func NewOptions() *Options {
	return &Options{
		Type:     "fs",
		Provider: map[string]interface{}{},
	}
}

func (o *Options) Validate() error {
	providers := sets.NewString()
	for key := range providerFactories {
		providers.Insert(key)
	}
	if !providers.Has(o.Type) {
		return fmt.Errorf("unsupport backend store type: %s", o.Type)
	}
	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Type, "backupstore-type", o.Type, "backup store type: fs/s3")
}
