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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	"github.com/pkg/errors"

	apierror "github.com/kubeclipper/kubeclipper/pkg/errors"
)

type headers map[string][]string

type Queries query.Query

func (q *Queries) ToRawQuery() url.Values {
	queryParameters := url.Values{}
	if q.Pagination != nil {
		queryParameters.Set(query.PagingParam, fmt.Sprintf("limit=%d,page=%d", q.Pagination.Limit, q.Pagination.Offset))
	}
	if q.Watch {
		queryParameters.Set(query.ParameterWatch, "true")
	}
	if q.DryRun != "" {
		queryParameters.Set(query.ParamDryRun, q.DryRun)
	}
	if q.TimeoutSeconds != nil {
		queryParameters.Set(query.ParameterTimeoutSeconds, fmt.Sprintf("%d", *q.TimeoutSeconds))
	}
	if q.LabelSelector != "" {
		queryParameters.Set(query.ParameterLabelSelector, q.LabelSelector)
	}
	if q.FieldSelector != "" {
		queryParameters.Set(query.ParameterFieldSelector, q.FieldSelector)
	}

	if len(q.FuzzySearch) > 0 {
		var fuzzy string
		for k, v := range q.FuzzySearch {
			fuzzy += fmt.Sprintf("%s~%s", k, v)
		}
		queryParameters.Set(query.ParameterFuzzySearch, fuzzy)
	}

	return queryParameters
}

// serverResponse is a wrapper for http API responses.
type serverResponse struct {
	body       io.ReadCloser
	header     http.Header
	statusCode int
	reqURL     *url.URL
}

// head sends a http request to the docker API using the method HEAD.
//
//nolint:unused
func (cli *Client) head(ctx context.Context, path string, query url.Values, headers map[string][]string) (serverResponse, error) {
	return cli.sendRequest(ctx, "HEAD", path, query, nil, headers)
}

// get sends a http request to the docker API using the method GET with a specific Go context.
func (cli *Client) get(ctx context.Context, path string, query url.Values, headers map[string][]string) (serverResponse, error) {
	return cli.sendRequest(ctx, "GET", path, query, nil, headers)
}

func (cli *Client) patch(ctx context.Context, path string, query url.Values, obj interface{}, headers map[string][]string) (serverResponse, error) {
	body, headers, err := encodeBody(obj, headers)
	if err != nil {
		return serverResponse{}, err
	}
	return cli.sendRequest(ctx, "PATCH", path, query, body, headers)
}

// post sends a http request to the docker API using the method POST with a specific Go context.
func (cli *Client) post(ctx context.Context, path string, query url.Values, obj interface{}, headers map[string][]string) (serverResponse, error) {
	body, headers, err := encodeBody(obj, headers)
	if err != nil {
		return serverResponse{}, err
	}
	return cli.sendRequest(ctx, "POST", path, query, body, headers)
}

//nolint:unused
func (cli *Client) postRaw(ctx context.Context, path string, query url.Values, body io.Reader, headers map[string][]string) (serverResponse, error) {
	return cli.sendRequest(ctx, "POST", path, query, body, headers)
}

// put sends a http request to the docker API using the method PUT.
func (cli *Client) put(ctx context.Context, path string, query url.Values, obj interface{}, headers map[string][]string) (serverResponse, error) {
	body, headers, err := encodeBody(obj, headers)
	if err != nil {
		return serverResponse{}, err
	}
	return cli.sendRequest(ctx, "PUT", path, query, body, headers)
}

// putRaw sends a http request to the docker API using the method PUT.
//
//nolint:unused
func (cli *Client) putRaw(ctx context.Context, path string, query url.Values, body io.Reader, headers map[string][]string) (serverResponse, error) {
	return cli.sendRequest(ctx, "PUT", path, query, body, headers)
}

