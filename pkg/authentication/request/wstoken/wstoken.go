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

package wstoken

import (
	"errors"
	"net/http"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

type Authenticator struct {
	auth authenticator.Token
}

func New(auth authenticator.Token) *Authenticator {
	return &Authenticator{auth}
}

var ErrInvalidToken = errors.New("invalid bearer token")

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	token := getQueryToken(req)
	if token == "" {
		return nil, false, nil
	}
	resp, ok, err := a.auth.AuthenticateToken(req.Context(), token)
	// If the token authenticator didn't error, provide a default error
	if !ok && err == nil {
		err = ErrInvalidToken
	}
	return resp, ok, err
}

// Notice: workaround for websocket auth
func getQueryToken(req *http.Request) string {
	query := req.URL.Query()
	return query.Get("token")
}
