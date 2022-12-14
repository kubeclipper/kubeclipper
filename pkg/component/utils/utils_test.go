package utils

import (
	"context"
	"testing"
)

func TestLoadImage(t *testing.T) {

	type args struct {
		ctx     context.Context
		dryRun  bool
		file    string
		criType string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "load a docker image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "docker",
			},
		},
		{
			name: "load a containerd image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "containerd",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadImage(tt.args.ctx, tt.args.dryRun, tt.args.file, tt.args.criType); err != nil {
				t.Errorf("LoadImage() error = %v", err)
			}
		})
	}
}