// delete sends a http request to the docker API using the method DELETE.
func (cli *Client) delete(ctx context.Context, path string, query url.Values, headers map[string][]string) (serverResponse, error) {
	return cli.sendRequest(ctx, "DELETE", path, query, nil, headers)
}

func (cli *Client) sendRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, headers headers) (serverResponse, error) {
	req, err := cli.buildRequest(method, cli.getAPIPath(ctx, path, query), body, headers)
	if err != nil {
		return serverResponse{}, err
	}
	resp, err := cli.doRequest(ctx, req)
	if err != nil {
		return resp, err
	}
	err = cli.checkResponseErr(resp)
	return resp, err
}

func (cli *Client) checkResponseErr(serverResp serverResponse) error {
	if serverResp.statusCode >= 200 && serverResp.statusCode < 400 {
		return nil
	}
	var body []byte
	var err error
	if serverResp.body != nil {
		bodyMax := 2 * 1024 * 1024 // 2 MiB
		bodyR := &io.LimitedReader{
			R: serverResp.body,
			N: int64(bodyMax),
		}
		body, err = io.ReadAll(bodyR)
		if err != nil {
			return err
		}
		if bodyR.N == 0 {
			return fmt.Errorf("request returned %s with a message (> %d bytes) for API route and version %s, check if the server supports the requested API version", http.StatusText(serverResp.statusCode), bodyMax, serverResp.reqURL)
		}
	}
	if len(body) == 0 {
		return fmt.Errorf("request returned %s for API route and version %s, check if the server supports the requested API version", http.StatusText(serverResp.statusCode), serverResp.reqURL)
	}
	httpError := apierror.HTTPError{}
	if err := json.Unmarshal(body, &httpError); err != nil {
		return err
	}
	return &apierror.StatusError{
		Message: httpError.Message,
		Reason:  apierror.StatusReason(httpError.Reason),
		Details: nil,
		Code:    int32(httpError.Code),
	}
}

func (cli *Client) buildRequest(method, path string, body io.Reader, h headers) (*http.Request, error) {
	expectedPayload := method == "POST" || method == "PUT"
	if expectedPayload && body == nil {
		body = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	req = cli.addHeaders(req, h)

	req.URL.Host = cli.host
	req.URL.Scheme = cli.scheme

	if expectedPayload && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "text/plain")
	}
	if cli.bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cli.bearerToken))
	}
	return req, nil
}

func (cli *Client) addHeaders(req *http.Request, headers headers) *http.Request {
	for k, v := range headers {
		req.Header[k] = v
	}
	return req
}

func (cli *Client) doRequest(ctx context.Context, req *http.Request) (serverResponse, error) {
	serverResp := serverResponse{statusCode: -1, reqURL: req.URL}

	req = req.WithContext(ctx)
	resp, err := cli.HTTPClient().Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return serverResp, err
		}
		return serverResp, errors.Wrap(err, "error during connect")
	}

	if resp != nil {
		serverResp.statusCode = resp.StatusCode
		serverResp.body = resp.Body
		serverResp.header = resp.Header
	}
	return serverResp, nil
}

func ensureReaderClosed(response serverResponse) {
	if response.body != nil {
		// Drain up to 512 bytes and close the body to let the Transport reuse the connection
		_, _ = io.CopyN(io.Discard, response.body, 512)
		_ = response.body.Close()
	}
}

func encodeBody(obj interface{}, headers headers) (io.Reader, headers, error) {
	if obj == nil {
		return nil, headers, nil
	}

	body, err := encodeData(obj)
	if err != nil {
		return nil, headers, err
	}
	if headers == nil {
		headers = make(map[string][]string)
	}
	headers["Content-Type"] = []string{"application/json"}
	return body, headers, nil
}

func encodeData(data interface{}) (*bytes.Buffer, error) {
	params := bytes.NewBuffer(nil)
	if data != nil {
		if err := json.NewEncoder(params).Encode(data); err != nil {
			return nil, err
		}
	}
	return params, nil
}
