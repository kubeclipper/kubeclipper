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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
)

type Action string

const (
	ActionApply  Action = "kubectl-apply"
	ActionDelete Action = "kubectl-delete"
	ActionProxy  Action = "proxy"
)

const (
	ProxyHandlerPrefix = "/cluster-proxy/"
)

type handler struct {
	clusterReader cluster.ClusterReader
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	clusterName, action, k8sResource := parseClusterFromPath(request.URL.Path)
	if clusterName == "" || (action == ActionProxy && k8sResource == "") {
		http.Error(writer, "cluster name cannot be empty", http.StatusBadRequest)
		return
	}

	c, err := h.clusterReader.GetClusterEx(request.Context(), clusterName, "0")
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

	if action != ActionProxy {
		emulateKubectlHandler(writer, request, action, restConfig)
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

func newClusterProxyHandler(restConfig *rest.Config, requestURL *url.URL) (http.Handler, error) {
	clusterURL, err := toClusterURL(restConfig.Host, requestURL)
	if err != nil {
		return nil, err
	}
	restTransport, err := rest.TransportFor(restConfig)
	if err != nil {
		return nil, err
	}
	upgrade, err := makeUpgradeTransport(restConfig, 0)
	if err != nil {
		return nil, err
	}
	h := proxy.NewUpgradeAwareHandler(clusterURL, restTransport, true, false, nil)
	h.UpgradeTransport = upgrade
	return h, nil
}

func toClusterURL(restHost string, requestURL *url.URL) (*url.URL, error) {
	if requestURL == nil {
		return nil, fmt.Errorf("request url is nil")
	}
	if !strings.HasPrefix(strings.ToLower(restHost), "http") {
		restHost = "http://" + restHost
	}
	u, err := url.Parse(restHost)
	if err != nil {
		return nil, fmt.Errorf("parse restHost:%w", err)
	}
	if requestURL != nil {
		// without host and scheme
		u.Opaque = requestURL.Opaque
		u.User = requestURL.User
		u.Path = requestURL.Path
		u.RawPath = requestURL.RawPath
		u.OmitHost = requestURL.OmitHost
		u.ForceQuery = requestURL.ForceQuery
		u.RawQuery = requestURL.RawQuery
		u.Fragment = requestURL.Fragment
		u.RawFragment = requestURL.RawFragment
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

func parseClusterFromPath(p string) (clusterName string, action Action, proxyURL string) {
	// /cluster/{clusterName}/{resource}
	// /cluster/{clusterName}/kubectl-apply
	// /cluster/{clusterName}/kubectl-delete
	p = strings.TrimPrefix(p, ProxyHandlerPrefix)
	parts := strings.SplitN(p, "/", 2)

	if len(parts) == 2 {
		if strings.HasPrefix(parts[1], "kubectl-") {
			return parts[0], Action(parts[1]), ""
		}
		return parts[0], ActionProxy, parts[1]
	}
	return parts[0], "", ""
}
