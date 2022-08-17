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

package oauth

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type MappingMethod string
type IdentityProviderType string

const (
	// MappingMethodAuto  The default value.The user will automatically create and mapping when login successful.
	// Fails if a user with that username is already mapped to another identity.
	MappingMethodAuto MappingMethod = "auto"
	// MappingMethodLookup Looks up an existing identity, user identity mapping, and user, but does not automatically
	// provision users or identities. Using this method requires you to manually provision users.
	// not supported yet.
	MappingMethodLookup MappingMethod = "lookup"
	// MappingMethodMixed  A user entity can be mapped with multiple identifyProvider.
	// not supported yet.
	MappingMethodMixed MappingMethod = "mixed"
)

var (
	ErrorProviderNotFound = errors.New("the identity provider was not found")
)

type Token struct {
	// AccessToken is the token that authorizes and authenticates
	// the requests.
	AccessToken string `json:"access_token"`

	// TokenType is the type of token.
	// The Type method returns either this or "Bearer", the default.
	TokenType string `json:"token_type,omitempty"`

	// RefreshToken is a token that's used by the application
	// (as opposed to the user) to refresh the access token
	// if it expires.
	RefreshToken string `json:"refresh_token,omitempty"`

	// ExpiresIn is the optional expiration second of the access token.
	ExpiresIn int `json:"expires_in,omitempty"`

	// RefreshExpiresIn is the optional expiration second of the refresh token.
	RefreshExpiresIn int `json:"refresh_expires_in,omitempty"`
}

type Options struct {
	IdentityProviders            []IdentityProviderOptions `json:"identityProviders,omitempty" yaml:"identityProviders,omitempty"`
	AccessTokenMaxAge            time.Duration             `json:"accessTokenMaxAge" yaml:"accessTokenMaxAge"`
	AccessTokenInactivityTimeout time.Duration             `json:"accessTokenInactivityTimeout" yaml:"accessTokenInactivityTimeout"`
}

type IdentityProviderOptions struct {
	// The provider name.
	Name string `json:"name" yaml:"name"`

	// Defines how new identities are mapped to users when they login. Allowed values are:
	//  - auto:   The default value.The user will automatically create and mapping when login successful.
	//            Fails if a user with that user name is already mapped to another identity.
	//  - lookup: Looks up an existing identity, user identity mapping, and user, but does not automatically
	//            provision users or identities. Using this method requires you to manually provision users.
	//  - mixed:  A user entity can be mapped with multiple identifyProvider.
	MappingMethod MappingMethod `json:"mappingMethod" yaml:"mappingMethod"`

	// The type of identify provider
	// OpenIDIdentityProvider LDAPIdentityProvider GitHubIdentityProvider
	Type string `json:"type" yaml:"type"`

	// The options of identify provider
	Provider DynamicOptions `json:"provider" yaml:"provider"`
}

type DynamicOptions map[string]interface{}

func (o DynamicOptions) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(desensitize(o))
	return data, err
}

var (
	sensitiveKeys = [...]string{"password", "secret"}
)

func isSensitiveData(key string) bool {
	for _, v := range sensitiveKeys {
		if strings.Contains(strings.ToLower(key), v) {
			return true
		}
	}
	return false
}

func desensitize(data map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for k, v := range data {
		if isSensitiveData(k) {
			continue
		}
		switch temp := v.(type) {
		case map[interface{}]interface{}:
			output[k] = desensitize(convert(temp))
		default:
			output[k] = v
		}
	}
	return output
}

func convert(m map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for k, v := range m {
		switch temp := k.(type) {
		case string:
			output[temp] = v
		}
	}
	return output
}

func (o *Options) IdentityProviderOptions(name string) (*IdentityProviderOptions, error) {
	for _, found := range o.IdentityProviders {
		if found.Name == name {
			return &found, nil
		}
	}
	return nil, ErrorProviderNotFound
}

func NewOauthOptions() *Options {
	return &Options{
		IdentityProviders:            make([]IdentityProviderOptions, 0),
		AccessTokenMaxAge:            time.Hour * 2,
		AccessTokenInactivityTimeout: time.Hour * 2,
	}
}
