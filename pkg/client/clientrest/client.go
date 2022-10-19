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

package clientrest

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

func RESTClientFor(config *rest.Config) (rest.Interface, error) {
	if config.GroupVersion == nil {
		return nil, fmt.Errorf("GroupVersion is required when initializing a RESTClient")
	}
	if config.NegotiatedSerializer == nil {
		return nil, fmt.Errorf("NegotiatedSerializer is required when initializing a RESTClient")
	}

	baseURL, versionedAPIPath, err := defaultServerURLFor(config)
	if err != nil {
		return nil, err
	}

	transport, err := TransportFor(config)
	if err != nil {
		return nil, err
	}

	var httpClient *http.Client
	if transport != http.DefaultTransport {
		httpClient = &http.Client{Transport: transport}
		if config.Timeout > 0 {
			httpClient.Timeout = config.Timeout
		}
	}

	rateLimiter := config.RateLimiter
	if rateLimiter == nil {
		qps := config.QPS
		if config.QPS == 0.0 {
			qps = rest.DefaultQPS
		}
		burst := config.Burst
		if config.Burst == 0 {
			burst = rest.DefaultBurst
		}
		if qps > 0 {
			rateLimiter = flowcontrol.NewTokenBucketRateLimiter(qps, burst)
		}
	}

	var gv schema.GroupVersion
	if config.GroupVersion != nil {
		gv = *config.GroupVersion
	}
	clientContent := rest.ClientContentConfig{
		AcceptContentTypes: config.AcceptContentTypes,
		ContentType:        config.ContentType,
		GroupVersion:       gv,
		Negotiator:         runtime.NewClientNegotiator(config.NegotiatedSerializer, gv),
	}

	return rest.NewRESTClient(baseURL, versionedAPIPath, clientContent, rateLimiter, httpClient)
}

func IsInformerRawQuery(req *http.Request) bool {
	if v := req.Header.Get(QueryTypeHeader); v == InformerQuery {
		return true
	}
	return false
}

func InternalRestConfig(address, username, token string) *rest.Config {
	return &rest.Config{
		Host:                address,
		APIPath:             "",
		ContentConfig:       rest.ContentConfig{},
		Username:            username,
		Password:            token,
		BearerToken:         "",
		BearerTokenFile:     "",
		Impersonate:         rest.ImpersonationConfig{},
		AuthProvider:        nil,
		AuthConfigPersister: nil,
		ExecProvider:        nil,
		TLSClientConfig:     rest.TLSClientConfig{},
		UserAgent:           "",
		DisableCompression:  false,
		Transport:           nil,
		WrapTransport:       nil,
		QPS:                 0,
		Burst:               0,
		RateLimiter:         nil,
		WarningHandler:      nil,
		Timeout:             0,
		Dial:                nil,
		Proxy:               nil,
	}
}
