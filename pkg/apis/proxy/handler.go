/*
 *
 *  * Copyright 2021 KubeClipper Authors.
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
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
)

const (
	ProxyHandlerPrefix = "/cluster/"
)

type handler struct {
	clusterReader cluster.ClusterReader
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	clusterName, k8sResource := parseClusterFromPath(request.URL.Path)
	if clusterName == "" {
		http.Error(writer, fmt.Sprintf("cluster name cannot be empty"), http.StatusBadRequest)
		return
	}

	c, err := h.clusterReader.GetClusterEx(request.Context(), clusterName, "")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			http.Error(writer, fmt.Sprintf("cluster %s not exists", clusterName), http.StatusNotFound)
		} else {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if len(c.KubeConfig) == 0 {
		http.Error(writer, fmt.Sprintf("cluster %s not init", clusterName), http.StatusBadRequest)
		return
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(c.KubeConfig)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	request.URL.Path = k8sResource
	request.Header.Del("Authorization")

	proxyHandle, err := newClusterProxyHandler(restConfig, request.URL)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	proxyHandle.ServeHTTP(writer, request)
}

func newHandler(clusterReader cluster.ClusterReader) *handler {
	return &handler{
		clusterReader: clusterReader,
	}
}

func (h *handler) getClusterConfig(ctx context.Context, clusterName string) (*rest.Config, error) {
	clu, err := h.clusterReader.GetCluster(ctx, clusterName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return nil, restful.NewError(http.StatusNotFound, fmt.Sprintf("cluster %s not exists", clusterName))
		}
		return nil, err
	}
	if clu.KubeConfig == nil {
		return nil, restful.NewError(http.StatusNotFound, fmt.Sprintf("cluster %s clientset not init", clusterName))
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(clu.KubeConfig)
	if err != nil {
		return nil, err
	}
	return restCfg, nil
}

func newClusterProxyHandler(restConfig *rest.Config, requestURL *url.URL) (http.Handler, error) {
	clusterUrl, err := toClusterUrl(restConfig.Host, requestURL)
	if err != nil {
		return nil, err
	}
	restTransport, err := rest.TransportFor(restConfig)
	upgrade, err := makeUpgradeTransport(restConfig, 0)
	if err != nil {
		return nil, err
	}
	h := proxy.NewUpgradeAwareHandler(clusterUrl, restTransport, true, false, nil)
	h.UpgradeTransport = upgrade
	return h, nil
}

func toClusterUrl(restHost string, requestUrl *url.URL) (*url.URL, error) {
	if requestUrl == nil {
		return nil, fmt.Errorf("request url is nil")
	}
	if !strings.HasPrefix(strings.ToLower(restHost), "http") {
		restHost = "http://" + restHost
	}
	u, err := url.Parse(restHost)
	if err != nil {
		return nil, fmt.Errorf("parse restHost:%w", err)
	}
	if requestUrl != nil {
		// without host and scheme
		u.Opaque = requestUrl.Opaque
		u.User = requestUrl.User
		u.Path = requestUrl.Path
		u.RawPath = requestUrl.RawPath
		u.OmitHost = requestUrl.OmitHost
		u.ForceQuery = requestUrl.ForceQuery
		u.RawQuery = requestUrl.RawQuery
		u.Fragment = requestUrl.Fragment
		u.RawFragment = requestUrl.RawFragment
	}
	return u, nil
}

// copy from k8s.io/kubectl/pkg/proxy/proxy_server.go
// makeUpgradeTransport creates a transport that explicitly bypasses HTTP2 support
// for proxy connections that must upgrade.
func makeUpgradeTransport(config *rest.Config, keepalive time.Duration) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: keepalive,
		}).DialContext,
	})

	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}

func parseClusterFromPath(p string) (clusterName, res string) {
	p = strings.TrimPrefix(p, ProxyHandlerPrefix)
	items := strings.SplitN(p, "/", 2)
	switch len(items) {
	case 2:
		return items[0], items[1]
	case 1:
		return items[0], ""
	}
	return "", ""
}
