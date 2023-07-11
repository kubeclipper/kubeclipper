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
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

const (
	KcUserHeader    = "Kc-User"
	KcTokenHeader   = "Kc-Token"
	QueryTypeHeader = "Kc-Query-Type"
	InformerQuery   = "kc-informer"
)

func TransportFor(config *rest.Config) (http.RoundTripper, error) {
	cfg, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	return NewTransport(cfg)
}

func NewTransport(config *transport.Config) (http.RoundTripper, error) {
	// Set transport level security
	if config.Transport != nil && (config.HasCA() || config.HasCertAuth() || config.HasCertCallback() || config.TLS.Insecure) {
		return nil, fmt.Errorf("using a custom transport with TLS certificate options or the insecure flag is not allowed")
	}

	var rt http.RoundTripper
	var err error
	if config.Transport != nil {
		rt = config.Transport
	} else {
		rt, err = initTransport(config)
		if err != nil {
			return nil, err
		}
	}
	return HTTPWrappersForConfig(config, rt)
}

func initTransport(config *transport.Config) (http.RoundTripper, error) {
	// Get the TLS options for this client config
	tlsConfig, err := transport.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}
	// The options didn't require a custom TLS config
	if tlsConfig == nil && config.DialHolder == nil && config.Proxy == nil {
		return http.DefaultTransport, nil
	}

	dialHolder := config.DialHolder
	if dialHolder == nil {
		dial := (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext
		dialHolder = &transport.DialHolder{Dial: dial}
	}

	proxy := http.ProxyFromEnvironment
	if config.Proxy != nil {
		proxy = config.Proxy
	}

	rt := utilnet.SetTransportDefaults(&http.Transport{
		Proxy:               proxy,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 25,
		DialContext:         dialHolder.Dial,
		DisableCompression:  config.DisableCompression,
	})

	return rt, nil
}

// HTTPWrappersForConfig wraps a round tripper with any relevant layered
// behavior from the config. Exposed to allow more clients that need HTTP-like
// behavior but then must hijack the underlying connection (like WebSocket or
// HTTP2 clients). Pure HTTP clients should use the RoundTripper returned from
// New.
func HTTPWrappersForConfig(config *transport.Config, rt http.RoundTripper) (http.RoundTripper, error) {
	if config.WrapTransport != nil {
		rt = config.WrapTransport(rt)
	}
	// Set authentication wrappers
	switch {
	case config.HasBasicAuth() && config.HasTokenAuth():
		return nil, fmt.Errorf("username/password or bearer token may be set, but not both")
	case config.HasBasicAuth():
		rt = NewUserTokenAuthRoundTripper(config.Username, config.Password, rt)
	}
	if len(config.UserAgent) > 0 {
		rt = transport.NewUserAgentRoundTripper(config.UserAgent, rt)
	}
	return rt, nil
}

type userTokenRoundTripper struct {
	username string
	token    string
	rt       http.RoundTripper
}

func NewUserTokenAuthRoundTripper(username, token string, rt http.RoundTripper) http.RoundTripper {
	return &userTokenRoundTripper{username, token, rt}
}

func (rt *userTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return rt.rt.RoundTrip(req)
	}
	req = utilnet.CloneRequest(req)
	req.Header.Add(KcUserHeader, rt.username)
	req.Header.Add(KcTokenHeader, rt.token)
	req.Header.Add(QueryTypeHeader, InformerQuery)
	return rt.rt.RoundTrip(req)
}

func (rt *userTokenRoundTripper) CancelRequest(req *http.Request) {
	tryCancelRequest(rt.WrappedRoundTripper(), req)
}

func (rt *userTokenRoundTripper) WrappedRoundTripper() http.RoundTripper { return rt.rt }

func tryCancelRequest(rt http.RoundTripper, req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	switch rt := rt.(type) {
	case canceler:
		rt.CancelRequest(req)
	case utilnet.RoundTripperWrapper:
		tryCancelRequest(rt.WrappedRoundTripper(), req)
	default:
		logger.Warn("unable to cancel request for", zap.Any("rt_type", rt))
	}
}
