/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package identityprovider

import (
	"errors"
	"fmt"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

var (
	oauthProviderFactories = make(map[string]OAuthProviderFactory)
	oauthProviders         = make(map[string]OAuthProvider)
)

var (
	errIdentityProviderNotFound = errors.New("identity provider not found")
)

// SetupWithOptions will verify the configuration and initialize the identityProviders
func SetupWithOptions(options []oauth.IdentityProviderOptions) error {
	for _, o := range options {
		if oauthProviders[o.Name] != nil {
			return fmt.Errorf("duplicate identity provider found: %s", o.Name)
		}
		if oauthProviderFactories[o.Type] == nil {
			return fmt.Errorf("identity provider %s with type %s is not supported", o.Name, o.Type)
		}
		if factory, ok := oauthProviderFactories[o.Type]; ok {
			if provider, err := factory.Create(o.Provider); err != nil {
				// don’t return errors, decoupling external dependencies
				logger.Errorf("failed to create identity provider %s: %s", o.Name, err)
			} else {
				oauthProviders[o.Name] = provider
				logger.Debugf("create identity provider %s successfully", o.Name)
			}
		}
	}
	return nil
}

// GetOAuthProvider returns OAuthProvider with given name
func GetOAuthProvider(providerName string) (OAuthProvider, error) {
	if provider, ok := oauthProviders[providerName]; ok {
		return provider, nil
	}
	return nil, errIdentityProviderNotFound
}

// RegisterOAuthProvider register OAuthProviderFactory with the specified type
func RegisterOAuthProvider(factory OAuthProviderFactory) {
	oauthProviderFactories[factory.Type()] = factory
}
