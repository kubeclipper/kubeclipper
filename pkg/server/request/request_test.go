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

package request

import (
	"net/http"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	req1, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes", nil)
	req2, _ = http.NewRequest("POST", "/api/core.kubeclipper.io/v1/nodes", nil)
	req3, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes/node1", nil)
	req4, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes?watch=true", nil)
	req5, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes?watch=true&fieldSelector=metadata.name=node1", nil)
	req6, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes/node1/terminal", nil)
	req7, _ = http.NewRequest("DELETE", "/api/core.kubeclipper.io/v1/nodes/node1/plugins/plugin1", nil)
)

func TestInfoFactory_NewRequestInfo(t *testing.T) {
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *Info
		wantErr bool
	}{
		{
			name: "",
			args: args{req: req1},
			want: &Info{
				IsResourceRequest: true,
				Path:              req1.URL.Path,
				Verb:              "list",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req2},
			want: &Info{
				IsResourceRequest: true,
				Path:              req2.URL.Path,
				Verb:              "create",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req3},
			want: &Info{
				IsResourceRequest: true,
				Path:              req3.URL.Path,
				Verb:              "get",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
				Name:              "node1",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req4},
			want: &Info{
				IsResourceRequest: true,
				Path:              req4.URL.Path,
				Verb:              "watch",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req5},
			want: &Info{
				IsResourceRequest: true,
				Path:              req5.URL.Path,
				Verb:              "watch",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
				Name:              "node1",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req6},
			want: &Info{
				IsResourceRequest: true,
				Path:              req6.URL.Path,
				Verb:              "get",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
				Name:              "node1",
				Subresource:       "terminal",
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{req: req7},
			want: &Info{
				IsResourceRequest: true,
				Path:              req7.URL.Path,
				Verb:              "delete",
				APIPrefix:         "api",
				APIGroup:          "core.kubeclipper.io",
				APIVersion:        "v1",
				Resource:          "nodes",
				Name:              "node1",
				Subresource:       "plugins",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InfoFactory{
				APIPrefixes: sets.New("api"),
			}
			got, err := i.NewRequestInfo(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequestInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRequestInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_splitPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "url is not empty",
			args: args{
				path: "baidu.com/login",
			},
			want: []string{"baidu.com", "login"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPath(tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
