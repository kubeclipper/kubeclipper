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

package filters

import (
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"

	"github.com/emicklei/go-restful"

	"github.com/kubeclipper/kubeclipper/pkg/auditing"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/server/request"
)

func WithAudit(p auditing.Interface) restful.FilterFunction {
	if p == nil {
		logger.Warn("Authentication is disabled")
		return nil
	}
	a := PathExclude{
		prefixes: []string{"oauth/"},
	}

	return func(req *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		if !p.Enabled() || clientrest.IsInformerRawQuery(req.Request) {
			chain.ProcessFilter(req, response)
			return
		}
		pth := strings.TrimPrefix(req.Request.URL.Path, "/")
		info, ok := request.InfoFrom(req.Request.Context())
		if !ok || (!info.IsResourceRequest && !a.hasPrefix(pth)) {
			chain.ProcessFilter(req, response)
			return
		}
		e := p.LogRequestObject(req.Request, info)
		if e != nil {
			respCapture := auditing.NewResponseCapture(response.ResponseWriter)
			response.ResponseWriter = respCapture
			chain.ProcessFilter(req, response)
			go p.LogResponseObject(e, respCapture)
			return
		}
		chain.ProcessFilter(req, response)
	}
}
