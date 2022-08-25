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

	"k8s.io/component-base/version"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/utils/certs"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"

	serverconfig "github.com/kubeclipper/kubeclipper/pkg/server/config"

	"github.com/emicklei/go-restful"

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
	metas := scheme.PackageMetadata{}
	if err := metas.ReadMetadata(online, h.serverConfig.StaticServerOptions.Path); err != nil {
		restplus.HandleInternalError(response, request, err)
		return
	}
	result := kc.ComponentMeta{
		Rules:  metas.GetK8sVersionControlRules(version.Get().GitVersion),
		Addons: metas.Addons,
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
	var (
		err     error
		setting *v1.PlatformSetting
	)
	if err = req.ReadEntity(c); err != nil {
		restplus.HandleBadRequest(resp, req, err)
		return
	}
	for index := range c.InsecureRegistry {
		if c.InsecureRegistry[index].CreateAt.IsZero() {
			c.InsecureRegistry[index].CreateAt = metav1.Now()
		}
	}
	setting, err = h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if setting == nil || setting.Name == "" {
		setting = generatePlatformSetting()
		setting.Template = *c
		_, err = h.platformOperator.CreatePlatformSetting(req.Request.Context(), setting)
	} else {
		setting.Template = *c
		_, err = h.platformOperator.UpdatePlatformSetting(req.Request.Context(), setting)
	}
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	_ = resp.WriteHeaderAndEntity(http.StatusOK, c)
}

func (h *handler) GetSSHRSAKey(req *restful.Request, resp *restful.Response) {
	var (
		err     error
		setting *v1.PlatformSetting
	)

	setting, err = h.platformOperator.GetPlatformSetting(req.Request.Context())
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
	}
	if setting == nil || setting.Name == "" {
		setting = generatePlatformSetting()
		setting.Terminal, err = generateWebTerminal()
		if err != nil {
			restplus.HandleInternalError(resp, req, err)
			return
		}
		_, err = h.platformOperator.CreatePlatformSetting(req.Request.Context(), setting)
	} else {
		if setting.Terminal.PrivateKey == "" {
			setting.Terminal, err = generateWebTerminal()
			if err != nil {
				restplus.HandleInternalError(resp, req, err)
				return
			}
			_, err = h.platformOperator.UpdatePlatformSetting(req.Request.Context(), setting)
		}
	}
	if err != nil {
		restplus.HandleInternalError(resp, req, err)
		return
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

func generateWebTerminal() (v1.WebTerminal, error) {
	priv, pub, err := certs.GetSSHKeyPair(512)
	if err != nil {
		return v1.WebTerminal{}, err
	}
	return v1.WebTerminal{
		PrivateKey: priv,
		PublicKey:  pub,
	}, nil

}

func generatePlatformSetting() *v1.PlatformSetting {
	return &v1.PlatformSetting{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PlatformSetting",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "system-",
		},
	}
}
