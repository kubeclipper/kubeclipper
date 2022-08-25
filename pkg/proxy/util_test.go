package proxy

import (
	"reflect"
	"testing"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	agentconfig "github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
)

func TestReplaceByProxy(t *testing.T) {
	withProxy := &agentconfig.Config{
		Metadata:          options.Metadata{ProxyServer: "192.168.20.220"},
		DownloaderOptions: &downloader.Options{Address: "192.168.10.10:8081"},
		MQOptions:         &natsio.NatsOptions{Client: natsio.ClientOptions{ServerAddress: []string{"192.168.10.10:9889", "192.168.10.11:9889", "192.168.10.12:9889"}}},
	}
	withProxyTarget := &agentconfig.Config{
		Metadata:          options.Metadata{ProxyServer: "192.168.20.220"},
		DownloaderOptions: &downloader.Options{Address: options.NatsAltNameProxy + ":8081"},
		MQOptions:         &natsio.NatsOptions{Client: natsio.ClientOptions{ServerAddress: []string{options.NatsAltNameProxy + ":9889"}}},
	}
	ReplaceByProxy(withProxy)
	if !reflect.DeepEqual(withProxy, withProxyTarget) {
		t.Fatalf("want %v got %v", withProxy, withProxy)
	}
	withoutProxy := &agentconfig.Config{
		DownloaderOptions: &downloader.Options{Address: "192.168.10.10:8081"},
		MQOptions:         &natsio.NatsOptions{Client: natsio.ClientOptions{ServerAddress: []string{"192.168.10.10:9889", "192.168.10.11:9889", "192.168.10.12:9889"}}},
	}
	withoutProxyTarget := &agentconfig.Config{
		DownloaderOptions: &downloader.Options{Address: "192.168.10.10:8081"},
		MQOptions:         &natsio.NatsOptions{Client: natsio.ClientOptions{ServerAddress: []string{"192.168.10.10:9889", "192.168.10.11:9889", "192.168.10.12:9889"}}},
	}
	ReplaceByProxy(withoutProxy)
	if !reflect.DeepEqual(withoutProxy, withoutProxyTarget) {
		t.Fatalf("want %v got %v", withoutProxyTarget, withoutProxy)
	}
}

func Test_replaceDomain(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "1", args: args{address: "192.168.10.10:9889"}, want: options.NatsAltNameProxy + ":9889"},
		{name: "2", args: args{address: "192.168.10.10:8081"}, want: options.NatsAltNameProxy + ":8081"},
		{name: "3", args: args{address: "http://192.168.10.10:8081"}, want: "http://" + options.NatsAltNameProxy + ":8081"},
		{name: "4", args: args{address: "https://192.168.10.10:8081"}, want: "https://" + options.NatsAltNameProxy + ":8081"},
		{name: "5", args: args{address: ""}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replaceDomain(tt.args.address); got != tt.want {
				t.Errorf("replaceDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}
