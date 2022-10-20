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

package deploy

import (
	"reflect"
	"testing"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

func TestBuildAgentRegionPrinter(t *testing.T) {
	// agents:=[]string{"192.168.1.1"}
	// agents := []string{"192.168.1.1,192.168.1.2"}
	// agents := []string{"192.168.1.1,192.168.1.2-"} // invalid
	// agents := []string{"192.168.1.1,192.168.1.2,192.168.1.3-192.168.1.10"}
	// agents := []string{"192.168.1.1-192.168.1.10"}
	// agents := []string{"us-west-1:192.168.1.1-192.168.1.10"}
	// agents := []string{"us-west-1:192.168.1.1,192.168.1.2"}
	// agents := []string{"us-west-1:192.168.10.210,192.168.10.75,192.168.10.206"}
	// agents := []string{"us-west-1:192.168.1.1,192.168.1.2,192.168.1.3-192.168.1.10"}
	// agents := []string{"us-west-1:192.168.1.1,192.168.1.2,192.168.1.3-192.168.1.10", "us-west-2:192.168.1.11,192.168.1.12,192.168.1.13-192.168.1.20"}
	agents := []string{"1.1.1.1-1.1.1.10", "us-west-1:192.168.1.1,192.168.1.2,192.168.1.3-192.168.1.10", "us-west-2:192.168.1.11,192.168.1.12,192.168.1.13-192.168.1.20"}
	m, err := BuildAgentRegion(agents, "default")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ips:%#v", m.ListIP())
	t.Logf("ips:%#v", m)
	// t.Logf("region:%#v", set.List())
}

func TestBuildAgentRegion(t *testing.T) {
	type args struct {
		agentRegions  []string
		defaultRegion string
	}
	tests := []struct {
		name    string
		args    args
		want    options.AgentRegions
		wantErr bool
	}{
		{"1", args{agentRegions: []string{"192.168.1.1"}, defaultRegion: "default"},
			map[string][]string{"default": {"192.168.1.1"}}, false},
		{"2", args{agentRegions: []string{"192.168.1.1,192.168.1.2"}, defaultRegion: "default"},
			map[string][]string{"default": {"192.168.1.1", "192.168.1.2"}}, false},
		{"3", args{agentRegions: []string{"192.168.1.1-192.168.1.3"}, defaultRegion: "default"},
			map[string][]string{"default": {"192.168.1.1", "192.168.1.2", "192.168.1.3"}}, false},
		{"4", args{agentRegions: []string{"us-west-1:192.168.1.1-192.168.1.3"}, defaultRegion: "default"},
			map[string][]string{"us-west-1": {"192.168.1.1", "192.168.1.2", "192.168.1.3"}}, false},
		{"5", args{agentRegions: []string{"192.168.1.1,192.168.1.2-"}, defaultRegion: "default"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildAgentRegion(tt.args.agentRegions, tt.args.defaultRegion)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAgentRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildAgentRegion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFIP(t *testing.T) {
	type args struct {
		fips []string
	}
	tests := []struct {
		name    string
		args    args
		want    options.FIPs
		wantErr bool
	}{
		{
			name: "1",
			args: args{fips: []string{
				"192.168.1.1:1.1.1.1",
				"192.168.1.2:2.2.2.2",
				"192.168.1.3:3.3.3.3"}},
			want: options.FIPs{
				"192.168.1.1": "1.1.1.1",
				"192.168.1.2": "2.2.2.2",
				"192.168.1.3": "3.3.3.3"},
			wantErr: false,
		},
		{
			name: "2",
			args: args{fips: []string{
				"192.168.1.3:3.3.3.333"}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFIP(tt.args.fips)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildFIP() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAgent(t *testing.T) {
	agentRegions := []string{
		"us-west-1:192.168.1.1,192.168.1.2,192.168.1.3-192.168.1.4"}
	fips := []string{
		"192.168.1.1:1.1.1.1",
		"192.168.1.2:2.2.2.2",
		"192.168.1.3:3.3.3.3",
	}
	defaultRegion := "default"
	want := options.Agents{
		"192.168.1.1": options.Metadata{Region: "us-west-1", FloatIP: "1.1.1.1"},
		"192.168.1.2": options.Metadata{Region: "us-west-1", FloatIP: "2.2.2.2"},
		"192.168.1.3": options.Metadata{Region: "us-west-1", FloatIP: "3.3.3.3"},
		"192.168.1.4": options.Metadata{Region: "us-west-1"},
	}
	agent, err := BuildAgent(agentRegions, fips, defaultRegion)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(agent, want) {
		t.Fatalf("got: %#v want: %#v", agent, want)
	}
}
