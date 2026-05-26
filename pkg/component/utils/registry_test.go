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

package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddOrRemoveInsecureRegistryToCRI(t *testing.T) {
	type args struct {
		ctx      context.Context
		criType  string
		registry string
		add      bool
		dryRun   bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "add an insecure registry to docker",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
		},
		{
			name: "remove an insecure registry from docker",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker",
				registry: "127.0.0.1:5000",
				add:      false,
				dryRun:   true,
			},
		},
		{
			name: "add an insecure registry to containerd",
			args: args{
				ctx:      context.TODO(),
				criType:  "containerd",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
		},
		{
			name: "remove an insecure registry from containerd",
			args: args{
				ctx:      context.TODO(),
				criType:  "containerd",
				registry: "127.0.0.1:5000",
				add:      false,
				dryRun:   true,
			},
		},
		{
			name: "test with a wrong Cri",
			args: args{
				ctx:      context.TODO(),
				criType:  "docker1",
				registry: "127.0.0.1:5000",
				add:      true,
				dryRun:   true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddOrRemoveInsecureRegistryToCRI(tt.args.ctx, tt.args.criType, tt.args.registry, tt.args.add, tt.args.dryRun); (err != nil) != tt.wantErr {
				t.Errorf("AddOrRemoveInsecureRegistryToCRI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddOrRemoveContainerdInsecureRegistry_V3Config(t *testing.T) {
	v3Config := `version = 3
root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    [plugins."io.containerd.cri.v1.images".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.cri.v1.images".registry.mirrors]

  [plugins."io.containerd.cri.v1.runtime"]
`

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(v3Config), 0644))

	origDefault := containerdDefaultConfig
	containerdDefaultConfig = configPath
	defer func() { containerdDefaultConfig = origDefault }()

	// Error from systemctl restart is expected in test environment; file is written before that.
	_ = addOrRemoveContainerdInsecureRegistry(context.Background(), "myregistry.io:5000", true, false)

	result, err := toml.LoadFile(configPath)
	require.NoError(t, err)

	endpoint := result.GetPath([]string{"plugins", "io.containerd.cri.v1.images", "registry", "mirrors", "myregistry.io:5000", "endpoint"})
	require.NotNil(t, endpoint, "insecure registry should be added under cri.v1.images path for v3 config")
	endpointArr, ok := endpoint.([]interface{})
	require.True(t, ok)
	assert.Equal(t, "http://myregistry.io:5000", endpointArr[0])
}

func TestAddOrRemoveContainerdInsecureRegistry_V2Config(t *testing.T) {
	v2Config := `version = 2
root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(v2Config), 0644))

	origDefault := containerdDefaultConfig
	containerdDefaultConfig = configPath
	defer func() { containerdDefaultConfig = origDefault }()

	// Error from systemctl restart is expected in test environment; file is written before that.
	_ = addOrRemoveContainerdInsecureRegistry(context.Background(), "myregistry.io:5000", true, false)

	result, err := toml.LoadFile(configPath)
	require.NoError(t, err)

	endpoint := result.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "registry", "mirrors", "myregistry.io:5000", "endpoint"})
	require.NotNil(t, endpoint, "insecure registry should be added under grpc.v1.cri path for v2 config")
}
