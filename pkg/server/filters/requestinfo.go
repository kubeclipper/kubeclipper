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
	"github.com/emicklei/go-restful"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

func WithRequestInfo(resolver request.Resolver) restful.FilterFunction {
	if resolver == nil {
		logger.Warn("RequestInfo is disabled")
		return nil
	}
	return func(r *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		ctx := r.Request.Context()
		info, err := resolver.NewRequestInfo(r.Request)
		if err != nil {
			restplus.HandleInternalError(response, r, err)
			return
		}
		r.Request = r.Request.WithContext(request.WithInfo(ctx, info))
		chain.ProcessFilter(r, response)
	}
}
