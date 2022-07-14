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
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/token"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	iammock "github.com/kubeclipper/kubeclipper/pkg/models/iam/mock"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

func authenticateRequestPrepare(iamMockOpera *iammock.MockOperator) {
	iamMockOpera.EXPECT().ListTokens(gomock.Any(), gomock.Any()).Return(
		&v1.TokenList{
			Items: make([]v1.Token, 1),
		}, nil).AnyTimes()
	iamMockOpera.EXPECT().GetUserEx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			},
			Spec: v1.UserSpec{
				Groups: make([]string, 0),
			},
		}, nil).AnyTimes()
}

func getToken() string {
	s := token.NewTokenIssuer("D9ykGOmuE3yXe35Wh3mRniGT", 0)
	u := &user.DefaultInfo{
		Name: "admin",
	}
	tokenStr, err := s.IssueTo(u, "bearer", 10*time.Hour)
	if err != nil {
		return ""
	}
	return "token=" + tokenStr
}

func TestAuthenticator_AuthenticateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	iamMockOpera := iammock.NewMockOperator(ctrl)
	authenticateRequestPrepare(iamMockOpera)
	tokenOpera := auth.NewTokenOperator(iamMockOpera, &options.AuthenticationOptions{
		MaximumClockSkew: 0,
		JwtSecret:        "D9ykGOmuE3yXe35Wh3mRniGT",
		OAuthOptions: &oauth.Options{
			AccessTokenMaxAge: 10 * time.Hour,
		},
	})

	a := &Authenticator{
		auth: auth.NewTokenAuthenticator(iamMockOpera, tokenOpera),
	}

	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *authenticator.Response
		want1   bool
		wantErr bool
	}{
		{
			name: "AuthenticateRequest test",
			args: args{
				req: &http.Request{
					URL: &url.URL{
						RawQuery: getToken(),
					},
				},
			},
			want: &authenticator.Response{
				User: &user.DefaultInfo{
					Name:   "admin",
					Groups: []string{"system:authenticated"},
				},
			},
			want1:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := a.AuthenticateRequest(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthenticateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthenticateRequest() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("AuthenticateRequest() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
