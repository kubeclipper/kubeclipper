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

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"

	"github.com/kubeclipper/kubeclipper/pkg/models/tenant"
	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/server/runtime"
)

const (
	CoreTenantTag = "Core-Tenant"
)

func AddToContainer(c *restful.Container, tenantOperator tenant.Operator, clusterOperator cluster.Operator, iamOperator iam.Operator) error {

	webservice := runtime.NewWebService(schema.GroupVersion{Group: "tenant.kubeclipper.io", Version: "v1"})

	h := newHandler(tenantOperator, clusterOperator, iamOperator)

	webservice.Route(webservice.POST("/projects").
		To(h.CreateProject).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("Create project.").
		Reads(tenantv1.Project{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), tenantv1.Project{}).
		Returns(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/projects/{name}").
		To(h.DescribeProject).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("Describe project.").
		Param(webservice.PathParameter(query.ParameterName, "project name").
			Required(true).
			DataType("string")).
		Param(webservice.QueryParameter(query.ParameterResourceVersion, "resource version to query").
			Required(false).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), tenantv1.Project{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil))

	webservice.Route(webservice.PUT("/projects/{name}").
		To(h.UpdateProject).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("Update projects profile.").
		Reads(tenantv1.Project{}).
		Param(webservice.PathParameter("name", "project name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), tenantv1.Project{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.DELETE("/projects/{name}").
		To(h.DeleteProject).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("Delete project.").
		Param(webservice.PathParameter("name", "project name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/projects").
		To(h.ListProjects).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("List projects.").
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
		Param(webservice.QueryParameter(query.ParameterWatch, "watch request").Required(false).
			DataType("boolean")).
		Param(webservice.QueryParameter(query.ParameterTimeoutSeconds, "watch timeout seconds").
			DataType("integer").
			DefaultValue("60").
			Required(false)).
		Param(webservice.QueryParameter(query.ParameterFuzzySearch, "fuzzy search conditions").
			DataFormat("foo~bar,bar~baz").
			Required(false)).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), models.PageableResponse{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/projects/{name}/nodes").
		To(h.AddOrRemoveNode).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreTenantTag}).
		Doc("Add or remove node to project.").
		Reads(PatchNodes{}).
		Param(webservice.PathParameter(query.ParameterName, "project name").
			Required(true).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil))

	c.Add(webservice)
	return nil
}
