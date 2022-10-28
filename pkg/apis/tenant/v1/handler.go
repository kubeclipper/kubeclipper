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
	"fmt"
	"math/rand"
	"net/http"
	"time"

	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"
	"github.com/kubeclipper/kubeclipper/pkg/models/tenant"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/validation"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	"github.com/emicklei/go-restful"

	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

type handler struct {
	tenantOperator  tenant.Operator
	clusterOperator cluster.Operator
	iamOperator     iam.Operator
}

func newHandler(tenantOperator tenant.Operator, clusterOperator cluster.Operator, iamOperator iam.Operator) *handler {
	return &handler{
		tenantOperator:  tenantOperator,
		clusterOperator: clusterOperator,
		iamOperator:     iamOperator,
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
	if err := h.nodeCheck(ctx, p.Spec.Nodes, p.Name); err != nil {
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

func (h *handler) nodeCheck(ctx context.Context, nodes []string, projectName string) error {
	for _, nodeID := range nodes {
		node, err := h.clusterOperator.GetNode(ctx, nodeID)
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
	if clientrest.IsInformerRawQuery(req.Request) {
		result, err := h.tenantOperator.ListProjects(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	} else {
		result, err := h.tenantOperator.ListProjectsEx(req.Request.Context(), q)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_ = resp.WriteHeaderAndEntity(http.StatusOK, result)
	}
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

type PatchNodes struct {
	Nodes   []string `json:"nodes"`
	Project string   `json:"project"`
	OP      string   `json:"op"`
}

func (h *handler) AddOrRemoveNode(request *restful.Request, response *restful.Response) {
	pn := &PatchNodes{}
	if err := request.ReadEntity(pn); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	projectName := request.PathParameter("name")
	if pn.Project != projectName {
		restplus.HandleBadRequest(response, request, fmt.Errorf("the name of the object (%s) does not match the name on the URL (%s)", pn.Project, projectName))
		return
	}

	dryRun := query.GetBoolValueWithDefault(request, query.ParamDryRun, false)
	ctx := request.Request.Context()
	project, err := h.tenantOperator.GetProjectEx(ctx, projectName, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			restplus.HandleNotFound(response, request, err)
			return
		}
		restplus.HandleInternalError(response, request, err)
		return
	}

	if err = h.nodeCheck(ctx, pn.Nodes, pn.Project); err != nil {
		restplus.HandleBadRequest(response, request, err)
		return
	}

	if !dryRun {
		if project, err = mergeNodes(project, pn); err != nil {
			return
		}
		_, err = h.tenantOperator.UpdateProject(request.Request.Context(), project)
		if err != nil {
			restplus.HandleInternalError(response, request, err)
			return
		}
	}
	response.WriteHeader(http.StatusOK)
}

func mergeNodes(p *tenantv1.Project, pn *PatchNodes) (*tenantv1.Project, error) {
	set := sets.NewString(p.Spec.Nodes...)

	switch pn.OP {
	case corev1.NodesOperationAdd:
		set.Insert(pn.Nodes...)
	case corev1.NodesOperationRemove:
		set.Delete(pn.Nodes...)
	default:
		return nil, fmt.Errorf("invalid operation %s", pn.OP)
	}
	p.Spec.Nodes = set.List()
	return p, nil
}
