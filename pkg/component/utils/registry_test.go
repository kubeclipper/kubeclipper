package utils

import (
	"context"
	"testing"
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
