package kc

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/query"
)

func TestQueries_ToRawQuery(t *testing.T) {
	q := query.New()
	q.LabelSelector = "a=b"
	q.FieldSelector = "metadata.name=demo"
	q.Watch = true
	q.FuzzySearch = map[string]string{
		"name": "demo",
	}
	kcq := Queries(*q)
	rawQuery := kcq.ToRawQuery()
	got := rawQuery.Encode()

	target := make(url.Values)
	target.Set("paging", "limit=-1,page=0")
	target.Set("watch", "true")
	target.Set("fieldSelector", "metadata.name=demo")
	target.Set("labelSelector", "a=b")
	target.Set("fuzzy", "name~demo")
	want := target.Encode()

	if got != want {
		t.Fatalf("want:%s got:%s\n", want, got)
	}
}

func TestClient_head(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method head",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method head with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.head(tt.args.ctx, tt.args.path, tt.args.query, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("head() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_get(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method get",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method get with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.get(tt.args.ctx, tt.args.path, tt.args.query, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_patch(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		obj     interface{}
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method patch",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method patch with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.patch(tt.args.ctx, tt.args.path, tt.args.query, tt.args.obj, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("patch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_post(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		obj     interface{}
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method post",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method post with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.post(tt.args.ctx, tt.args.path, tt.args.query, tt.args.obj, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("post() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_postRaw(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		body    io.Reader
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method postRaw",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				body:    nil,
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method postRaw with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				body:    nil,
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.postRaw(tt.args.ctx, tt.args.path, tt.args.query, tt.args.body, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("postRaw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_put(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		obj     interface{}
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method put",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method put with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				obj:     nil,
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.put(tt.args.ctx, tt.args.path, tt.args.query, tt.args.obj, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("put() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_putRaw(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		body    io.Reader
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method putRaw",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				body:    nil,
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method putRaw with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				body:    nil,
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.putRaw(tt.args.ctx, tt.args.path, tt.args.query, tt.args.body, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("putRaw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_delete(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		ctx     context.Context
		path    string
		query   url.Values
		headers map[string][]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for method delete",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "www.baidu.com",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
		},
		{
			name: "test for method delete with a wrong host",
			fields: fields{
				client: http.DefaultClient,
				scheme: defaultHTTPScheme,
				host:   "localhost111",
			},
			args: args{
				ctx:     context.TODO(),
				path:    "",
				query:   url.Values{},
				headers: headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.delete(tt.args.ctx, tt.args.path, tt.args.query, tt.args.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestClient_buildRequest(t *testing.T) {
	type fields struct {
		client                *http.Client
		host                  string
		bearerToken           string
		scheme                string
		insecureSkipTLSVerify bool
		tlsServerName         string
		caPool                *x509.CertPool
	}
	type args struct {
		method string
		path   string
		body   io.Reader
		h      headers
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test for a get request",
			fields: fields{
				client: http.DefaultClient,
				host:   "www.baidu.com",
				scheme: defaultHTTPScheme,
			},
			args: args{
				method: "Get",
				path:   "",
				body:   nil,
				h:      headers{},
			},
		},
		{
			name: "test for a get request with wrong method",
			fields: fields{
				client: http.DefaultClient,
				host:   "www.baidu.com",
				scheme: defaultHTTPScheme,
			},
			args: args{
				method: "A new method",
				path:   "",
				body:   nil,
				h:      headers{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &Client{
				client:                tt.fields.client,
				host:                  tt.fields.host,
				bearerToken:           tt.fields.bearerToken,
				scheme:                tt.fields.scheme,
				insecureSkipTLSVerify: tt.fields.insecureSkipTLSVerify,
				tlsServerName:         tt.fields.tlsServerName,
				caPool:                tt.fields.caPool,
			}
			_, err := cli.buildRequest(tt.args.method, tt.args.path, tt.args.body, tt.args.h)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
