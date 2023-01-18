package oplog

import "testing"

func TestOperationLog_GetRootDir(t *testing.T) {
	type fields struct {
		cfg    *Options
		suffix string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "base",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			want: "/root/demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &OperationLog{
				cfg:    tt.fields.cfg,
				suffix: tt.fields.suffix,
			}
			if got := op.GetRootDir(); got != tt.want {
				t.Errorf("GetRootDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperationLog_CreateOperationDir(t *testing.T) {
	type fields struct {
		cfg    *Options
		suffix string
	}
	type args struct {
		opID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "op ID is invalid",
			fields: fields{
				cfg: &Options{
					Dir: "/root/thisisademo",
				},
			},
			args: args{
				opID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &OperationLog{
				cfg:    tt.fields.cfg,
				suffix: tt.fields.suffix,
			}
			if err := op.CreateOperationDir(tt.args.opID); (err != nil) != tt.wantErr {
				t.Errorf("CreateOperationDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOperationLog_GetOperationDir(t *testing.T) {
	type fields struct {
		cfg    *Options
		suffix string
	}
	type args struct {
		opID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantPath string
		wantErr  bool
	}{
		{
			name: "base",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			args: args{
				opID: "123456",
			},
			wantPath: "/root/demo/123456",
			wantErr:  false,
		},
		{
			name: "op ID is invalid",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			args: args{
				opID: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &OperationLog{
				cfg:    tt.fields.cfg,
				suffix: tt.fields.suffix,
			}
			gotPath, err := op.GetOperationDir(tt.args.opID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOperationDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath {
				t.Errorf("GetOperationDir() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestOperationLog_GetStepLogFile(t *testing.T) {
	type fields struct {
		cfg    *Options
		suffix string
	}
	type args struct {
		opID   string
		stepID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantPath string
		wantErr  bool
	}{
		{
			name: "base",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			args: args{
				opID:   "123456",
				stepID: "654321",
			},
			wantPath: "/root/demo/123456/654321.log",
			wantErr:  false,
		},
		{
			name: "op ID is empty",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			args: args{
				opID:   "",
				stepID: "12345",
			},
			wantPath: "",
			wantErr:  true,
		},
		{
			name: "step ID is empty",
			fields: fields{
				cfg: &Options{
					Dir: "/root/demo",
				},
			},
			args: args{
				opID:   "12345",
				stepID: "",
			},
			wantPath: "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &OperationLog{
				cfg:    tt.fields.cfg,
				suffix: tt.fields.suffix,
			}
			gotPath, err := op.GetStepLogFile(tt.args.opID, tt.args.stepID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStepLogFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath {
				t.Errorf("GetStepLogFile() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}
