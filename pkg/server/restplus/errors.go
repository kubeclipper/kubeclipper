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

package restplus

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"go.uber.org/zap"
	apiError "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

func HandleInternalError(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusInternalServerError, response, req, http.StatusInternalServerError, "Internal server error", err)
}

// HandleBadRequest writes http.StatusBadRequest and log error
func HandleBadRequest(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusBadRequest, response, req, http.StatusBadRequest, "Bad request", err)
}

func HandleNotFound(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusNotFound, response, req, http.StatusNotFound, "Object not found", err)
}

func HandleForbidden(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusForbidden, response, req, http.StatusForbidden, "Forbidden", err)
}

func HandleUnauthorized(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusUnauthorized, response, req, http.StatusUnauthorized, "Unauthorized", err)
}

func HandleTooManyRequests(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusTooManyRequests, response, req, http.StatusTooManyRequests, "Too many request", err)
}

func HandleConflict(response *restful.Response, req *restful.Request, err error) {
	handle(http.StatusConflict, response, req, http.StatusConflict, "Request conflict", err)
}

func handle(statusCode int, response *restful.Response, req *restful.Request, code int, message string, err error) {
	var reason string
	if err != nil {
		reason = err.Error()
	}
	_ = response.WriteHeaderAndEntity(statusCode, errors.HTTPError{
		Code:    code,
		Message: message,
		Reason:  reason,
	})
}

func HandlerErrorWithCustomCode(response *restful.Response, req *restful.Request, httpStatus, code int, message string, err error) {
	handle(httpStatus, response, req, code, message, err)
}

func HandlerCrash() {
	if r := recover(); r != nil {
		logger.Error("handler crash", zap.Any("recover_for", r))
	}
}

func HandleError(response *restful.Response, req *restful.Request, err error) {
	var statusCode int
	switch t := err.(type) {
	case apiError.APIStatus:
		statusCode = int(t.Status().Code)
	case restful.ServiceError:
		statusCode = t.Code
	default:
		statusCode = http.StatusInternalServerError
	}
	handle(statusCode, response, req, statusCode, http.StatusText(statusCode), err)
}
