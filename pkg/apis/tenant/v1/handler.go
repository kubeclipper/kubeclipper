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

// Package v1 implements all tenant/v1 api.
package v1

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"
	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/models/tenant"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/validation"
	"github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	"github.com/emicklei/go-restful"

	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

type handler struct {
	tenantOperator  tenant.Operator
	clusterOperator cluster.Operator
	iamOperator     iam.Operator
	authorizer      authorizer.Authorizer
}

func newHandler(tenantOperator tenant.Operator, clusterOperator cluster.Operator, iamOperator iam.Operator, authorizer authorizer.Authorizer) *handler {
	return &handler{
		tenantOperator:  tenantOperator,
		clusterOperator: clusterOperator,
		iamOperator:     iamOperator,
		authorizer:      authorizer,
	}
}

func (h *handler) CreateProject(request *restful.Request, response *restful.Response) {
	p := &tenantv1.Project{}
	if err := request.ReadEntity(p); err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	if errs := validation.ValidateUserCreate(p); len(errs) > 0 {
		restplus.HandleBadRequest(response, request, errs.ToAggregate())
		return
	}

	ctx := request.Request.Context()
	if err := h.projectCheck(ctx, p); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}
	c, err := h.tenantOperator.CreateProject(ctx, p)
	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	// todo create default projectRole and binding adminRole to projectManager
	_ = response.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) projectCheck(ctx context.Context, p *tenantv1.Project) error {
	if err := NodeCheck(ctx, h.clusterOperator, p.Spec.Nodes, p.Name); err != nil {
		return err
	}
	_, err := h.iamOperator.GetUser(ctx, p.Spec.Manager)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return fmt.Errorf("project manager %s not found", p.Spec.Manager)
		}
		return err
	}

	return nil
}

// NodeCheck check is all node belong to specify project
func NodeCheck(ctx context.Context, clusterOperator cluster.Operator, nodes []string, projectName string) error {
	for _, nodeID := range nodes {
		node, err := clusterOperator.GetNode(ctx, nodeID)
		if err != nil {
			return err
		}
		nodeProject, ok := node.Labels[common.LabelProject]
		if ok && nodeProject != projectName {
			return fmt.Errorf("node %s already in project %s", nodeID, nodeProject)
		}
	}
	return nil
}

func (h *handler) DescribeProject(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter(query.ParameterName)
	resourceVersion := strutil.StringDefaultIfEmpty("0", req.QueryParameter(query.ParameterResourceVersion))
	reg, err := h.tenantOperator.GetProjectEx(req.Request.Context(), name, resourceVersion)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(resp, req, err)
			return
		}
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, reg)
}

func (h *handler) UpdateProject(request *restful.Request, response *restful.Response) {
	p := &tenantv1.Project{}
	if err := request.ReadEntity(p); err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	name := request.PathParameter("name")
	if p.Name != name {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", p.Name, name))
		return
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	ctx := request.Request.Context()
	_, err := h.tenantOperator.GetProjectEx(ctx, name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if errs := validation.ValidateProjectUpdate(p); len(errs) > 0 {
		restplus.HandleBadRequest(response, request, errs.ToAggregate())
		return
	}

	if err = h.projectCheck(ctx, p); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if !dryRun {
		p, err = h.tenantOperator.UpdateProject(request.Request.Context(), p)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	_ = response.WriteHeaderAndEntity(http.StatusOK, p)
}

func (h *handler) DeleteProject(request *restful.Request, response *restful.Response) {
	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	name := request.PathParameter("name")
	ctx := request.Request.Context()
	_, err := h.tenantOperator.GetProjectEx(ctx, name, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	// todo check clusters and nodes in projects

	if !dryRun {
		err = h.tenantOperator.DeleteProject(request.Request.Context(), name)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	response.WriteHeader(http.StatusOK)
}

func (h *handler) ListProjects(req *restful.Request, resp *restful.Response) {
	q := query.ParseQueryParameter(req)
	if q.Watch {
		h.watchProjects(req, resp, q)
		return
	}
	currentUser, ok := request.UserFrom(req.Request.Context())
	if !ok {
		restplus.HandleForbidden(resp, req, fmt.Errorf("can not obtain user info in request"))
		return
	}
	//  if current user have enough permission,we just list projects,
	//  if not,we list current user's projectRoleBinding,and get project from label
	projects, err := h.listProjects(req.Request.Context(), currentUser, q, clientrest.IsInformerRawQuery(req.Request))
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, projects)
}

func (h *handler) listProjects(ctx context.Context, reqUser user.Info, q *query.Query, isInformer bool) (any, error) {
	// check internal user system:kc-server
	if reqUser.GetName() != "system:kc-server" {
		userInfo := reqUser.(*user.DefaultInfo)
		userInfo.Groups = sliceutil.RemoveString(userInfo.Groups, func(item string) bool {
			return item == user.AllAuthenticated
		})
		reqUser = user.Info(userInfo)
	}

	listWS := &authorizer.AttributesRecord{
		User:            reqUser,
		Verb:            "list",
		APIGroup:        "*",
		Resource:        "projects",
		ResourceRequest: true,
		ResourceScope:   request.GlobalScope,
	}

	decision, _, err := h.authorizer.Authorize(ctx, listWS)
	if err != nil {
		return nil, err
	}

	// allowed to list all projects
	if decision == authorizer.DecisionAllow {
		if isInformer {
			return h.tenantOperator.ListProjects(ctx, q)
		}
		return h.tenantOperator.ListProjectsEx(ctx, q)
	}

	// retrieving associated resources through role binding
	workspaceRoleBindings, err := h.listProjectRoleBinding(ctx, reqUser)
	if err != nil {
		logger.Error("list project role binding", zap.Error(err))
		return nil, err
	}

	projectList := make([]tenantv1.Project, 0)
	for _, roleBinding := range workspaceRoleBindings {
		projectName := roleBinding.Labels[common.LabelProject]
		project, err := h.tenantOperator.GetProject(ctx, projectName)
		if err != nil {
			if apimachineryErrors.IsNotFound(err) {
				logger.Warnf("project role binding: %+v found but project not exist", roleBinding.Name)
				continue
			}
			return nil, err
		}

		//  label matching selector, remove duplicate entity
		if q.GetLabelSelector().Matches(labels.Set(project.Labels)) {
			projectList = append(projectList, *project)
		}
	}
	list := &tenantv1.ProjectList{
		Items: projectList,
	}
	return models.DefaultList(list.DeepCopyObject(), q, tenant.ProjectFilter, nil, nil)
}

func (h *handler) listProjectRoleBinding(ctx context.Context, user user.Info) ([]iamv1.ProjectRoleBinding, error) {
	binding, err := h.iamOperator.ListProjectRoleBinding(ctx, query.New())
	if err != nil {
		return nil, err
	}
	list := make([]iamv1.ProjectRoleBinding, 0)
	for _, item := range binding.Items {
		if len(item.Subjects) != 0 && strings.Contains(item.Subjects[0].Name, user.GetName()) {
			list = append(list, item)
		}
	}
	return list, nil
}

func (h *handler) watchProjects(req *restful.Request, resp *restful.Response, q *query.Query) {
	timeout := time.Duration(0)
	if q.TimeoutSeconds != nil {
		timeout = time.Duration(*q.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = time.Duration(float64(query.MinTimeoutSeconds) * (rand.Float64() + 1.0))
	}

	watcher, err := h.tenantOperator.WatchProjects(req.Request.Context(), q)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	restplus.ServeWatch(watcher, tenantv1.SchemeGroupVersion.WithKind("Project"), req, resp, timeout)
}
