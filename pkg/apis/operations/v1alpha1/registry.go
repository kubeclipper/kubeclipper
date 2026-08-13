/*
 *
 *  * Copyright 2026 KubeClipper Authors.
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

package v1alpha1

import (
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	operationv2 "github.com/kubeclipper/kubeclipper/pkg/models/operationv2"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	serverruntime "github.com/kubeclipper/kubeclipper/pkg/server/runtime"
)

const operationTag = "Operation-v2"

func AddToContainer(
	container *restful.Container,
	store operationv2.Store,
	clusterReader cluster.OperatorReader,
	caFile, certFile, keyFile string,
) error {
	logClient, err := newLogClient(caFile, certFile, keyFile)
	if err != nil {
		return err
	}
	h := &handler{store: store, clusterReader: clusterReader, logClient: logClient}
	container.Add(SetupWebService(h))
	return nil
}

func SetupWebService(h *handler) *restful.WebService {
	ws := serverruntime.NewWebService(operations.SchemeGroupVersion)

	ws.Route(ws.POST("/operations").To(h.createOperation).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Reads(operations.Operation{}).
		Returns(http.StatusCreated, http.StatusText(http.StatusCreated), operations.Operation{}))
	ws.Route(ws.GET("/operations").To(h.listOperations).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.OperationList{}))
	ws.Route(ws.GET("/operations/{name}").To(h.getOperation).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.Operation{}))
	ws.Route(ws.POST("/operations/{name}/cancel").To(h.cancelOperation).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Reads(operations.OperationControlRequest{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.Operation{}))
	ws.Route(ws.POST("/operations/{name}/retry").To(h.retryOperation).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Reads(operations.OperationControlRequest{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.Operation{}))

	ws.Route(ws.GET("/operationtasks").To(h.listTasks).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.OperationTaskList{}))
	ws.Route(ws.GET("/operationtasks/{name}").To(h.getTask).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.OperationTask{}))
	ws.Route(ws.GET("/operationtasks/{name}/logs").To(h.getTaskLogs).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), map[string]any{}))
	ws.Route(ws.PUT("/operationtasks/{name}/status").To(h.updateTaskStatus).
		Metadata(restfulspec.KeyOpenAPITags, []string{operationTag}).
		Reads(operations.OperationTask{}).
		Returns(http.StatusOK, http.StatusText(http.StatusOK), operations.OperationTask{}))

	return ws
}
