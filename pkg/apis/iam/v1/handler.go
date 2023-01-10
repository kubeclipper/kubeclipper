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
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/iam/validation"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/auth"

	"github.com/kubeclipper/kubeclipper/pkg/models"

	authuser "k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/models/iam"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	apirequest "github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
	"github.com/kubeclipper/kubeclipper/pkg/utils/hashutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

const (
	resourceExistCheckerHeader = "X-CHECK-EXIST"
)

type handler struct {
	iamOperator   iam.Operator
	authz         authorizer.Authorizer
	tokenOperator auth.TokenManagementInterface
}

func newHandler(iamOperator iam.Operator, authz authorizer.Authorizer, tokenOperator auth.TokenManagementInterface) *handler {
	return &handler{
		iamOperator:   iamOperator,
		authz:         authz,
		tokenOperator: tokenOperator,
	}
}

func (h *handler) CreateTokens(request *restful.Request, response *restful.Response) {
	c := &iamv1.Token{}
	if err := request.ReadEntity(c); err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	c, err := h.iamOperator.CreateToken(request.Request.Context(), c)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) watchToken(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.iamOperator.WatchTokens(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, iamv1.SchemeGroupVersion.WithKind("Token"), req, resp, timeout)
}

func (h *handler) ListTokens(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchToken(request, response, q)
		return
	}
	if clientrest.IsInformerRawQuery(request.Request) {
		result, err := h.iamOperator.ListTokens(request.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.iamOperator.ListTokenEx(context.TODO(), q)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		_ = response.WriteHeaderAndEntity(http.StatusOK, result)
	}
}

func (h *handler) DescribeToken(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.iamOperator.GetTokenEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) CreateUsers(request *restful.Request, response *restful.Response) {
	u := &iamv1.User{}
	if err := request.ReadEntity(u); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if errs := validation.ValidateUser(u); len(errs) > 0 {
		restplus.HandleBadRequest(response, request, errs.ToAggregate())
		return
	}
	var err error
	u.Spec.EncryptedPassword, err = hashutil.EncryptPassword(u.Spec.EncryptedPassword)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	role := u.Annotations[common.RoleAnnotation]
	delete(u.Annotations, common.RoleAnnotation)
	// TODO: make user status maintainer in controller
	stateActive := iamv1.UserActive
	u.Status.State = &stateActive
	u, err = h.iamOperator.CreateUser(request.Request.Context(), u)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	if role != "" {
		newRolebinding, err := h.iamOperator.GetRoleBindingEx(request.Request.Context(), role, "0")
		if err != nil {
			if !apimachineryErrors.IsNotFound(err) {
				restplus.HandleInternalError(response, request, err)
				return
			}
			_, err = h.iamOperator.CreateRoleBinding(request.Request.Context(), &iamv1.GlobalRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GlobalRoleBinding",
					APIVersion: iamv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: role,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: iamv1.SchemeGroupVersion.Group,
					Kind:     "GlobalRole",
					Name:     role,
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup: rbacv1.SchemeGroupVersion.Group,
						Kind:     rbacv1.UserKind,
						Name:     u.Name,
					},
				},
			})
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
		} else {
			newRolebinding.Subjects = append(newRolebinding.Subjects, rbacv1.Subject{
				Kind:      rbacv1.UserKind,
				APIGroup:  rbacv1.SchemeGroupVersion.Group,
				Name:      u.Name,
				Namespace: "",
			})
			_, err = h.iamOperator.UpdateRoleBinding(request.Request.Context(), newRolebinding)
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
		}
	}
	u.Spec.EncryptedPassword = ""
	if u.Annotations == nil {
		u.Annotations = make(map[string]string)
	}
	u.Annotations[common.RoleAnnotation] = role
	_ = response.WriteHeaderAndEntity(http.StatusOK, u)
}

