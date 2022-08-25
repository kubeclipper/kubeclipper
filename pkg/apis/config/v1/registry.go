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

package v1

import (
	"net/http"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	restfulspec "github.com/emicklei/go-restful-openapi"

	"github.com/kubeclipper/kubeclipper/pkg/component"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/runtime/schema"

	serverconfig "github.com/kubeclipper/kubeclipper/pkg/server/config"
	"github.com/kubeclipper/kubeclipper/pkg/server/runtime"
)

const (
	GroupName = "config.kubeclipper.io"
)

const (
	StatusOK      = "ok"
	CoreConfigTag = "Core-Config"
)

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}

func AddToContainer(c *restful.Container, platformOperator platform.Operator, config *serverconfig.Config) error {
	h := newHandler(platformOperator, config)

	webservice := runtime.NewWebService(GroupVersion)
	webservice.Route(webservice.GET("/oauth").
		Doc("Information about the authorization server are published.").
		To(func(request *restful.Request, response *restful.Response) {
			_ = response.WriteEntity(config.AuthenticationOptions.OAuthOptions)
		}))

	webservice.Route(webservice.GET("/configz").
		Doc("Information about the server configuration").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		To(func(request *restful.Request, response *restful.Response) {
			_ = response.WriteAsJson(config.ToMap())
		}).Returns(http.StatusOK, StatusOK, map[string]bool{}))

	webservice.Route(webservice.GET("/components").
		Doc("Information about components").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		Param(webservice.QueryParameter(query.ParameterLabelSelector, "resource filter by metadata label").
			Required(false).
			DataFormat("labelSelector=%s=%s")).
		Param(webservice.QueryParameter("lang", "component language, support en and zh").
			Required(false).
			DataFormat("lang=en")).
		To(func(request *restful.Request, response *restful.Response) {
			q := query.ParseQueryParameter(request)
			langStr := request.QueryParameter("lang")
			var lang component.Lang
			switch component.Lang(langStr) {
			case component.Chinese:
				lang = component.Chinese
			case component.English:
				fallthrough
			default:
				lang = component.English
			}
			_ = response.WriteHeaderAndEntity(http.StatusOK, component.GetJSONSchemas(q.GetLabelSelector(), lang))
		}).Returns(http.StatusOK, StatusOK, []component.Meta{}))

	webservice.Route(webservice.GET("/componentmeta").
		To(h.ListOfflineResource).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		Doc("List offline resource metadata info.").
		Param(webservice.QueryParameter("online", "online or offline resource").
			Required(false).
			DefaultValue("false")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), kc.ComponentMeta{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil))

	webservice.Route(webservice.GET("/template").
		Doc("Information about platform template").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		To(h.DescribeTemplate).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), v1.DockerRegistry{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))
	webservice.Route(webservice.PUT("/template").
		Doc("Update template").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		To(h.UpdateTemplate).
		Reads(v1.DockerRegistry{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), v1.DockerRegistry{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/terminal.key").
		Doc("Get rsa public key").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		To(h.GetSSHRSAKey).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), v1.WebTerminal{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))
	webservice.Route(webservice.POST("/terminal.key").
		Doc("Create rsa key pair").
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreConfigTag}).
		To(h.CreateSSHRSAKey).
		Reads(v1.WebTerminal{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), v1.WebTerminal{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	c.Add(webservice)
	return nil
}
