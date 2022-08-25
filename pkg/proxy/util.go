package proxy

import (
	"strings"

	kcctloptions "github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	agentconfig "github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"
)

// ReplaceByProxy when config proxyServer，replace nats server and static server by proxyServer.
func ReplaceByProxy(c *agentconfig.Config) {
	if c.Metadata.ProxyServer != "" {
		c.MQOptions.Client.ServerAddress = []string{replaceDomain(c.MQOptions.Client.ServerAddress[0])}
		c.DownloaderOptions.Address = replaceDomain(c.DownloaderOptions.Address)
	}
}

// input: 192.168.10.1:9889 output: proxy.kubeclipper.io:9889
// input: http://192.168.10.1:9889 output: http://proxy.kubeclipper.io:9889
func replaceDomain(address string) string {
	url, ok := httputil.IsURL(address)
	if ok {
		url.Host = replaceDomain(url.Host)
		return url.String()
	}

	split := strings.Split(address, ":")
	if len(split) != 2 {
		return address
	}
	split[0] = kcctloptions.NatsAltNameProxy
	return strings.Join(split, ":")
}
