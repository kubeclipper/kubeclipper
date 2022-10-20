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

package resource

import (
	"testing"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

func TestResourceOptions_parsePackageName(t *testing.T) {
	type fields struct {
		IOStreams    options.IOStreams
		PrintFlags   *printer.PrintFlags
		SSHConfig    *sshutils.SSH
		DeployConfig string
		deployConfig *options.DeployConfig
		List         string
		Push         string
		Delete       string
		Type         string
		Name         string
		Version      string
		Arch         string
		Pkg          string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		want1   string
		want2   string
		wantErr bool
	}{
		{
			name: "base",
			fields: fields{
				Pkg: "k8s-v1.18.6-x86_64.tar.gz",
			},
			want:  "k8s",
			want1: "v1.18.6",
			want2: "x86_64",
		},
		{
			name: "name-test",
			fields: fields{
				Pkg: "/root/k8s-v1.18.6-x86_64.tar.gz",
			},
			want:  "k8s",
			want1: "v1.18.6",
			want2: "x86_64",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ResourceOptions{
				IOStreams:    tt.fields.IOStreams,
				PrintFlags:   tt.fields.PrintFlags,
				deployConfig: tt.fields.deployConfig,
				List:         tt.fields.List,
				Push:         tt.fields.Push,
				Delete:       tt.fields.Delete,
				Type:         tt.fields.Type,
				Name:         tt.fields.Name,
				Version:      tt.fields.Version,
				Arch:         tt.fields.Arch,
				Pkg:          tt.fields.Pkg,
			}
			got, got1, got2, err := o.parsePackageName()
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePackageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePackageName() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parsePackageName() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("parsePackageName() got2 = %v, want %v", got2, tt.want2)
			}
		})
	}
}
