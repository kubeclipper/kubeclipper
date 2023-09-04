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

package kc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/query"
)

const (
	defaultHTTPScheme = "http"
)

type Client struct {
	client                *http.Client
	host                  string
	bearerToken           string
	scheme                string
	insecureSkipTLSVerify bool
	tlsServerName         string
	caPool                *x509.CertPool
	cliCert               *tls.Certificate
}

func FromConfig(c config.Config) (*Client, error) {
	ctx := c.Contexts[c.CurrentContext]
	currentServer := c.Servers[ctx.Server]
	currentAuthInfo := c.AuthInfos[ctx.AuthInfo]
	var opts []Opt
	opts = append(opts,
		WithEndpoint(c.Servers[ctx.Server].Server),
		WithBearerAuth(c.AuthInfos[ctx.AuthInfo].Token))
	if currentServer.TLSServerName != "" {
		opts = append(opts, WithServerName(currentServer.TLSServerName))
	}
	if currentServer.InsecureSkipTLSVerify {
		opts = append(opts, WithInsecureSkipTLSVerify())
	}
	switch {
	case len(currentServer.CertificateAuthorityData) != 0:
		opts = append(opts, WithCAData(currentServer.CertificateAuthorityData))
	case currentServer.CertificateAuthority != "":
		caData, err := os.ReadFile(currentServer.CertificateAuthority)
		if err != nil {
			return nil, err
		}
		opts = append(opts, WithCAData(caData))
	}

	switch {
	case len(currentAuthInfo.ClientCertificateData) != 0 && len(currentAuthInfo.ClientKeyData) != 0:
		opts = append(opts, WithCertData(currentAuthInfo.ClientCertificateData, currentAuthInfo.ClientKeyData))
	case currentAuthInfo.ClientCertificate != "" && currentAuthInfo.ClientKey != "":
		certData, err := os.ReadFile(currentAuthInfo.ClientCertificate)
		if err != nil {
			return nil, err
		}
		keyData, err := os.ReadFile(currentAuthInfo.ClientKey)
		if err != nil {
			return nil, err
		}
		opts = append(opts, WithCertData(certData, keyData))
	}

	cli, err := NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}

	// call api to  check is token valid
	q := query.New()
	if _, err = cli.ListConfigMaps(context.TODO(), Queries(*q)); err != nil {
		if strings.Contains(err.Error(), "Unauthorized") {
			return nil, errors.New("unauthorized,please use kcctl login cmd to login first")
		}
		return nil, err
	}
	return cli, nil
}

func NewClientWithOpts(opts ...Opt) (*Client, error) {
	c := &Client{
		scheme: defaultHTTPScheme,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	// init tlsConfig
	if c.insecureSkipTLSVerify || c.tlsServerName != "" || c.caPool != nil || c.cliCert != nil {
		tr, err := c.httpTransport()
		if err != nil {
			return nil, err
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = new(tls.Config)
		}
		if c.insecureSkipTLSVerify {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
		if c.tlsServerName != "" {
			tr.TLSClientConfig.ServerName = c.tlsServerName
		}
		if c.caPool != nil {
			tr.TLSClientConfig.RootCAs = c.caPool
		}

		if c.cliCert != nil {
			tr.TLSClientConfig.Certificates = []tls.Certificate{*c.cliCert}
		}
	}

	if c.client == nil {
		c.client = http.DefaultClient
	}
	return c, nil
}

// HTTPClient returns a copy of the HTTP client bound to the server
func (cli *Client) HTTPClient() *http.Client {
	return cli.client
}

func (cli *Client) Host() string {
	return cli.host
}

func (cli *Client) Token() string {
	return cli.bearerToken
}

func (cli *Client) Scheme() string {
	return cli.scheme
}

// getAPIPath returns the versioned request path to call the api.
// It appends the query parameters to the path if they are not empty.
func (cli *Client) getAPIPath(ctx context.Context, p string, query url.Values) string {
	return (&url.URL{Path: p, RawQuery: query.Encode()}).String()
}

func (cli *Client) Validate() error {
	// TODO
	return nil
}

func (cli *Client) httpTransport() (*http.Transport, error) {
	if cli.client == nil {
		cli.client = &http.Client{Transport: defaultHTTPTransport()}
	}
	if cli.client.Transport == nil {
		cli.client.Transport = defaultHTTPTransport()
	}
	tr, ok := cli.client.Transport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("http client Transport not is http.Transport")
	}
	return tr, nil

}

func defaultHTTPTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       new(tls.Config),
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
