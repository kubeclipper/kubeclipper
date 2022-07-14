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

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/server/runtime"
)

const (
	CoreAuditTag = "Core-Audit"
)

func AddToContainer(c *restful.Container, operator platform.Operator) error {
	webservice := runtime.NewWebService(schema.GroupVersion{Group: "audit.kubeclipper.io", Version: "v1"})
	h := newHandler(operator)

	webservice.Route(webservice.GET("/events").
		To(h.ListEvents).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreAuditTag}).
		Doc("List audit events.").
		Param(webservice.QueryParameter(query.PagingParam, "paging query, e.g. limit=100,page=1").
			Required(false).
			DataFormat("limit=%d,page=%d").
			DefaultValue("limit=10,page=1")).
		Param(webservice.QueryParameter(query.ParameterLabelSelector, "resource filter by metadata label").
			Required(false).
			DataFormat("labelSelector=%s=%s")).
		Param(webservice.QueryParameter(query.ParameterFieldSelector, "resource filter by field").
			Required(false).
			DataFormat("fieldSelector=%s=%s")).
		Param(webservice.QueryParameter(query.ParamReverse, "resource sort reverse or not").Required(false).
			DataType("boolean")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), models.PageableResponse{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/events/{name}").
		To(h.DescribeEvent).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreAuditTag}).
		Doc("Describe event.").
		Param(webservice.PathParameter(query.ParameterName, "event name").
			Required(true).
			DataType("string")).
		Param(webservice.QueryParameter(query.ParameterResourceVersion, "resource version to query").
			Required(false).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), corev1.Event{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	c.Add(webservice)
	return nil
}
