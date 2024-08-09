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

package dnscontroller

import (
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const expected = `.:53 {
	errors
	health {
		lameduck 5s
	}
	ready
	kubernetes www.example.com in-addr.arpa ip6.arpa {
	   pods insecure
	   fallthrough in-addr.arpa ip6.arpa
	   ttl 30
	}
	prometheus :9153
	forward . /etc/resolv.conf
	cache 10
	loop
	reload 10s
	loadbalance
}
`

func Test_isGenericRecord(t *testing.T) {
	type args struct {
		rr string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				rr: "10.0.0.1",
			},
			want: false,
		},
		{
			name: "contains *.x or *",
			args: args{
				rr: "*.0.0.1",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGenericRecord(tt.args.rr); got != tt.want {
				t.Errorf("isGenericRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_renderCorefile(t *testing.T) {
	type args struct {
		data      []*v1.Domain
		vip       string
		dnsDomain string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "base",
			args: args{
				data:      []*v1.Domain{},
				vip:       "10.0.0.1",
				dnsDomain: "www.example.com",
			},
			want: expected,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderCorefile(tt.args.data, tt.args.vip, tt.args.dnsDomain)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderCorefile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("renderCorefile() got = %v, want %v", got, tt.want)
			}
		})
	}
}
