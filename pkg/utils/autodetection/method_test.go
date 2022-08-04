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

package autodetection

import (
	"net"
	"regexp"
	"testing"
)

func Test_autoDetectCIDRFirstFound(t *testing.T) {
	ipNet, err := autoDetectCIDRFirstFound(IPv4)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*ipNet)
}

func Test_autoDetectCIDRByInterface(t *testing.T) {
	ipNet, err := autoDetectCIDRByInterface(nil, IPv4)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*ipNet)
}

func Test_autoDetectCIDRByCIDR(t *testing.T) {
	cidrStr := "0.0.0.0/0"
	var matches []net.IPNet
	for _, r := range regexp.MustCompile(`\s*,\s*`).Split(cidrStr, -1) {
		_, cidr, err := parseCIDR(r)
		if err != nil {
			t.Fatal(err)
		}
		matches = append(matches, *cidr)
	}
	ipNet, err := autoDetectCIDRByCIDR(matches, IPv4)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(*ipNet)
}

func TestCheckMethod(t *testing.T) {
	type args struct {
		method string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "1", args: args{method: ""}, want: true},
		{name: "2", args: args{method: "first-found"}, want: true},
		{name: "3", args: args{method: "interface=eth1"}, want: true},
		{name: "4", args: args{method: "cidr=192.168.10.0/24"}, want: true},
		{name: "5", args: args{method: "cidrrr=192.168.10.0/24"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckMethod(tt.args.method); got != tt.want {
				t.Errorf("CheckMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}
