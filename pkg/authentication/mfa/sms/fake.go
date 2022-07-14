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

package sms

import (
	"fmt"
	"net/url"
	"time"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"
)

func init() {
	mfa.RegisterProviderFactory(&fakeSMSProviderFactory{})
}

const (
	FakeSMSProvider = "fake_sms"
)

type fakeSMSProviderFactory struct {
}

func (d *fakeSMSProviderFactory) Type() string {
	return FakeSMSProvider
}

func (d *fakeSMSProviderFactory) Create(cache cache.Interface, options oauth.DynamicOptions) (mfa.Provider, error) {
	ttl, err := getDuration(options, "ttl")
	if err != nil {
		return nil, err
	}
	return &fakeSMSProvider{
		ttl:   ttl,
		cache: cache,
	}, nil
}

type fakeSMSProvider struct {
	ttl   time.Duration
	cache cache.Interface
}

func (d *fakeSMSProvider) Verify(req url.Values, user user.Info) error {
	phone := user.GetExtra()["phone"][0]
	code := req.Get("code")

	key := smsCacheKey(FakeSMSProvider, phone)

	val, err := d.cache.Get(key)
	if err != nil {
		return err
	}
	if val != code {
		return fmt.Errorf("incorrect verification code")
	}
	_ = d.cache.Remove(key)
	return nil
}

func (d *fakeSMSProvider) Request(user user.Info) error {
	phone := user.GetExtra()["phone"][0]
	key := smsCacheKey(FakeSMSProvider, phone)
	err := rateLimit(d.cache, key, smsSendInterval)
	if err != nil {
		return err
	}

	code := generateNumberCode(6)
	err = d.cache.Set(smsCacheKey(FakeSMSProvider, phone), code, d.ttl)
	if err != nil {
		return err
	}
	fmt.Printf("fake SMS Verification code:%s phone: %s \n", code, phone)

	return nil
}

func (d fakeSMSProvider) UserProviderConfig(info user.Info, token string) mfa.UserMFAProvider {
	phone := info.GetExtra()["phone"][0]
	return mfa.UserMFAProvider{
		Type:  FakeSMSProvider,
		Token: token,
		Value: sliceutil.StringMask(phone, 3, 8, '*'),
	}
}

func getDuration(options oauth.DynamicOptions, key string) (time.Duration, error) {
	v, ok := options[key]
	if !ok {
		return 0, fmt.Errorf("not exists key: %s", key)
	}
	s, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("%s must be string", key)
	}
	return time.ParseDuration(s)
}
