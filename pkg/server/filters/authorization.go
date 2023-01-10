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
	"context"
	"errors"

	"github.com/kubeclipper/kubeclipper/pkg/server/request"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

func WithAuthorization(authorizers authorizer.Authorizer) restful.FilterFunction {
	if authorizers == nil {
		logger.Warn("Authorization is disabled")
		return nil
	}
	return func(req *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		ctx := req.Request.Context()
		attributes, err := getAuthorizerAttributes(ctx)
		if err != nil {
			restplus.HandleInternalError(response, req, err)
			return
		}
		authorized, reason, err := authorizers.Authorize(ctx, attributes)
		if authorized == authorizer.DecisionAllow {
			chain.ProcessFilter(req, response)
			return
		}
		if err != nil {
			restplus.HandleInternalError(response, req, err)
			return
		}
		logger.Debug("request forbidden", zap.String("url", req.Request.RequestURI),
			zap.String("reason", reason))

		restplus.HandleForbidden(response, req, errors.New(reason))
	}
}

func getAuthorizerAttributes(ctx context.Context) (authorizer.Attributes, error) {
	attribs := authorizer.AttributesRecord{}

	user, ok := request.UserFrom(ctx)
	if ok {
		attribs.User = user
	}

	requestInfo, found := request.InfoFrom(ctx)
	if !found {
		return nil, errors.New("no RequestInfo found in the context")
	}

	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Verb = requestInfo.Verb
	attribs.APIGroup = requestInfo.APIGroup
	attribs.APIVersion = requestInfo.APIVersion
	attribs.Resource = requestInfo.Resource
	attribs.Subresource = requestInfo.Subresource
	attribs.Name = requestInfo.Name

	return &attribs, nil
}
