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
	"net/http"
	"reflect"
	"testing"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

func TestAuthenticator_AuthenticateRequest(t *testing.T) {
	type fields struct {
		username string
		token    string
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *authenticator.Response
		want1   bool
		wantErr bool
	}{
		{
			fields: fields{
				username: "testName",
				token:    "testToken",
			},
			args: args{
				req: &http.Request{
					Header: http.Header{
						"Kc-User":  []string{""},
						"Kc-Token": []string{""},
					},
				},
			},
			want:    nil,
			want1:   false,
			wantErr: false,
		},
		{
			fields: fields{
				username: "testName",
				token:    "testToken",
			},
			args: args{
				req: &http.Request{
					Header: http.Header{
						"Kc-User":  []string{"testName"},
						"Kc-Token": []string{"testToken"},
					},
				},
			},
			want: &authenticator.Response{
				User: &user.DefaultInfo{
					Name:   "testName",
					Groups: []string{user.AllAuthenticated},
				},
			},
			want1:   true,
			wantErr: false,
		},
		{
			fields: fields{
				username: "testName",
				token:    "testToken",
			},
			args: args{
				req: &http.Request{
					Header: http.Header{
						"Kc-User":  []string{"wrongName"},
						"Kc-Token": []string{"wrongToken"},
					},
				},
			},
			want:    nil,
			want1:   false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Authenticator{
				username: tt.fields.username,
				token:    tt.fields.token,
			}
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
