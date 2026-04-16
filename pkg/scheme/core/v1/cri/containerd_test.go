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

package cri

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestContainerdRunnable_renderTo(t *testing.T) {
	type fields struct {
		Base           Base
		LocalRegistry  string
		KubeVersion    string
		PauseVersion   string
		installSteps   []v1.Step
		uninstallSteps []v1.Step
		upgradeSteps   []v1.Step
	}
	tests := []struct {
		name    string
		fields  fields
		wantW   string
		wantErr bool
	}{
		{
			name: "set insecure registry",
			fields: fields{
				Base: Base{
					Version:     "1.6.4",
					Offline:     true,
					DataRootDir: "/var/lib/containerd",
					Registies: []v1.RegistrySpec{
						{
							Scheme: "http",
							Host:   "127.0.0.1:5000",
						},
						{
							Scheme: "http",
							Host:   "127.0.0.1:6000",
							RegistryAuth: &v1.RegistryAuth{
								Username: "myuser",
								Password: "mypassword",
							},
						},
					},
					RegistryWithAuth: []v1.RegistrySpec{
						{
							Scheme: "http",
							Host:   "127.0.0.1:6000",
							RegistryAuth: &v1.RegistryAuth{
								Username: "myuser",
								Password: "mypassword",
							},
						},
					},
				},
				LocalRegistry: "",
				KubeVersion:   "1.23.6",
				PauseVersion:  "3.2",
			},
		},
		{
			name: "use local registry",
			fields: fields{
				Base: Base{
					Version:     "1.6.4",
					Offline:     true,
					DataRootDir: "/var/lib/containerd",
					Registies: []v1.RegistrySpec{
						{
							Scheme: "http",
							Host:   "127.0.0.1:5000",
						},
					},
				},
				LocalRegistry: "127.0.0.1:5000",
				KubeVersion:   "1.23.6",
				PauseVersion:  "3.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runnable := &ContainerdRunnable{
				Base:           tt.fields.Base,
				LocalRegistry:  tt.fields.LocalRegistry,
				KubeVersion:    tt.fields.KubeVersion,
				PauseVersion:   tt.fields.PauseVersion,
				installSteps:   tt.fields.installSteps,
				uninstallSteps: tt.fields.uninstallSteps,
				upgradeSteps:   tt.fields.upgradeSteps,
			}
			w := &bytes.Buffer{}
			err := runnable.renderTo(w)
			if err != nil {
				t.Errorf("renderTo() error = %v", err)
				return
			}
			t.Log(w.String())
		})
	}
}

