package errors

import (
	"reflect"
	"testing"
)

func TestStatusError_Error(t *testing.T) {
	type fields struct {
		Message string
		Reason  StatusReason
		Details *StatusDetails
		Code    int32
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "base",
			fields: fields{
				Message: "Demo failed",
				Reason:  StatusReasonUnknown,
			},
			want: "Demo failed due to reason ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &StatusError{
				Message: tt.fields.Message,
				Reason:  tt.fields.Reason,
				Details: tt.fields.Details,
				Code:    tt.fields.Code,
			}
			if got := s.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusErrorCause(t *testing.T) {
	type args struct {
		err  error
		name CauseType
	}
	tests := []struct {
		name  string
		args  args
		want  StatusCause
		want1 bool
	}{
		{
			name: "base",
			args: args{
				err: &StatusError{
					Message: "This is a demo error",
					Details: &StatusDetails{
						Causes: []StatusCause{
							{
								Type:    Marshal,
								Message: "marshal error",
							},
						},
					},
				},
				name: Marshal,
			},
			want: StatusCause{
				Message: "marshal error",
				Type:    Marshal,
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := StatusErrorCause(tt.args.err, tt.args.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StatusErrorCause() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("StatusErrorCause() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				err: &StatusError{
					Code: 409,
				},
			},
			want: true,
		},
		{
			name: "not a conflict error",
			args: args{
				err: &StatusError{
					Code: 404,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflict(tt.args.err); got != tt.want {
				t.Errorf("IsConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				err: &StatusError{
					Code: 404,
				},
			},
			want: true,
		},
		{
			name: "message contains not found",
			args: args{
				err: &StatusError{
					Reason: "This is a error: not found",
				},
			},
			want: true,
		},
		{
			name: "not a not found error",
			args: args{
				err: &StatusError{
					Code: 409,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.args.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInternalError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				err: &StatusError{
					Code: 500,
				},
			},
			want: true,
		},
		{
			name: "not a internal error",
			args: args{
				err: &StatusError{
					Code: 404,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInternalError(tt.args.err); got != tt.want {
				t.Errorf("IsInternalError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTooManyRequests(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "base",
			args: args{
				err: &StatusError{
					Code: 429,
				},
			},
			want: true,
		},
		{
			name: "not a too many request error",
			args: args{
				err: &StatusError{
					Code: 404,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTooManyRequests(tt.args.err); got != tt.want {
				t.Errorf("IsTooManyRequests() = %v, want %v", got, tt.want)
			}
		})
	}
}
