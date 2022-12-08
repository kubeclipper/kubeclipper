package httputil

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
)

func TestCommonRequest(t *testing.T) {
	type args struct {
		requestURL string
		httpMethod string
		header     map[string]string
		rawQuery   map[string]string
		postBody   json.RawMessage
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		want1   int
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				requestURL: "https://www.baidu.com",
				httpMethod: "GET",
				header:     map[string]string{},
				rawQuery:   map[string]string{},
				postBody:   nil,
			},
			want1: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got, err := CommonRequest(tt.args.requestURL, tt.args.httpMethod, tt.args.header, tt.args.rawQuery, tt.args.postBody)
			if (err != nil) != tt.wantErr {
				t.Errorf("CommonRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want1 {
				t.Errorf("CommonRequest() got1 = %v, want %v", got, tt.want1)
			}
		})
	}
}

func TestCodeDispose(t *testing.T) {
	type args struct {
		body []byte
		code int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "got a 200",
			args: args{
				body: []byte("this is a test"),
				code: 200,
			},
			want: []byte("this is a test"),
		},
		{
			name: "got a 202",
			args: args{
				body: []byte("this is a test"),
				code: 202,
			},
			want: []byte("this is a test"),
		},
		{
			name: "got a 500",
			args: args{
				body: []byte("this is a test"),
				code: 500,
			},
			want:    []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CodeDispose(tt.args.body, tt.args.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("CodeDispose() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CodeDispose() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	type args struct {
		u string
	}
	tests := []struct {
		name  string
		args  args
		want  url.URL
		want1 bool
	}{
		{
			name: "A correct url",
			args: args{
				u: "https://www.baidu.com",
			},
			want1: true,
		},
		{
			name: "An incorrect url",
			args: args{
				u: "",
			},
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got1 := IsURL(tt.args.u)
			if got1 != tt.want1 {
				t.Errorf("IsURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
