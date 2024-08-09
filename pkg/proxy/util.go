/*
 *
 *  * Copyright 2024 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

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
