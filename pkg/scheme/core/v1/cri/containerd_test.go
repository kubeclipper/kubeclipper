package cri

import (
	"bytes"
	"testing"

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
			name: "base",
			fields: fields{
				Base: Base{
					Version:          "1.6.4",
					Offline:          true,
					DataRootDir:      "/var/lib/containerd",
					InsecureRegistry: []string{"127.0.0.1:5000", "127.0.0.1:6000"},
					Arch:             "amd64",
				},
				LocalRegistry: "127.0.0.1:5000",
				KubeVersion:   "1.23.6",
				PauseVersion:  "127.0.0.1:5000/pause:3.2",
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