func TestContainerdRegistryRender(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		require.NoError(t, err)
	}
	defer os.RemoveAll(dir)

	r := ContainerdRegistry{
		Server: "docker.io",
		Hosts: []ContainerdHost{
			{
				Scheme:       "https",
				Host:         "local1.registry.com",
				Capabilities: []string{CapabilityPull, CapabilityPull},
				SkipVerify:   true,
			},
			{
				Scheme:       "https",
				Host:         "local2.registry.com",
				Capabilities: []string{CapabilityPull, CapabilityPull},
				CA:           []byte("ca data"),
			},
		},
	}
	err = r.renderConfigs(dir)
	require.NoError(t, err)

	cafile := filepath.Join(dir, "docker.io", "local2.registry.com.pem")
	ca, err := os.ReadFile(cafile)
	require.NoError(t, err)
	assert.Equal(t, "ca data", string(ca))
	exp := fmt.Sprintf(`server = "docker.io"

[host]

  [host."https://local1.registry.com"]
    capabilities = ["pull", "pull"]
    skip_verify = true

  [host."https://local2.registry.com"]
    ca = "%s"
    capabilities = ["pull", "pull"]
`, strings.ReplaceAll(cafile, `\`, `\\`))

	hostConfig, err := os.ReadFile(filepath.Join(dir, "docker.io", "hosts.toml"))
	require.NoError(t, err)
	assert.Equal(t, exp, string(hostConfig))
}

func TestMergeRegistryAuthIntoConfig(t *testing.T) {
	baseConfig := `disabled_plugins = []
root = "/var/lib/containerd"
state = "/run/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.2"

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`

	tests := []struct {
		name              string
		existingConfig    string
		registryWithAuth  []v1.RegistrySpec
		wantAuthHosts     []string
		wantAuthUsername  map[string]string
		wantAuthPassword  map[string]string
		wantPreservePath  []string
		wantPreserveValue string
	}{
		{
			name:           "add new auth preserves nvidia runtime",
			existingConfig: baseConfig,
			registryWithAuth: []v1.RegistrySpec{
				{
					Host: "registry.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "admin",
						Password: "s3cret",
					},
				},
			},
			wantAuthHosts:     []string{"registry.example.com"},
			wantAuthUsername:  map[string]string{"registry.example.com": "admin"},
			wantAuthPassword:  map[string]string{"registry.example.com": "s3cret"},
			wantPreservePath:  []string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "nvidia", "options", "BinaryName"},
			wantPreserveValue: "/usr/bin/nvidia-container-runtime",
		},
		{
			name: "update existing auth preserves other sections",
			existingConfig: `disabled_plugins = []
root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.2"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

        [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.example.com"]

          [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.example.com".auth]
            username = "olduser"
            password = "oldpass"
            auth = ""
            identitytoken = ""

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`,
			registryWithAuth: []v1.RegistrySpec{
				{
					Host: "registry.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "newuser",
						Password: "newpass",
					},
				},
			},
			wantAuthHosts:     []string{"registry.example.com"},
			wantAuthUsername:  map[string]string{"registry.example.com": "newuser"},
			wantAuthPassword:  map[string]string{"registry.example.com": "newpass"},
			wantPreservePath:  []string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"},
			wantPreserveValue: "registry.k8s.io/pause:3.2",
		},
		{
			name: "add auth when configs section is empty",
			existingConfig: `root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.2"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"
`,
			registryWithAuth: []v1.RegistrySpec{
				{
					Host: "my-registry.local",
					RegistryAuth: &v1.RegistryAuth{
						Username: "testuser",
						Password: "testpass",
					},
				},
			},
			wantAuthHosts:     []string{"my-registry.local"},
			wantAuthUsername:  map[string]string{"my-registry.local": "testuser"},
			wantAuthPassword:  map[string]string{"my-registry.local": "testpass"},
			wantPreservePath:  []string{"root"},
			wantPreserveValue: "/var/lib/containerd",
		},
		{
			name:           "multiple registry auth entries",
			existingConfig: baseConfig,
			registryWithAuth: []v1.RegistrySpec{
				{
					Host: "registry1.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "user1",
						Password: "pass1",
					},
				},
				{
					Host: "registry2.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "user2",
						Password: "pass2",
					},
				},
			},
			wantAuthHosts:     []string{"registry1.example.com", "registry2.example.com"},
			wantAuthUsername:  map[string]string{"registry1.example.com": "user1", "registry2.example.com": "user2"},
			wantAuthPassword:  map[string]string{"registry1.example.com": "pass1", "registry2.example.com": "pass2"},
			wantPreservePath:  []string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"},
			wantPreserveValue: "nvidia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.toml")
			err := os.WriteFile(configPath, []byte(tt.existingConfig), 0644)
			require.NoError(t, err)

			runnable := &ContainerdRunnable{
				Base: Base{
					RegistryWithAuth: tt.registryWithAuth,
				},
			}

			err = runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, false)
			require.NoError(t, err)

			result, err := toml.LoadFile(configPath)
			require.NoError(t, err)

			for _, host := range tt.wantAuthHosts {
				authPath := []string{"plugins", "io.containerd.grpc.v1.cri", "registry", "configs", host, "auth"}
				authTree := result.GetPath(authPath)
				require.NotNil(t, authTree, "expected auth config for %s", host)

				authToml, ok := authTree.(*toml.Tree)
				require.True(t, ok, "auth section for %s should be a TOML tree", host)

				assert.Equal(t, tt.wantAuthUsername[host], authToml.Get("username"), "username mismatch for %s", host)
				assert.Equal(t, tt.wantAuthPassword[host], authToml.Get("password"), "password mismatch for %s", host)
				assert.NotNil(t, authToml.Get("auth"), "auth field should exist for %s", host)
				assert.NotNil(t, authToml.Get("identitytoken"), "identitytoken field should exist for %s", host)
			}

			preservedValue := result.GetPath(tt.wantPreservePath)
			assert.Equal(t, tt.wantPreserveValue, preservedValue,
				"preserved value at %v should remain unchanged", tt.wantPreservePath)
		})
	}
}

func TestMergeRegistryAuthIntoConfig_NoAuthEntries(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	existingConfig := `root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.2"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

        [plugins."io.containerd.grpc.v1.cri".registry.configs."stale-registry.local"]

          [plugins."io.containerd.grpc.v1.cri".registry.configs."stale-registry.local".auth]
            username = "olduser"
            password = "oldpass"
            auth = ""
            identitytoken = ""

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`
	err := os.WriteFile(configPath, []byte(existingConfig), 0644)
	require.NoError(t, err)

	runnable := &ContainerdRunnable{
		Base: Base{
			RegistryWithAuth: []v1.RegistrySpec{},
		},
	}

	err = runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, false)
	require.NoError(t, err)

	result, err := toml.LoadFile(configPath)
	require.NoError(t, err)

	// Stale auth should be removed
	authPath := []string{"plugins", "io.containerd.grpc.v1.cri", "registry", "configs", "stale-registry.local", "auth"}
	assert.Nil(t, result.GetPath(authPath), "stale auth should be removed when no RegistryWithAuth")

	// Other config preserved
	assert.Equal(t, "registry.k8s.io/pause:3.2", result.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"}))
}

func TestMergeRegistryAuthIntoConfig_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	runnable := &ContainerdRunnable{
		Base: Base{
			Version:     "1.6.4",
			DataRootDir: "/var/lib/containerd",
			RegistryWithAuth: []v1.RegistrySpec{
				{
					Host: "registry.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "admin",
						Password: "s3cret",
					},
				},
			},
		},
		KubeVersion:         "1.23.6",
		PauseVersion:        "3.2",
		RegistryConfigDir:   "/etc/containerd/certs.d",
		EnableSystemdCgroup: "true",
	}

	err := runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, false)
	require.NoError(t, err)

	_, err = os.Stat(configPath)
	assert.NoError(t, err, "config file should be created when it doesn't exist")
}

func TestMergeRegistryAuthIntoConfig_RemoveStaleAuth(t *testing.T) {
	existingConfig := `root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.2"

    [plugins."io.containerd.grpc.v1.cri".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]

        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.grpc.v1.cri".registry.configs]

        [plugins."io.containerd.grpc.v1.cri".registry.configs."old-registry.local"]

          [plugins."io.containerd.grpc.v1.cri".registry.configs."old-registry.local".auth]
            username = "olduser"
            password = "oldpass"
            auth = ""
            identitytoken = ""

        [plugins."io.containerd.grpc.v1.cri".registry.configs."new-registry.local"]

      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	err := os.WriteFile(configPath, []byte(existingConfig), 0644)
	require.NoError(t, err)

	runnable := &ContainerdRunnable{
		Base: Base{
			RegistryWithAuth: []v1.RegistrySpec{
				{
					Host: "new-registry.local",
					RegistryAuth: &v1.RegistryAuth{
						Username: "newuser",
						Password: "newpass",
					},
				},
			},
		},
	}

	err = runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, false)
	require.NoError(t, err)

	result, err := toml.LoadFile(configPath)
	require.NoError(t, err)

	// Old auth should be removed
	oldAuthPath := []string{"plugins", "io.containerd.grpc.v1.cri", "registry", "configs", "old-registry.local", "auth"}
	assert.Nil(t, result.GetPath(oldAuthPath), "stale auth for old-registry.local should be removed")

	// New auth should be added
	newAuthPath := []string{"plugins", "io.containerd.grpc.v1.cri", "registry", "configs", "new-registry.local", "auth"}
	newAuthTree := result.GetPath(newAuthPath)
	require.NotNil(t, newAuthTree, "auth for new-registry.local should be added")
	newAuthToml, ok := newAuthTree.(*toml.Tree)
	require.True(t, ok)
	assert.Equal(t, "newuser", newAuthToml.Get("username"))
	assert.Equal(t, "newpass", newAuthToml.Get("password"))

	// nvidia runtime preserved (sandbox_image as proxy check, nvidia options.BinaryName not in this test config)
	assert.Equal(t, "registry.k8s.io/pause:3.2", result.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"}))
	assert.Equal(t, "nvidia", result.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}))
}

func TestMergeRegistryAuthIntoConfig_DryRun(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	originalContent := "root = \"/var/lib/containerd\"\n"
	err := os.WriteFile(configPath, []byte(originalContent), 0644)
	require.NoError(t, err)

	runnable := &ContainerdRunnable{
		Base: Base{
			RegistryWithAuth: []v1.RegistrySpec{
				{
					Host: "registry.example.com",
					RegistryAuth: &v1.RegistryAuth{
						Username: "admin",
						Password: "s3cret",
					},
				},
			},
		},
	}

	err = runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, true)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(data))
}
