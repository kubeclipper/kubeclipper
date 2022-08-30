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
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var (
	JSONContentTypeHeader = map[string][]string{
		"Content-Type": {"application/json"},
	}
)

type Opt func(*Client) error

func WithHost(host string) Opt {
	return func(c *Client) error {
		host = strings.TrimSuffix(host, "/")
		//c.host = host

		hostURL, err := ParseHostURL(host)
		if err != nil {
			return err
		}
		c.host = hostURL.Host
		c.scheme = hostURL.Scheme

		//c.basePath = host
		return nil
	}
}

// ParseHostURL parses a url string, validates the string is a host url, and
// returns the parsed URL
func ParseHostURL(host string) (*url.URL, error) {
	protoAddrParts := strings.SplitN(host, "://", 2)
	if len(protoAddrParts) == 1 {
		return nil, fmt.Errorf("unable to parse host `%s`", host)
	}

	proto, addr := protoAddrParts[0], protoAddrParts[1]
	return &url.URL{
		Scheme: proto,
		Host:   addr,
	}, nil
}

// WithHTTPClient overrides the client http client with the specified one
func WithHTTPClient(client *http.Client) Opt {
	return func(c *Client) error {
		if client != nil {
			c.client = client
		}
		return nil
	}
}

// WithScheme overrides the client scheme with the specified one
func WithScheme(scheme string) Opt {
	return func(c *Client) error {
		c.scheme = scheme
		return nil
	}
}

func WithBearerAuth(token string) Opt {
	return func(c *Client) error {
		c.bearerToken = token
		return nil
	}
}
