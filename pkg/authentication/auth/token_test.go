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

package auth

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/oauth"
	authoptions "github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/authentication/token"
	iammock "github.com/kubeclipper/kubeclipper/pkg/models/iam/mock"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

func newTokenOpera(iamMockOpera *iammock.MockOperator) *tokenOperator {
	authOpt := &authoptions.AuthenticationOptions{
		MaximumClockSkew: 10 * time.Hour,
		JwtSecret:        "D9ykGOmuE3yXe35Wh3mRniGT",
		OAuthOptions: &oauth.Options{
			AccessTokenMaxAge: 10 * time.Hour,
		},
	}
	tokenOpera := &tokenOperator{
		issuer:     token.NewTokenIssuer(authOpt.JwtSecret, authOpt.MaximumClockSkew),
		options:    authOpt,
		tokenCache: iamMockOpera,
	}
	return tokenOpera
}

func issueToTestPrepare(iamMockOpera *iammock.MockOperator) {
	iamMockOpera.EXPECT().DeleteTokenCollection(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	iamMockOpera.EXPECT().CreateToken(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
}

func verifyTestPrepare(iamMockOpera *iammock.MockOperator) {
	iamMockOpera.EXPECT().ListTokens(gomock.Any(), gomock.Any()).Return(
		&v1.TokenList{
			Items: make([]v1.Token, 1),
		}, nil).AnyTimes()
}

func authenticateTokenTestPrepare(iamMockOpera *iammock.MockOperator) {
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
	return tokenStr
}

func Test_tokenOperator_IssueTo(t1 *testing.T) {
	ctrl := gomock.NewController(t1)
	defer ctrl.Finish()

	iamMockOpera := iammock.NewMockOperator(ctrl)
	tokenOpera := newTokenOpera(iamMockOpera)
	issueToTestPrepare(iamMockOpera)

	type args struct {
		user user.Info
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "IssueTo test",
			args: args{
				user: &user.DefaultInfo{
					Name:   "testName",
					UID:    "testUID",
					Groups: nil,
					Extra:  nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			_, err := tokenOpera.IssueTo(tt.args.user)
			if (err != nil) != tt.wantErr {
				t1.Errorf("IssueTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_tokenOperator_Verify(t1 *testing.T) {
	ctrl := gomock.NewController(t1)
	defer ctrl.Finish()

	iamMockOpera := iammock.NewMockOperator(ctrl)
	tokenOpera := newTokenOpera(iamMockOpera)
	verifyTestPrepare(iamMockOpera)

	type args struct {
		tokenStr string
	}
	tests := []struct {
		name    string
		args    args
		want    user.Info
		wantErr bool
	}{
		{
			name: "auth_Verify test",
			args: args{
				tokenStr: getToken(),
			},
			want: &user.DefaultInfo{
				Name:   "admin",
				UID:    "",
				Groups: nil,
				Extra:  nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			got, err := tokenOpera.Verify(tt.args.tokenStr)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Verify() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tokenAuthenticator_AuthenticateToken(t1 *testing.T) {
	ctrl := gomock.NewController(t1)
	defer ctrl.Finish()

	iamMockOpera := iammock.NewMockOperator(ctrl)
	tokenOpera := newTokenOpera(iamMockOpera)
	tokenAuth := &tokenAuthenticator{
		userOperator:  iamMockOpera,
		tokenOperator: tokenOpera,
	}
	authenticateTokenTestPrepare(iamMockOpera)

	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    *authenticator.Response
		want1   bool
		wantErr bool
	}{
		{
			name: "AuthenticateToken test",
			args: args{
				ctx:   context.TODO(),
				token: getToken(),
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
		t1.Run(tt.name, func(t1 *testing.T) {
			got, got1, err := tokenAuth.AuthenticateToken(tt.args.ctx, tt.args.token)
			fmt.Println("got: ", got)
			if (err != nil) != tt.wantErr {
				t1.Errorf(" 111 AuthenticateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf(" 222 AuthenticateToken() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t1.Errorf(" 333 AuthenticateToken() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
