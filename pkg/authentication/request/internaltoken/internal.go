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

package internaltoken

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"

	"k8s.io/apiserver/pkg/authentication/user"

	"k8s.io/apiserver/pkg/authentication/authenticator"
)

type Authenticator struct {
	username string
	token    string
}

func New(username, token string) *Authenticator {
	return &Authenticator{username: username, token: token}
}

var ErrInvalidToken = errors.New("invalid internal token")

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	username := strings.TrimSpace(req.Header.Get(clientrest.KcUserHeader))
	token := strings.TrimSpace(req.Header.Get(clientrest.KcTokenHeader))
	if username == "" || token == "" {
		return nil, false, nil
	}

	if username == a.username && token == a.token {
		return &authenticator.Response{
			User: &user.DefaultInfo{
				Name:   a.username,
				Groups: []string{user.AllAuthenticated},
			},
		}, true, nil
	}
	return nil, false, ErrInvalidToken
}
