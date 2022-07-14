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

package auth

import (
	"net/http"
	"net/url"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
)

type TokenManagementInterface interface {
	// Verify verifies a token, and return a User if it's a valid token, otherwise return error
	Verify(token string) (user.Info, error)
	// IssueTo issues a token a User, return error if issuing process failed
	IssueTo(user user.Info) (*oauth.Token, error)
	// RevokeAllUserTokens revoke all user tokens
	RevokeAllUserTokens(username string) error
}

type PasswordAuthenticator interface {
	Authenticate(username, password string) (user.Info, string, error)
}

type OAuthAuthenticator interface {
	Authenticate(provider string, req *http.Request) (user.Info, string, error)
}

type UserMFAProviders struct {
	Providers []mfa.UserMFAProvider `json:"providers,omitempty"`
}

type MFAAuthenticator interface {
	Enabled() bool
	Providers(info user.Info) (UserMFAProviders, error)
	ProviderRequest(provider mfa.UserMFAProvider) error
	Authenticate(provider string, token string, req url.Values) (user.Info, error)
}
