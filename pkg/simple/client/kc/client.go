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
	"net/http"
	"net/url"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
)

const (
	defaultHTTPScheme = "http"
)

type Client struct {
	client      *http.Client
	host        string
	bearerToken string
	basePath    string
	scheme      string
}

func FromConfig(c config.Config) (*Client, error) {
	ctx := c.Contexts[c.CurrentContext]

	return NewClientWithOpts(WithHost(c.Servers[ctx.Server].Server),
		WithScheme("http"),
		WithBearerAuth(c.AuthInfos[ctx.AuthInfo].Token))
}

func NewClientWithOpts(opts ...Opt) (*Client, error) {

	c := &Client{
		client: http.DefaultClient,
		scheme: defaultHTTPScheme,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
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
