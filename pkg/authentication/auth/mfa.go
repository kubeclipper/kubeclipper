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
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"

	"github.com/google/uuid"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
)

const (
	maxVerificationTimes = 3
)

type mfaAuthenticator struct {
	iamOperator iam.Operator
	cache       cache.Interface
	opts        *mfa.Options
}

func NewMFAAuthenticator(iamOperator iam.Operator, cache cache.Interface, opts *mfa.Options) MFAAuthenticator {
	return &mfaAuthenticator{
		iamOperator: iamOperator,
		cache:       cache,
		opts:        opts,
	}
}

func (m *mfaAuthenticator) Enabled() bool {
	return m.opts.Enabled
}

func (m *mfaAuthenticator) Providers(info user.Info) (UserMFAProviders, error) {
	result := UserMFAProviders{}
	if len(m.opts.MFAProviders) == 0 {
		return UserMFAProviders{}, fmt.Errorf("server not have mfa provider config")
	}
	token := uuid.New().String()
	if err := m.cache.Set(fmt.Sprintf("mfa-%s", token),
		marshalTokenValue(info.GetName(), 0), 15*time.Minute); err != nil {
		return result, err
	}
	for _, providerType := range m.opts.MFAProviders {
		provider, err := mfa.GetProvider(providerType.Type)
		if err != nil {
			return result, err
		}
		result.Providers = append(result.Providers, provider.UserProviderConfig(info, token))
	}
	return result, nil
}

func (m *mfaAuthenticator) ProviderRequest(provider mfa.UserMFAProvider) error {
	value, err := m.cache.Get(fmt.Sprintf("mfa-%s", provider.Token))
	if err != nil {
		return err
	}
	p, err := mfa.GetProvider(provider.Type)
	if err != nil {
		return err
	}
	usrName, _, err := parseTokenValue(value)
	if err != nil {
		return err
	}
	usr, err := m.iamOperator.GetUserEx(context.TODO(), usrName, "0", false, false)
	if err != nil {
		return err
	}
	return p.Request(&user.DefaultInfo{
		Name:   usrName,
		UID:    "",
		Groups: nil,
		Extra: map[string][]string{
			"phone": {usr.Spec.Phone},
			"email": {usr.Spec.Email},
		},
	})
}

func (m *mfaAuthenticator) Authenticate(provider string, token string, values url.Values) (user.Info, error) {
	tokenKey := fmt.Sprintf("mfa-%s", token)
	value, err := m.cache.Get(tokenKey)
	if err != nil {
		return nil, err
	}
	p, err := mfa.GetProvider(provider)
	if err != nil {
		return nil, err
	}

	usrName, failures, err := parseTokenValue(value)
	if err != nil {
		return nil, err
	}
	usr, err := m.iamOperator.GetUserEx(context.TODO(), usrName, "0", false, false)
	if err != nil {
		return nil, err
	}
	info := &user.DefaultInfo{
		Name:   usrName,
		UID:    "",
		Groups: nil,
		Extra: map[string][]string{
			"phone": {usr.Spec.Phone},
			"email": {usr.Spec.Email},
		},
	}
	if err = p.Verify(values, info); err != nil {
		failures++
		if failures >= maxVerificationTimes {
			if e := m.cache.Remove(tokenKey); e != nil {
				logger.Errorf("cache remove key:%s", err)
			}
		} else {
			if e := m.cache.Update(tokenKey, marshalTokenValue(usrName, failures)); e != nil {
				logger.Errorf("cache Update key:%s", err)
			}
		}
		return nil, err
	}
	return info, nil
}

func marshalTokenValue(username string, count int) string {
	return fmt.Sprintf("%s:%d", username, count)

}

func parseTokenValue(value string) (username string, count int, err error) {
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid token value")
	}
	count, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid token value: %w", err)
	}
	return parts[0], count, nil
}
