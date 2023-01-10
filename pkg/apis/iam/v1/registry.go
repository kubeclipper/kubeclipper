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

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/server/runtime"
)

const (
	CoreIAMTag = "Core-IAM"
)

type PasswordReset struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// AddToContainer add webservice to container.
func AddToContainer(c *restful.Container, iamOperator iam.Operator, authz authorizer.Authorizer, tokenOperator auth.TokenManagementInterface) error {

	webservice := runtime.NewWebService(schema.GroupVersion{Group: "iam.kubeclipper.io", Version: "v1"})

	h := newHandler(iamOperator, authz, tokenOperator)

	webservice.Route(webservice.GET("/tokens").
		To(h.ListTokens).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("List tokens.").
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
		Returns(http.StatusOK, http.StatusText(http.StatusOK), models.PageableResponse{}))

	webservice.Route(webservice.GET("/tokens/{name}").
		To(h.DescribeToken).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Describe token.").
		Param(webservice.PathParameter(query.ParameterName, "token name").
			Required(true).
			DataType("string")).
		Param(webservice.QueryParameter(query.ParameterResourceVersion, "resource version to query").
			Required(false).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.Token{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), nil))

	webservice.Route(webservice.POST("/users").
		To(h.CreateUsers).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Create users.").
		Reads(iamv1.User{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.User{}).
		Returns(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/users").
		To(h.ListUsers).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("List users.").
		Param(webservice.QueryParameter(query.PagingParam, "paging query, e.g. limit=100,page=1").
			Required(false).
			DataFormat("limit=%d,page=%d").
			DefaultValue("limit=10,page=1")).
		Param(webservice.QueryParameter(query.ParamRole, "resource filter by user role").
			Required(false).
			DataFormat("role=%s")).
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

	webservice.Route(webservice.HEAD("/users").
		To(h.CheckUserExist).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Check users exist.").
		Param(webservice.QueryParameter(query.ParameterLabelSelector, "resource filter by metadata label").
			Required(false).
			DataFormat("labelSelector=%s=%s")).
		Param(webservice.QueryParameter(query.ParameterFieldSelector, "resource filter by field").
			Required(false).
			DataFormat("fieldSelector=%s=%s")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/users/{name}").
		To(h.DescribeUser).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Describe user.").
		Param(webservice.PathParameter(query.ParameterName, "user name").
			Required(true).
			DataType("string")).
		Param(webservice.QueryParameter(query.ParameterResourceVersion, "resource version to query").
			Required(false).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.User{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.DELETE("/users/{name}").
		To(h.DeleteUser).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Delete user.").
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/users/{name}").
		To(h.UpdateUser).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Update user profile.").
		Reads(iamv1.User{}).
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.User{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/users/{name}/password").
		To(h.UpdateUserPassword).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Update user password.").
		Reads(PasswordReset{}).
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/users/{name}/loginrecords").
		To(h.ListUserLoginRecords).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("List user LoginRecords.").
		Param(webservice.PathParameter("name", "user name")).
		Param(webservice.QueryParameter(query.PagingParam, "paging query, e.g. limit=100,page=1").
			Required(false).
			DataFormat("limit=%d,page=%d").
			DefaultValue("limit=10,page=1")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusForbidden, http.StatusText(http.StatusForbidden), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/users/{name}/enable").
		To(h.EnableUser).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Enable user.").
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusForbidden, http.StatusText(http.StatusForbidden), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/users/{name}/disable").
		To(h.DisableUser).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Disable user.").
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusForbidden, http.StatusText(http.StatusForbidden), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/users/{name}/roles").
		To(h.RetrieveRoleTemplates).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Get user role rules.").
		Param(webservice.PathParameter("name", "user name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), []iamv1.GlobalRole{}).
		Returns(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.POST("/roles").
		To(h.CreateRoles).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Create roles.").
		Reads(iamv1.GlobalRole{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.GlobalRole{}).
		Returns(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/roles").
		To(h.ListRoles).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("List roles.").
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
		Returns(http.StatusOK, http.StatusText(http.StatusOK), models.PageableResponse{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.HEAD("/roles").
		To(h.CheckRolesExist).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Check roles exist.").
		Param(webservice.QueryParameter(query.ParameterLabelSelector, "resource filter by metadata label").
			Required(false).
			DataFormat("labelSelector=%s=%s")).
		Param(webservice.QueryParameter(query.ParameterFieldSelector, "resource filter by field").
			Required(false).
			DataFormat("fieldSelector=%s=%s")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.GET("/roles/{name}").
		To(h.DescribeRole).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Describe role.").
		Param(webservice.PathParameter(query.ParameterName, "role name").
			Required(true).
			DataType("string")).
		Param(webservice.QueryParameter(query.ParameterResourceVersion, "resource version to query").
			Required(false).
			DataType("string")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.GlobalRole{}).
		Returns(http.StatusNotFound, http.StatusText(http.StatusNotFound), errors.HTTPError{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.DELETE("/roles/{name}").
		To(h.DeleteRole).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Delete role.").
		Param(webservice.PathParameter("name", "role name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), nil).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	webservice.Route(webservice.PUT("/roles/{name}").
		To(h.UpdateRole).
		Metadata(restfulspec.KeyOpenAPITags, []string{CoreIAMTag}).
		Doc("Update role.").
		Reads(iamv1.GlobalRole{}).
		Param(webservice.PathParameter("name", "role name")).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), iamv1.GlobalRole{}).
		Returns(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), errors.HTTPError{}))

	c.Add(webservice)

	return nil
}
