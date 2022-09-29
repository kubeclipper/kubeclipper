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

package httputil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type RespError struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
	Detail  interface{} `json:"detail,omitempty"`
}

func CommonRequest(requestURL, httpMethod string, header, rawQuery map[string]string, postBody json.RawMessage) ([]byte, int, error) {
	var req *http.Request
	var reqErr error

	req, reqErr = http.NewRequest(httpMethod, requestURL, bytes.NewReader(postBody))
	if reqErr != nil {
		return []byte{}, http.StatusInternalServerError, reqErr
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	for key, val := range header {
		req.Header.Add(key, val)
	}
	if rawQuery != nil {
		p := make(url.Values)
		for key, val := range rawQuery {
			p.Add(key, val)
		}
		req.URL.RawQuery = p.Encode()
	}

	ts := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: ts,
	}
	client.Timeout = 5 * time.Second
	resp, respErr := client.Do(req)
	if respErr != nil {
		return []byte{}, http.StatusInternalServerError, respErr
	}
	defer resp.Body.Close()
	body, readBodyErr := io.ReadAll(resp.Body)
	if readBodyErr != nil {
		return []byte{}, http.StatusInternalServerError, readBodyErr
	}
	return body, resp.StatusCode, nil
}

func CodeDispose(body []byte, code int) ([]byte, error) {
	switch code {
	case 200:
		return body, nil
	case 202:
		return body, nil
	default:
		errs := make(map[string][]RespError)
		if marshalErr := json.Unmarshal(body, &errs); marshalErr != nil {
			return []byte{}, marshalErr
		}
		var res string
		for _, err := range errs {
			for _, e := range err {
				res = e.Message
			}
		}
		return []byte{}, fmt.Errorf("there is an error in the input: %s", res)
	}
}

func IsURL(u string) (url.URL, bool) {
	if uu, err := url.Parse(u); err == nil && uu != nil && uu.Host != "" {
		return *uu, true
	}
	return url.URL{}, false
}
