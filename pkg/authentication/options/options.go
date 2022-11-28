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

package options

import (
	"errors"
	"fmt"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"

	"github.com/spf13/pflag"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/identityprovider"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
)

const (
	pwdValidRegex = "^(?=.*\\d)(?=.*[a-z])(?=.*[A-Z]).{8,16}$"
)

type AuthenticationOptions struct {
	AuthenticateRateLimiterMaxTries int            `json:"authenticateRateLimiterMaxTries" yaml:"authenticateRateLimiterMaxTries"`
	AuthenticateRateLimiterDuration time.Duration  `json:"authenticateRateLimiterDuration" yaml:"authenticateRateLimiterDuration"`
	MaximumClockSkew                time.Duration  `json:"maximumClockSkew" yaml:"maximumClockSkew"`
	LoginHistoryRetentionPeriod     time.Duration  `json:"loginHistoryRetentionPeriod" yaml:"loginHistoryRetentionPeriod"`
	LoginHistoryMaximumEntries      int            `json:"loginHistoryMaximumEntries" yaml:"loginHistoryMaximumEntries"`
	MultipleLogin                   bool           `json:"multipleLogin" yaml:"multipleLogin"`
	MFAOptions                      *mfa.Options   `json:"mfaOptions" yaml:"mfaOptions"`
	JwtSecret                       string         `json:"-" yaml:"jwtSecret"`
	InitialPassword                 string         `json:"initialPassword" yaml:"initialPassword"`
	OAuthOptions                    *oauth.Options `json:"oauthOptions" yaml:"oauthOptions"`
}

func NewAuthenticateOptions() *AuthenticationOptions {
	return &AuthenticationOptions{
		AuthenticateRateLimiterMaxTries: 5,
		AuthenticateRateLimiterDuration: time.Minute * 10,
		MaximumClockSkew:                10 * time.Second,
		LoginHistoryRetentionPeriod:     time.Hour * 24 * 7,
		LoginHistoryMaximumEntries:      100,
		MFAOptions:                      mfa.NewOptions(),
		OAuthOptions:                    oauth.NewOauthOptions(),
		MultipleLogin:                   false,
		JwtSecret:                       "kubeclipper",
		InitialPassword:                 constatns.DefaultAdminUserPass,
	}
}

func (a *AuthenticationOptions) Validate() []error {
	var errs []error
	if len(a.JwtSecret) == 0 {
		errs = append(errs, errors.New("JWT secret MUST not be empty"))
	}
	if a.LoginHistoryMaximumEntries <= 0 {
		errs = append(errs, errors.New("loginHistoryMaximumEntries must greater than 0"))
	}
	if a.LoginHistoryRetentionPeriod < 10*time.Minute {
		errs = append(errs, errors.New("loginHistoryRetentionPeriod must greater than 10min"))
	}
	if a.AuthenticateRateLimiterMaxTries > a.LoginHistoryMaximumEntries {
		errs = append(errs, errors.New("authenticateRateLimiterMaxTries MUST not be greater than loginHistoryMaximumEntries"))
	}
	if err := identityprovider.SetupWithOptions(a.OAuthOptions.IdentityProviders); err != nil {
		errs = append(errs, err)
	}

	reg, err := regexp2.Compile(pwdValidRegex, 0)
	if err != nil {
		errs = append(errs, err)
	}
	ok, err := reg.MatchString(a.InitialPassword)
	if err != nil {
		errs = append(errs, err)
	} else if !ok {
		errs = append(errs, fmt.Errorf("password [%s] format error", a.InitialPassword))
	}

	return errs
}

func (a *AuthenticationOptions) AddFlags(fs *pflag.FlagSet) {
	a.MFAOptions.AddFlags(fs)
	fs.IntVar(&a.AuthenticateRateLimiterMaxTries, "authenticate-rate-limiter-max-retries", a.AuthenticateRateLimiterMaxTries, "")
	fs.DurationVar(&a.AuthenticateRateLimiterDuration, "authenticate-rate-limiter-duration", a.AuthenticateRateLimiterDuration, "")
	fs.BoolVar(&a.MultipleLogin, "multiple-login", a.MultipleLogin, "Allow multiple login with the same account, disable means only one user can login at the same time.")
	fs.StringVar(&a.JwtSecret, "jwt-secret", a.JwtSecret, "Secret to sign jwt token, must not be empty.")
	fs.DurationVar(&a.LoginHistoryRetentionPeriod, "login-history-retention-period", a.LoginHistoryRetentionPeriod, "login-history-retention-period defines how long login history should be kept.")
	fs.IntVar(&a.LoginHistoryMaximumEntries, "login-history-maximum-entries", a.LoginHistoryMaximumEntries, "login-history-maximum-entries defines how many entries of login history should be kept.")
	fs.DurationVar(&a.OAuthOptions.AccessTokenMaxAge, "access-token-max-age", a.OAuthOptions.AccessTokenMaxAge, "access-token-max-age control the lifetime of access tokens, 0 means no expiration.")
	fs.DurationVar(&a.MaximumClockSkew, "maximum-clock-skew", a.MaximumClockSkew, "The maximum time difference between the system clocks of the ks-apiserver that issued a JWT and the ks-apiserver that verified the JWT.")
	fs.StringVar(&a.InitialPassword, "initial-password", a.InitialPassword, "admin user password")
}
