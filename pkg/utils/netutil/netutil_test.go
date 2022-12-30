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

package netutil

import (
	"strings"
	"testing"
	"time"
)

const (
	NIP  = 3232235777
	NIP2 = 3232235786
	AIP  = "192.168.1.1"
	AIP2 = "192.168.1.10"
)

func TestInetNtoA(t *testing.T) {
	type args struct {
		ip int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"1", args{ip: NIP}, AIP},
		{"2", args{ip: NIP2}, AIP2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InetNtoA(tt.args.ip); got != tt.want {
				t.Errorf("InetNtoA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInetAtoN(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{"1", args{ip: AIP}, NIP},
		{"2", args{ip: AIP2}, NIP2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InetAtoN(tt.args.ip); got != tt.want {
				t.Errorf("InetAtoN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReachable(t *testing.T) {
	type args struct {
		protocol string
		addr     string
		timeout  time.Duration
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "base",
			args: args{
				protocol: "tcp",
				addr:     "127.0.0.1:6443",
				timeout:  time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Reachable(tt.args.protocol, tt.args.addr, tt.args.timeout)
			if err != nil && !strings.Contains(err.Error(), "connect: connection refused") {
				t.Errorf("test result: %v", err)
			}
		})
	}
}
