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

	"github.com/kubeclipper/kubeclipper/pkg/query"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"

	serverconfig "github.com/kubeclipper/kubeclipper/pkg/server/config"

	"github.com/emicklei/go-restful"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

type handler struct {
	platformOperator platform.Operator
	serverConfig     *serverconfig.Config
}

func newHandler(operator platform.Operator, config *serverconfig.Config) *handler {
	return &handler{
		platformOperator: operator,
		serverConfig:     config,
	}
}

func (h *handler) ListOfflineResource(request *restful.Request, response *restful.Response) {
	online := query.GetBoolValueWithDefault(request, "online", false)
	var (
		metas = scheme.ComponentMetaList{}
		err   error
	)
	if online {
		err = metas.ReadFromCloud(true)
	} else {
		err = metas.ReadFile(h.serverConfig.StaticServerOptions.Path, true)
	}

	if err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}

	items := make([]interface{}, 0, len(metas))
	for i := range metas {
		items = append(items, metas[i])
	}
	result := models.PageableResponse{
		Items:      items,
		TotalCount: len(items),
	}

	_ = response.WriteHeaderAndEntity(http.StatusOK, result)
}

// Deprecated: use core/v1/handler.DescribeTemplate instead
func (h *handler) DescribeTemplate(req *restful.Request, resp *restful.Response) {
	setting, err := h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, setting.Template)
}

// Deprecated: use core/v1/handler.UpdateTemplate instead
func (h *handler) UpdateTemplate(req *restful.Request, resp *restful.Response) {
	c := &v1.DockerRegistry{}
	if err := req.ReadEntity(c); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	for index := range c.InsecureRegistry {
		if c.InsecureRegistry[index].CreateAt.IsZero() {
			c.InsecureRegistry[index].CreateAt = metav1.Now()
		}
	}
	setting, err := h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	setting.Template = *c
	_, err = h.platformOperator.UpdatePlatformSetting(req.Request.Context(), setting)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) GetSSHRSAKey(req *restful.Request, resp *restful.Response) {
	t := v1.WebTerminal{}
	setting, err := h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if setting == nil {
		restplus.HandlerErrorWithCustomCode(resp, req, http.StatusExpectationFailed, http.StatusExpectationFailed, "platform setting should not be nil", nil)
		return
	}
	if setting.Terminal.PrivateKey == "" {
		t.PrivateKey, t.PublicKey, err = certs.GetSSHKeyPair(512)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		setting.Terminal = t
		setting, err = h.platformOperator.UpdatePlatformSetting(req.Request.Context(), setting)
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
	}
	setting.Terminal.PrivateKey = ""
	_ = resp.WriteHeaderAndEntity(http.StatusOK, setting.Terminal)
}

func (h *handler) CreateSSHRSAKey(req *restful.Request, resp *restful.Response) {
	t := v1.WebTerminal{}
	setting, err := h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	t.PrivateKey, t.PublicKey, err = certs.GetSSHKeyPair(512)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	setting.Terminal = t
	_, err = h.platformOperator.UpdatePlatformSetting(req.Request.Context(), setting)
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	resp.WriteHeader(http.StatusOK)
}
