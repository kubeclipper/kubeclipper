package wssstream

import (
	"net/http"
	"testing"
)

func TestIsWebSocketRequest(t *testing.T) {
	goodRequest, _ := http.NewRequest("GET", "http://example.com", nil)
	goodRequest.Header.Set("Upgrade", "websocket")
	goodRequest.Header.Set("Connection", "upgrade")
	badRequest, _ := http.NewRequest("GET", "http://example.com", nil)
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "websocket request",
			args: args{
				req: goodRequest,
			},
			want: true,
		},
		{
			name: "not a websocket request",
			args: args{
				req: badRequest,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsWebSocketRequest(tt.args.req); got != tt.want {
				t.Errorf("IsWebSocketRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
