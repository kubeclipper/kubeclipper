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
	cli, err := FromConfigWithoutValidation(c)
	if err != nil {
		return nil, err
	}

	// call api to check whether the token is valid
	q := query.New()
	if _, err = cli.ListConfigMaps(context.TODO(), Queries(*q)); err != nil {
		if strings.Contains(err.Error(), "Unauthorized") {
			return nil, errors.New("unauthorized,please use kcctl login cmd to login first")
		}
		return nil, err
	}
	return cli, nil
}

// FromConfigWithoutValidation creates a client without making a preliminary API request.
// Diagnostic commands use it so they can report a partially unavailable platform.
func FromConfigWithoutValidation(c config.Config) (*Client, error) {
	currentServer, currentAuthInfo, err := currentConnectionConfig(c)
	if err != nil {
		return nil, err
	}
	opts, err := serverClientOptions(currentServer)
	if err != nil {
		return nil, err
	}
	authOpts, err := authClientOptions(currentAuthInfo)
	if err != nil {
		return nil, err
	}
	return NewClientWithOpts(append(opts, authOpts...)...)
}

func currentConnectionConfig(c config.Config) (*config.Server, *config.AuthInfo, error) {
	ctx, ok := c.Contexts[c.CurrentContext]
	if !ok || ctx == nil {
		return nil, nil, fmt.Errorf("current context %q is not configured", c.CurrentContext)
	}
	currentServer, ok := c.Servers[ctx.Server]
	if !ok || currentServer == nil {
		return nil, nil, fmt.Errorf("server %q is not configured", ctx.Server)
	}
	currentAuthInfo, ok := c.AuthInfos[ctx.AuthInfo]
	if !ok || currentAuthInfo == nil {
		return nil, nil, fmt.Errorf("user %q is not configured", ctx.AuthInfo)
	}
	return currentServer, currentAuthInfo, nil
}

func serverClientOptions(currentServer *config.Server) ([]Opt, error) {
	opts := []Opt{
		WithEndpoint(currentServer.Server),
	}
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
	return opts, nil
}

func authClientOptions(currentAuthInfo *config.AuthInfo) ([]Opt, error) {
	opts := []Opt{WithBearerAuth(currentAuthInfo.Token)}
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
	return opts, nil
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
	if cli.host == "" {
		return errors.New("host must not be empty")
	}
	if cli.scheme == "" {
		return errors.New("scheme must not be empty")
	}
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