func (h *handler) ListUsers(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchUser(request, response, q)
		return
	}
	var (
		result *models.PageableResponse
		err    error
	)
	role := request.QueryParameter(query.ParamRole)
	if role != "" {
		result, err = h.iamOperator.ListUsersByRole(request.Request.Context(), q, role)
	} else {
		result, err = h.iamOperator.ListUserEx(request.Request.Context(), q, true, true)
	}
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) CheckUserExist(req *restful.Request, resp *restful.Response) {
	q := query.ParseQueryParameter(req)
	result, err := h.iamOperator.ListUserEx(context.TODO(), q, false, false)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if len(result.Items) > 0 {
		resp.Header().Set(resourceExistCheckerHeader, "true")
	} else {
		resp.Header().Set(resourceExistCheckerHeader, "false")
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) watchUser(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := query.MinTimeoutSeconds * time.Second
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	watcher, err := h.iamOperator.WatchUsers(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, iamv1.SchemeGroupVersion.WithKind("User"), req, resp, timeout)
}

func (h *handler) DescribeUser(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.iamOperator.GetUserEx(request.Request.Context(), name, resourceVersion, true, true)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) DeleteUser(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	// TODO: validate user before delete it
	user, err := h.iamOperator.GetUserEx(request.Request.Context(), name, "0", false, true)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("user has already not exist when delete", zap.String("username", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if _, ok := user.Annotations[common.AnnotationInternal]; ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("internal user can not be deleted"))
		return
	}
	role := user.Annotations[common.RoleAnnotation]
	// if user has rolebinding or other resource, should not delete it
	err = h.iamOperator.DeleteUser(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("user has already not exist when delete", zap.String("username", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if role != "" {
		// Notice: delete user and rolebinding not use transcation, should use controller
		oldRolebinding, err := h.iamOperator.GetRoleBindingEx(request.Request.Context(), role, "0")
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		for index, subjects := range oldRolebinding.Subjects {
			if subjects.Kind == rbacv1.UserKind && subjects.Name == user.Name {
				oldRolebinding.Subjects = append(oldRolebinding.Subjects[:index], oldRolebinding.Subjects[index+1:]...)
				break
			}
		}
		_, err = h.iamOperator.UpdateRoleBinding(request.Request.Context(), oldRolebinding)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) CreateRoles(request *restful.Request, response *restful.Response) {
	globalRole := &iamv1.GlobalRole{}
	if err := request.ReadEntity(globalRole); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	// TODO: add creator label to object meta
	// TODO: add role validation
	// if errs := validation.ValidateUser(u); len(errs) > 0 {
	//	restplus.HandleBadRequest(response, request, errs.ToAggregate())
	//	return
	// }

	globalRole.Rules = make([]rbacv1.PolicyRule, 0)
	if aggregateRoles := h.getAggregateRoles(globalRole.ObjectMeta); aggregateRoles != nil {
		for _, roleName := range aggregateRoles {
			aggregationRole, err := h.iamOperator.GetRoleEx(request.Request.Context(), roleName, "0")
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
			globalRole.Rules = append(globalRole.Rules, aggregationRole.Rules...)
		}
	}

	result, err := h.iamOperator.CreateRole(request.Request.Context(), globalRole)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) ListRoles(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	if q.Watch {
		h.watchUser(request, response, q)
		return
	}
	result, err := h.iamOperator.ListRoleEx(context.TODO(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) CheckRolesExist(request *restful.Request, response *restful.Response) {
	q := query.ParseQueryParameter(request)
	result, err := h.iamOperator.ListRoleEx(context.TODO(), q)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if len(result.Items) > 0 {
		response.Header().Set(resourceExistCheckerHeader, "true")
	} else {
		response.Header().Set(resourceExistCheckerHeader, "false")
	}
	response.WriteHeader(http.StatusOK)
}

//nolint:unused
func (h *handler) watchRole(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := query.MinTimeoutSeconds * time.Second
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	watcher, err := h.iamOperator.WatchRoles(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, iamv1.SchemeGroupVersion.WithKind("GlobalRole"), req, resp, timeout)
}

func (h *handler) getAggregateRoles(obj metav1.ObjectMeta) []string {
	if aggregateRolesAnnotation := obj.Annotations[common.AnnotationAggregationRoles]; aggregateRolesAnnotation != "" {
		var aggregateRoles []string
		if err := json.Unmarshal([]byte(aggregateRolesAnnotation), &aggregateRoles); err != nil {
			logger.Warn("invalid aggregation role annotation")
		}
		return aggregateRoles
	}
	return nil
}

func (h *handler) DescribeRole(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", request.QueryParameter(query.ParameterResourceVersion))
	c, err := h.iamOperator.GetRoleEx(request.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) DeleteRole(request *restful.Request, response *restful.Response) {
	name := request.PathParameter(query.ParameterName)
	// TODO: validate user before delete it
	// if role has rolebinding or other resource, should not delete it
	role, err := h.iamOperator.GetRoleEx(request.Request.Context(), name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("user has already not exist when delete", zap.String("role", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if _, ok := role.Annotations[common.AnnotationInternal]; ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("internal role can not be deleted"))
		return
	}

	err = h.iamOperator.DeleteRole(request.Request.Context(), name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("user has already not exist when delete", zap.String("username", name))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) UpdateRole(request *restful.Request, response *restful.Response) {
	role := request.PathParameter(query.ParameterName)

	var globalRole iamv1.GlobalRole
	if err := request.ReadEntity(&globalRole); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if globalRole.Name != role {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", globalRole.Name, role))
		return
	}

	oldRole, err := h.iamOperator.GetRoleEx(request.Request.Context(), role, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Debug("user has already not exist when delete", zap.String("role", role))
			response.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if _, ok := oldRole.Annotations[common.AnnotationInternal]; ok {
		restplus.HandleBadRequest(response, request, fmt.Errorf("internal role can not be updated"))
		return
	}

	globalRole.Rules = make([]rbacv1.PolicyRule, 0)
	if aggregateRoles := h.getAggregateRoles(globalRole.ObjectMeta); aggregateRoles != nil {
		for _, roleName := range aggregateRoles {
			aggregationRole, err := h.iamOperator.GetRoleEx(request.Request.Context(), roleName, "0")
			if err != nil {
				restplus.HandleInternalError(response, request, err)
				return
			}
			globalRole.Rules = append(globalRole.Rules, aggregationRole.Rules...)
		}
	}

	updated, err := h.iamOperator.UpdateRole(request.Request.Context(), &globalRole)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return

	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updated)
}

func (h *handler) RetrieveRoleTemplates(request *restful.Request, response *restful.Response) {
	username := request.PathParameter(query.ParameterName)
	role, err := h.iamOperator.GetRoleOfUser(request.Request.Context(), username)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}
	if role == nil {
		_ = response.WriteHeaderAndEntity(http.StatusOK, []interface{}{})
		return
	}
	roles, err := h.fetchAggregationRoles(request.Request.Context(), role)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, roles)
}

func (h *handler) UpdateUser(request *restful.Request, response *restful.Response) {
	username := request.PathParameter(query.ParameterName)
	var user iamv1.User
	if err := request.ReadEntity(&user); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if username != user.Name {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", user.Name, username))
		return
	}
	role := user.Annotations[common.RoleAnnotation]
	delete(user.Annotations, common.RoleAnnotation)

	updated, err := h.updateUser(request.Request.Context(), &user)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	updated.Spec.EncryptedPassword = ""

	operator, ok := apirequest.UserFrom(request.Request.Context())
	if role != "" && ok {
		err = h.updateRoleBinding(request.Request.Context(), operator, updated, role)
		if err != nil {
			if apimachineryErrors.IsForbidden(err) {
				restplus.HandleForbidden(response, request, err)
				return
			}
			restplus.HandleInternalError(response, request, err)
			return
		}
		if updated.Annotations == nil {
			updated.Annotations = make(map[string]string)
		}
		updated.Annotations[common.RoleAnnotation] = role
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, updated)
}

func (h *handler) updateUser(ctx context.Context, user *iamv1.User) (*iamv1.User, error) {
	u, err := h.iamOperator.GetUserEx(ctx, user.Name, "0", false, false)
	if err != nil {
		return nil, err
	}
	user.Spec.EncryptedPassword = u.Spec.EncryptedPassword
	user.Status = u.Status
	user.ResourceVersion = u.ResourceVersion
	return h.iamOperator.UpdateUser(ctx, user)
}

func (h *handler) updateRoleBinding(ctx context.Context, operator authuser.Info, user *iamv1.User, role string) error {
	oldRole, err := h.iamOperator.GetRoleOfUser(ctx, user.Name)
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		return err
	}
	if oldRole != nil && oldRole.Name == role {
		return nil
	}

	authRecord := &authorizer.AttributesRecord{
		User:            operator,
		Verb:            "update",
		APIGroup:        iamv1.GroupName,
		APIVersion:      iamv1.SchemeGroupVersion.Version,
		Resource:        "users",
		Subresource:     "",
		Name:            "",
		ResourceRequest: true,
		Path:            "",
	}
	decision, _, err := h.authz.Authorize(ctx, authRecord)
	if err != nil {
		return err
	}
	if decision != authorizer.DecisionAllow {
		return apimachineryErrors.NewForbidden(iamv1.Resource("user"), user.Name, fmt.Errorf("update global role binding is not allowed"))
	}

	if oldRole != nil {
		oldRolebinding, err := h.iamOperator.GetRoleBindingEx(ctx, oldRole.Name, "0")
		if err != nil {
			return nil
		}
		for index, subjects := range oldRolebinding.Subjects {
			if subjects.Kind == rbacv1.UserKind && subjects.Name == user.Name {
				oldRolebinding.Subjects = append(oldRolebinding.Subjects[:index], oldRolebinding.Subjects[index+1:]...)
				break
			}
		}
		_, err = h.iamOperator.UpdateRoleBinding(ctx, oldRolebinding)
		if err != nil {
			return err
		}
	}
	newRolebinding, err := h.iamOperator.GetRoleBindingEx(ctx, role, "0")
	if err != nil {
		if !apimachineryErrors.IsNotFound(err) {
			return err
		}
		_, err = h.iamOperator.CreateRoleBinding(ctx, &iamv1.GlobalRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       iamv1.KindGlobalRoleBinding,
				APIVersion: iamv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: role,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: iamv1.GroupName,
				Kind:     iamv1.KindGlobalRole,
				Name:     role,
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup: rbacv1.SchemeGroupVersion.Group,
					Kind:     rbacv1.UserKind,
					Name:     user.Name,
				},
			},
		})
		if err != nil {
			return err
		}
	} else {
		newRolebinding.Subjects = append(newRolebinding.Subjects, rbacv1.Subject{
			Kind:      rbacv1.UserKind,
			APIGroup:  rbacv1.SchemeGroupVersion.Group,
			Name:      user.Name,
			Namespace: "",
		})
		_, err = h.iamOperator.UpdateRoleBinding(ctx, newRolebinding)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *handler) UpdateUserPassword(request *restful.Request, response *restful.Response) {
	username := request.PathParameter(query.ParameterName)
	var passwordReset PasswordReset
	err := request.ReadEntity(&passwordReset)
	if err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	if passwordReset.NewPassword == "" {
		restplus.HandleBadRequest(response, request, fmt.Errorf("new password must be valid"))
		return
	}

	currentUser, ok := apirequest.UserFrom(request.Request.Context())
	if !ok {
		restplus.HandleInternalError(response, request, fmt.Errorf("can not obtain user info in request"))
		return
	}

	authRecord := &authorizer.AttributesRecord{
		User:            currentUser,
		Verb:            "update",
		APIGroup:        "iam.kubeclipper.io",
		APIVersion:      "v1",
		Resource:        "users/password",
		Subresource:     "",
		Name:            "",
		ResourceRequest: true,
		Path:            "",
	}

	decision, _, err := h.authz.Authorize(request.Request.Context(), authRecord)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if decision != authorizer.DecisionAllow || passwordReset.CurrentPassword != "" {
		ok, err := h.verifyUserPassword(request.Request.Context(), username, passwordReset.CurrentPassword)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
		if !ok {
			restplus.HandleBadRequest(response, request, fmt.Errorf("incorrect old password"))
			return
		}
		if passwordReset.CurrentPassword == passwordReset.NewPassword {
			restplus.HandleBadRequest(response, request, fmt.Errorf("new password cannot be same as old password"))
			return
		}
	}

	user, err := h.iamOperator.GetUserEx(request.Request.Context(), username, "0", false, false)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	encryptedPassword, err := hashutil.EncryptPassword(passwordReset.NewPassword)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	user.Spec.EncryptedPassword = encryptedPassword
	if _, err := h.iamOperator.UpdateUser(request.Request.Context(), user); err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	go func() {
		if err := h.tokenOperator.RevokeAllUserTokens(user.Name); err != nil {
			logger.Error("revoke all user token failed", zap.String("user", user.Name), zap.Error(err))
		}
	}()
	response.WriteHeader(http.StatusOK)
}

func (h *handler) ListUserLoginRecords(req *restful.Request, resp *restful.Response) {
	// TODO: check permission
	name := req.PathParameter(query.ParameterName)
	q := query.ParseQueryParameter(req)
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelUserReference, name)
	result, err := h.iamOperator.ListLoginRecordEx(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
}

func (h *handler) EnableUser(req *restful.Request, resp *restful.Response) {
	h.changeUserState(req, resp, true)
}

func (h *handler) DisableUser(req *restful.Request, resp *restful.Response) {
	h.changeUserState(req, resp, false)
}

func (h *handler) changeUserState(req *restful.Request, resp *restful.Response, enable bool) {
	name := req.PathParameter(query.ParameterName)
	user, err := h.iamOperator.GetUserEx(req.Request.Context(), name, "0", false, false)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			logger.Warnf("user %s not exist", user)
			resp.WriteHeader(http.StatusOK)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	var state iamv1.UserState
	if enable {
		state = iamv1.UserActive
	} else {
		state = iamv1.UserDisabled
	}

	if user.Status.State != nil && (*user.Status.State) == state {
		resp.WriteHeader(http.StatusOK)
		return
	}

	user.Status.State = &state
	_, err = h.iamOperator.UpdateUser(req.Request.Context(), user)
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	resp.WriteHeader(http.StatusOK)
}

func (h *handler) fetchAggregationRoles(ctx context.Context, role *iamv1.GlobalRole) ([]*iamv1.GlobalRole, error) {
	roles := make([]*iamv1.GlobalRole, 0)

	if v := role.Annotations[common.AnnotationAggregationRoles]; v != "" {
		var roleNames []string
		if err := json.Unmarshal([]byte(v), &roleNames); err != nil {
			return nil, err
		}

		for _, roleName := range roleNames {
			r, err := h.iamOperator.GetRoleEx(ctx, roleName, "0")
			if err != nil {
				if apimachineryErrors.IsNotFound(err) {
					logger.Warn("aggregation role invalid", zap.String("role", role.Name),
						zap.String("aggregation_role", roleName))
					continue
				}
				return nil, err
			}
			roles = append(roles, r)
		}
	}
	return roles, nil
}

func (h *handler) verifyUserPassword(ctx context.Context, username, password string) (bool, error) {
	user, err := h.iamOperator.GetUserEx(ctx, username, "0", false, false)
	if err != nil {
		return false, err
	}
	return hashutil.ComparePassword(password, user.Spec.EncryptedPassword), nil
}
