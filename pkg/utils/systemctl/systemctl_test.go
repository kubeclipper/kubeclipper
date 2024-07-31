package systemctl

import (
	"context"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
)

func Test_lookupUnitStatus(t *testing.T) {
	ctx := context.Background()
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		t.Errorf("failed to connect dbus: %v", err)
		return
	}
	defer conn.Close()

	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus bool
		wantErr    bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "systemd-journald.service",
			},
			wantStatus: true,
			wantErr:    false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "foo",
			},
			wantStatus: false,
			wantErr:    false,
		},
		{
			name: "case3",
			args: args{
				ctx:  context.Background(),
				unit: "systemd-journald",
			},
			wantStatus: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := lookupUnitStatus(ctx, conn, tt.args.unit)
			if (err != nil) != tt.wantErr {
				t.Errorf("lookupUnitStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (status != nil) != tt.wantStatus {
				t.Errorf("lookupUnitStatus() = %v, hasStatus %v", status, tt.wantStatus)
			}
		})
	}
}

func TestStartUnit(t *testing.T) {
	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "systemd-journald.service",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "systemd-journald",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StartUnit(tt.args.ctx, tt.args.unit); (err != nil) != tt.wantErr {
				t.Errorf("StartUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStopUnit(t *testing.T) {
	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "chronyd.service",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "systemd-journald",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StopUnit(tt.args.ctx, tt.args.unit); (err != nil) != tt.wantErr {
				t.Errorf("StopUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnableUnit(t *testing.T) {
	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "chronyd.service",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := EnableUnit(tt.args.ctx, tt.args.unit); (err != nil) != tt.wantErr {
				t.Errorf("EnableUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDisableUnit(t *testing.T) {
	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "chronyd.service",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "foo",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DisableUnit(tt.args.ctx, tt.args.unit); (err != nil) != tt.wantErr {
				t.Errorf("DisableUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestartUnit(t *testing.T) {
	type args struct {
		ctx  context.Context
		unit string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				ctx:  context.Background(),
				unit: "chronyd.service",
			},
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				ctx:  context.Background(),
				unit: "foo",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RestartUnit(tt.args.ctx, tt.args.unit); (err != nil) != tt.wantErr {
				t.Errorf("RestartUnit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
