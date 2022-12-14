package clientrest

import (
	"net/http"
	"testing"
)

func TestIsInformerRawQuery(t *testing.T) {
	request1, _ := http.NewRequest("GET", "http://localhost", nil)
	request2, _ := http.NewRequest("GET", "http://localhost", nil)
	request1.Header.Set(QueryTypeHeader, InformerQuery)
	request2.Header.Set("Connection", "keep-alive")
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "is informer raw query",
			args: args{
				request1,
			},
			want: true,
		},
		{
			name: "not informer raw query",
			args: args{
				request2,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInformerRawQuery(tt.args.req); got != tt.want {
				t.Errorf("IsInformerRawQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
