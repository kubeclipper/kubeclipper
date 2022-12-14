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

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchLinuxFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{
			path:     "/",
			expected: true,
		},
		{
			path:     "/dev/null",
			expected: true,
		},
		{
			path:     "etc",
			expected: false,
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, MatchLinuxFilePath(tt.path))
	}
}

func TestMatchKubernetesNamespace(t *testing.T) {
	type args struct {
		namespace string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test with a correct namespace",
			args: args{
				namespace: "kube-system",
			},
			want: true,
		},
		{
			name: "test with an incorrect namespace",
			args: args{
				namespace: "-kube-system",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, MatchKubernetesNamespace(tt.args.namespace), "MatchKubernetesNamespace(%v)", tt.args.namespace)
		})
	}
}

func TestMatchKubernetesStorageClass(t *testing.T) {
	type args struct {
		storageClass string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test with a correct storageClass",
			args: args{
				storageClass: "sc1",
			},
			want: true,
		},
		{
			name: "test with an incorrect storageClass",
			args: args{
				storageClass: "-storageClass-1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, MatchKubernetesStorageClass(tt.args.storageClass), "MatchKubernetesStorageClass(%v)", tt.args.storageClass)
		})
	}
}

func TestIsHostNameRFC952(t *testing.T) {
	type args struct {
		hostname string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test with a correct hostname",
			args: args{
				hostname: "localhost",
			},
			want: true,
		},
		{
			name: "test with an incorrect hostname",
			args: args{
				hostname: "-localhost",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsHostNameRFC952(tt.args.hostname), "IsHostNameRFC952(%v)", tt.args.hostname)
		})
	}
}

func TestIsURL(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test with a correct url",
			args: args{
				raw: "https://localhost:8080",
			},
			want: true,
		},
		{
			name: "test with an incorrect url",
			args: args{
				raw: "-https://localhost:8080",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsURL(tt.args.raw), "IsURL(%v)", tt.args.raw)
		})
	}
}

func TestMatchKubernetesReclaimPolicy(t *testing.T) {
	type args struct {
		policy string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test a Retain policy",
			args: args{
				policy: "Retain",
			},
		},
		{
			name: "test a Delete policy",
			args: args{
				policy: "Delete",
			},
		},
		{
			name: "test a wrong policy",
			args: args{
				policy: "Delete1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MatchKubernetesReclaimPolicy(tt.args.policy); (err != nil) && !tt.wantErr {
				t.Errorf("MatchKubernetesReclaimPolicy() error %v", err)
			}
		})
	}
}
