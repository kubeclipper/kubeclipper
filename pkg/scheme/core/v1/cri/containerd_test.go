package cri

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
