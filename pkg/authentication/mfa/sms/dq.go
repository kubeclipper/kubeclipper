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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/mfa"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/cache"
)

func init() {
	mfa.RegisterProviderFactory(&dqSMSProviderFactory{})
}

const (
	DQSMSProvider = "dq_sms"
)

var (
	ErrSMSRateLimitExceeded = fmt.Errorf("verification code was sent too frequently. Please try again later")
	ErrSMSSendLimitExceeded = fmt.Errorf("SMS sending limit reached")
)

type dqSMSProviderFactory struct {
}

func (d *dqSMSProviderFactory) Type() string {
	return DQSMSProvider
}

type dpSMSProvider struct {
	cache  cache.Interface
	Schema string `json:"schema" yaml:"schema"`
	Host   string `json:"host" yaml:"host"`
}

func (d *dqSMSProviderFactory) Create(cache cache.Interface, options oauth.DynamicOptions) (mfa.Provider, error) {
	var dqsms dpSMSProvider
	if err := mapstructure.Decode(options, &dqsms); err != nil {
		return nil, err
	}
	dqsms.cache = cache
	return &dqsms, nil
}

func (d dpSMSProvider) Verify(req url.Values, info user.Info) error {
	phone := info.GetExtra()["phone"][0]
	code := req.Get("code")

	u := d.url("tymh_interface_sms/random_code/verify", url.Values{
		"id":         []string{phone},
		"randomcode": []string{code},
		"type":       []string{"103"},
	})
	result, err := d.commonHTTPRequest(u)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("send verification code:%s", result.ErrorDescription)
	}
	return nil
}

func (d dpSMSProvider) Request(info user.Info) error {
	phone := info.GetExtra()["phone"][0]
	err := rateLimit(d.cache, smsCacheKey(DQSMSProvider, phone), smsSendInterval)
	if err != nil {
		return err
	}

	u := d.url("tymh_interface_sms/random_code_new", url.Values{
		"id":   []string{phone},
		"type": []string{"103"},
	})
	result, err := d.commonHTTPRequest(u)
	if err != nil {
		return err
	}
	switch result.Code {
	case 0:
		return nil
	case 104, 105:
		return ErrSMSSendLimitExceeded
	default:
		return fmt.Errorf("send verification code:%s", result.ErrorDescription)
	}
}

func (d dpSMSProvider) UserProviderConfig(info user.Info, token string) mfa.UserMFAProvider {
	phone := info.GetExtra()["phone"][0]
	return mfa.UserMFAProvider{
		Type:  DQSMSProvider,
		Token: token,
		Value: sliceutil.StringMask(phone, 3, 8, '*'),
	}
}

func (d *dpSMSProvider) url(path string, query url.Values) *url.URL {
	u := url.URL{
		Scheme:   d.Schema,
		Host:     d.Host,
		Path:     path,
		RawQuery: query.Encode(),
	}
	return &u
}

func (d *dpSMSProvider) commonHTTPRequest(u *url.URL) (*commonResponse, error) {
	request, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	bodyData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var result commonResponse
	err = json.Unmarshal(bodyData, err)
	if err != nil {
		return nil, fmt.Errorf("%w:http status:%d body:%s", err, response.StatusCode, bodyData)
	}
	return &result, nil
}

type commonResponse struct {
	Code             int    `json:"code"`
	ErrorDescription string `json:"errorDescription"`
}
