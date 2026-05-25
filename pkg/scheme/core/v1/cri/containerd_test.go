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

func TestIsContainerdV2(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"v1 minor", "1.7.29", false},
		{"v2 minor", "2.0.0", true},
		{"v2.2", "2.2.4", true},
		{"v3", "3.0.0", true},
		{"v0", "0.9.0", false},
		{"empty", "", false},
		{"no patch", "2.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ContainerdRunnable{Base: Base{Version: tt.version}}
			assert.Equal(t, tt.want, r.isContainerdV2())
		})
	}
}

func TestContainerdConfigVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			"v2 config",
			`version = 2
root = "/var/lib/containerd"`,
			2,
		},
		{
			"v3 config",
			`version = 3
root = "/var/lib/containerd"`,
			3,
		},
		{
			"no version field defaults to 2",
			`root = "/var/lib/containerd"`,
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "config.toml")
			require.NoError(t, os.WriteFile(p, []byte(tt.content), 0644))
			assert.Equal(t, tt.want, containerdConfigVersion(p))
		})
	}
}

func TestContainerdRunnable_renderTo_V3(t *testing.T) {
	runnable := &ContainerdRunnable{
		Base: Base{
			Version:     "2.2.4",
			Offline:     true,
			DataRootDir: "/var/lib/containerd",
			Registies: []v1.RegistrySpec{
				{
					Scheme: "http",
					Host:   "127.0.0.1:5000",
				},
			},
			RegistryWithAuth: []v1.RegistrySpec{
				{
					Host: "127.0.0.1:6000",
					RegistryAuth: &v1.RegistryAuth{
						Username: "myuser",
						Password: "mypassword",
					},
				},
			},
		},
		LocalRegistry:       "127.0.0.1:5000",
		KubeVersion:         "1.29.0",
		PauseVersion:        "3.9",
		PauseRegistry:       "registry.k8s.io",
		RegistryConfigDir:   "/etc/containerd/certs.d",
		EnableSystemdCgroup: "true",
	}

	w := &bytes.Buffer{}
	err := runnable.renderTo(w)
	require.NoError(t, err)

	result := w.String()
	conf, err := toml.Load(result)
	require.NoError(t, err)

	// version = 3
	assert.Equal(t, int64(3), conf.Get("version"))

	// CRI plugin split: cri.v1.images exists
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.cri.v1.images"}), "cri.v1.images plugin should exist")
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.cri.v1.runtime"}), "cri.v1.runtime plugin should exist")
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri"}), "grpc.v1.cri plugin should exist")

	// pinned_images.sandbox instead of sandbox_image
	sandbox := conf.GetPath([]string{"plugins", "io.containerd.cri.v1.images", "pinned_images", "sandbox"})
	assert.Equal(t, "127.0.0.1:5000/pause:3.9", sandbox)

	// sandbox_image should NOT exist at old path
	assert.Nil(t, conf.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"}))

	// registry under cri.v1.images
	assert.Equal(t, "/etc/containerd/certs.d", conf.GetPath([]string{"plugins", "io.containerd.cri.v1.images", "registry", "config_path"}))

	// auth under cri.v1.images
	authTree := conf.GetPath([]string{"plugins", "io.containerd.cri.v1.images", "registry", "configs", "127.0.0.1:6000", "auth"})
	require.NotNil(t, authTree, "auth should exist under cri.v1.images")
	authToml, ok := authTree.(*toml.Tree)
	require.True(t, ok)
	assert.Equal(t, "myuser", authToml.Get("username"))
	assert.Equal(t, "mypassword", authToml.Get("password"))

	// SystemdCgroup under cri.v1.runtime
	sysCgroup := conf.GetPath([]string{"plugins", "io.containerd.cri.v1.runtime", "containerd", "runtimes", "runc", "options", "SystemdCgroup"})
	assert.Equal(t, true, sysCgroup)

	// monitor renamed
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.monitor.task.v1.cgroups"}), "monitor.task.v1.cgroups should exist")
	assert.Nil(t, conf.GetPath([]string{"plugins", "io.containerd.monitor.v1.cgroups"}), "old monitor path should not exist")

	// deprecated fields removed
	assert.Nil(t, conf.GetPath([]string{"plugins", "io.containerd.runtime.v1.linux"}), "runtime.v1.linux should be removed")
	assert.Nil(t, conf.GetPath([]string{"plugins", "io.containerd.snapshotter.v1.aufs"}), "aufs snapshotter should be removed")
}

func TestContainerdRunnable_renderTo_V2_BackwardCompat(t *testing.T) {
	runnable := &ContainerdRunnable{
		Base: Base{
			Version:     "1.7.29",
			Offline:     true,
			DataRootDir: "/var/lib/containerd",
		},
		PauseVersion:        "3.9",
		PauseRegistry:       "registry.k8s.io",
		RegistryConfigDir:   "/etc/containerd/certs.d",
		EnableSystemdCgroup: "true",
	}

	w := &bytes.Buffer{}
	err := runnable.renderTo(w)
	require.NoError(t, err)

	conf, err := toml.Load(w.String())
	require.NoError(t, err)

	// version = 2
	assert.Equal(t, int64(2), conf.Get("version"))

	// monolithic CRI plugin
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri"}))
	// sandbox_image at old path
	assert.NotNil(t, conf.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"}))
}

func TestMergeRegistryAuthIntoConfig_V3(t *testing.T) {
	v3Config := `version = 3
root = "/var/lib/containerd"

[plugins]
  [plugins."io.containerd.cri.v1.images"]
    [plugins."io.containerd.cri.v1.images".pinned_images]
      sandbox = "registry.k8s.io/pause:3.9"

    [plugins."io.containerd.cri.v1.images".containerd]
      snapshotter = "overlayfs"

    [plugins."io.containerd.cri.v1.images".registry]
      config_path = "/etc/containerd/certs.d"

      [plugins."io.containerd.cri.v1.images".registry.configs]

      [plugins."io.containerd.cri.v1.images".registry.mirrors]

  [plugins."io.containerd.cri.v1.runtime"]
    [plugins."io.containerd.cri.v1.runtime".containerd]
      default_runtime_name = "nvidia"

      [plugins."io.containerd.cri.v1.runtime".containerd.runtimes]

        [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.cri.v1.runtime".containerd.runtimes.nvidia.options]
            BinaryName = "/usr/bin/nvidia-container-runtime"
            SystemdCgroup = true
`

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte(v3Config), 0644))

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

	err := runnable.mergeRegistryAuthIntoConfig(context.Background(), configPath, false)
	require.NoError(t, err)

	result, err := toml.LoadFile(configPath)
	require.NoError(t, err)

	// auth under cri.v1.images path
	authPath := []string{"plugins", "io.containerd.cri.v1.images", "registry", "configs", "registry.example.com", "auth"}
	authTree := result.GetPath(authPath)
	require.NotNil(t, authTree, "auth should be written to cri.v1.images path for v3 config")
	authToml, ok := authTree.(*toml.Tree)
	require.True(t, ok)
	assert.Equal(t, "admin", authToml.Get("username"))
	assert.Equal(t, "s3cret", authToml.Get("password"))

	// nvidia runtime preserved
	binName := result.GetPath([]string{"plugins", "io.containerd.cri.v1.runtime", "containerd", "runtimes", "nvidia", "options", "BinaryName"})
	assert.Equal(t, "/usr/bin/nvidia-container-runtime", binName)
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
