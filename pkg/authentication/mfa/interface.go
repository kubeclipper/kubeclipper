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

package mfa

import (
	"fmt"
	"net/url"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
)

var (
	ErrProviderNotFound = fmt.Errorf("provider not found")
)

type Provider interface {
	Verify(req url.Values, info user.Info) error
	Request(info user.Info) error
	UserProviderConfig(info user.Info, token string) UserMFAProvider
}

type ProviderFactory interface {
	Type() string
	Create(cache cache.Interface, options oauth.DynamicOptions) (Provider, error)
}

var (
	mfaProviderFactories = make(map[string]ProviderFactory)
	mfaProviders         = make(map[string]Provider)
)

func RegisterProviderFactory(factory ProviderFactory) {
	kind := factory.Type()
	if _, ok := mfaProviderFactories[kind]; ok {
		panic(fmt.Errorf("already registered type: %s", kind))
	}
	mfaProviderFactories[kind] = factory
}

func GetProvider(providerType string) (Provider, error) {
	if provider, ok := mfaProviders[providerType]; ok {
		return provider, nil
	}
	return nil, ErrProviderNotFound
}

func SetupWithOptions(p cache.Interface, opts *Options) error {
	if opts == nil || !opts.Enabled {
		return nil
	}
	for _, o := range opts.MFAProviders {
		if mfaProviders[o.Type] != nil {
			return fmt.Errorf("duplicate mfa provider type found: %s", o.Type)
		}
		if mfaProviderFactories[o.Type] == nil {
			return fmt.Errorf("mfa provider %s is not supported", o.Type)
		}
		if factory, ok := mfaProviderFactories[o.Type]; ok {
			if provider, err := factory.Create(p, o.Options); err != nil {
				logger.Errorf("failed to create mfa provider %s: %s", o.Type, err)
			} else {
				mfaProviders[o.Type] = provider
				logger.Debugf("create mfa provider %s successfully", o.Type)
			}
		}
	}
	return nil
}

type UserMFAProvider struct {
	Type  string `json:"type,omitempty"`
	Token string `json:"token,omitempty"`
	Value string `json:"value,omitempty"`
}
