/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package clientrest

import (
	"net/url"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestDefaultServerURL(t *testing.T) {
	type args struct {
		host         string
		apiPath      string
		groupVersion schema.GroupVersion
		defaultTLS   bool
	}
	tests := []struct {
		name    string
		args    args
		want    *url.URL
		want1   string
		wantErr bool
	}{
		{
			name: "host is empty",
			args: args{
				host: "",
			},
			want:    nil,
			want1:   "",
			wantErr: true,
		},
		{
			name: "host is not empty and is a http service",
			args: args{
				host:    "localhost:8080",
				apiPath: "login",
				groupVersion: schema.GroupVersion{
					Group:   "group1",
					Version: "v1",
				},
				defaultTLS: false,
			},
			want: &url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
			},
			want1: "/login/group1/v1",
		},
		{
			name: "host is not empty and is a https service",
			args: args{
				host:    "localhost:8080",
				apiPath: "login",
				groupVersion: schema.GroupVersion{
					Group:   "group1",
					Version: "v1",
				},
				defaultTLS: true,
			},
			want: &url.URL{
				Scheme: "https",
				Host:   "localhost:8080",
			},
			want1: "/login/group1/v1",
		},
		{
			name: "host is incorrect",
			args: args{
				host:    "localhost:8080/login",
				apiPath: "root",
				groupVersion: schema.GroupVersion{
					Group:   "group1",
					Version: "v1",
				},
				defaultTLS: false,
			},
			want:    nil,
			want1:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := DefaultServerURL(tt.args.host, tt.args.apiPath, tt.args.groupVersion, tt.args.defaultTLS)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultServerURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultServerURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("DefaultServerURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestDefaultVersionedAPIPath(t *testing.T) {
	type args struct {
		apiPath      string
		groupVersion schema.GroupVersion
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "both of group and version are empty",
			args: args{
				apiPath: "login",
				groupVersion: schema.GroupVersion{
					Group:   "",
					Version: "",
				},
			},
			want: "/login",
		},
		{
			name: "group is empty but version is not",
			args: args{
				apiPath: "login",
				groupVersion: schema.GroupVersion{
					Group:   "",
					Version: "v1",
				},
			},
			want: "/login/v1",
		},
		{
			name: "group and empty are not empty",
			args: args{
				apiPath: "login",
				groupVersion: schema.GroupVersion{
					Group:   "group1",
					Version: "v1",
				},
			},
			want: "/login/group1/v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultVersionedAPIPath(tt.args.apiPath, tt.args.groupVersion); got != tt.want {
				t.Errorf("DefaultVersionedAPIPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
