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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	userapi "k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	operationv2 "github.com/kubeclipper/kubeclipper/pkg/models/operationv2"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
	validation "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/validation"
	serverrequest "github.com/kubeclipper/kubeclipper/pkg/server/request"
	"github.com/kubeclipper/kubeclipper/pkg/server/restplus"
)

const (
	agentUsernamePrefix = "system:kc-agent:"
	agentGroup          = "system:kc-agents"
	defaultWatchTimeout = 30 * time.Minute
)

type handler struct {
	store         operationv2.Store
	clusterReader cluster.OperatorReader
	logClient     *http.Client
}

func (h *handler) createOperation(req *restful.Request, resp *restful.Response) {
	op := &operations.Operation{}
	if err := req.ReadEntity(op); err != nil {
		writeError(resp, apierrors.NewBadRequest(err.Error()))
		return
	}
	if op.Spec.Timeout.Duration == 0 {
		op.Spec.Timeout.Duration = operations.DefaultOperationTimeout
	}
	if op.Spec.DesiredState == "" {
		op.Spec.DesiredState = operations.OperationDesiredStateActive
	}
	if errs := validation.ValidateOperation(op); len(errs) > 0 {
		writeError(resp, validation.InvalidOperation(op.Name, errs))
		return
	}
	if err := h.validateTargetRefs(req, op); err != nil {
		writeError(resp, err)
		return
	}
	created, err := h.store.CreateOperation(req.Request.Context(), op)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusCreated, created); err != nil {
		return
	}
}

func (h *handler) validateTargetRefs(req *restful.Request, op *operations.Operation) error {
	if op.Spec.TargetRef.Kind != corev1.KindCluster {
		return apierrors.NewBadRequest("v2 operations may target only Cluster objects")
	}
	clusterObj, err := h.clusterReader.GetClusterEx(req.Request.Context(), op.Spec.TargetRef.Name, "")
	if err != nil {
		return err
	}
	if clusterObj.UID != op.Spec.TargetRef.UID {
		return apierrors.NewConflict(corev1.Resource("clusters"), clusterObj.Name, fmt.Errorf("cluster UID does not match"))
	}
	nodes := make(map[string]operations.NodeReference)
	for stepIndex := range op.Spec.Steps {
		for _, node := range op.Spec.Steps[stepIndex].Targets {
			nodes[node.Name] = node
		}
	}
	for _, ref := range nodes {
		node, err := h.clusterReader.GetNodeEx(req.Request.Context(), ref.Name, "")
		if err != nil {
			return err
		}
		if node.UID != ref.UID {
			return apierrors.NewConflict(corev1.Resource("nodes"), node.Name, fmt.Errorf("node UID does not match"))
		}
	}
	return nil
}

func (h *handler) getOperation(req *restful.Request, resp *restful.Response) {
	op, err := h.store.GetOperation(req.Request.Context(), req.PathParameter("name"), req.QueryParameter("resourceVersion"))
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, op); err != nil {
		return
	}
}

func (h *handler) listOperations(req *restful.Request, resp *restful.Response) {
	options, err := parseListOptions(req)
	if err != nil {
		writeError(resp, err)
		return
	}
	if options.Watch {
		watcher, watchErr := h.store.WatchOperationsWithOptions(req.Request.Context(), &options)
		if watchErr != nil {
			writeError(resp, watchErr)
			return
		}
		restplus.ServeWatch(watcher, operations.SchemeGroupVersion.WithKind(operations.KindOperation), req, resp, watchTimeout(&options))
		return
	}
	list, err := h.store.ListOperationsWithOptions(req.Request.Context(), &options)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, list); err != nil {
		return
	}
}

func (h *handler) cancelOperation(req *restful.Request, resp *restful.Response) {
	preconditions, ok := readControlRequest(req, resp)
	if !ok {
		return
	}
	op, err := h.store.GetOperation(req.Request.Context(), req.PathParameter("name"), "")
	if err != nil {
		writeError(resp, err)
		return
	}
	if op.UID != preconditions.UID {
		writeError(resp, conflictOperation(op.Name, "operation UID does not match"))
		return
	}
	if op.Status.Phase != operations.OperationPending && op.Status.Phase != operations.OperationRunning {
		writeError(resp, apierrors.NewBadRequest("only Pending or Running operations may be canceled"))
		return
	}
	if op.Spec.DesiredState != operations.OperationDesiredStateActive {
		writeError(resp, conflictOperation(op.Name, "operation is already canceled"))
		return
	}
	updated, err := h.store.UpdateOperationControl(
		req.Request.Context(),
		op.Name,
		op.UID,
		preconditions.ResourceVersion,
		func(spec *operations.OperationSpec) error {
			spec.DesiredState = operations.OperationDesiredStateCancelled
			return nil
		},
	)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, updated); err != nil {
		return
	}
}

func (h *handler) retryOperation(req *restful.Request, resp *restful.Response) {
	preconditions, ok := readControlRequest(req, resp)
	if !ok {
		return
	}
	op, err := h.loadRetryableOperation(req, preconditions)
	if err != nil {
		writeError(resp, err)
		return
	}
	updated, err := h.store.UpdateOperationControl(
		req.Request.Context(),
		op.Name,
		op.UID,
		preconditions.ResourceVersion,
		func(spec *operations.OperationSpec) error {
			spec.DesiredState = operations.OperationDesiredStateActive
			spec.RetryGeneration++
			return nil
		},
	)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, updated); err != nil {
		return
	}
}

func (h *handler) loadRetryableOperation(
	req *restful.Request,
	preconditions operations.OperationControlRequest,
) (*operations.Operation, error) {
	op, err := h.store.GetOperation(req.Request.Context(), req.PathParameter("name"), "")
	if err != nil {
		return nil, err
	}
	if op.UID != preconditions.UID {
		return nil, conflictOperation(op.Name, "operation UID does not match")
	}
	if !retryablePhase(op.Status.Phase) {
		return nil, apierrors.NewBadRequest("only Failed, TimedOut or Canceled operations may be retried")
	}
	if err := h.ensureTasksTerminal(req.Request.Context(), op); err != nil {
		return nil, err
	}
	if err := h.ensureLatestOperation(req.Request.Context(), op); err != nil {
		return nil, err
	}
	return op, nil
}

func retryablePhase(phase operations.OperationPhase) bool {
	return phase == operations.OperationFailed || phase == operations.OperationTimedOut || phase == operations.OperationCancelled
}

func (h *handler) ensureTasksTerminal(ctx context.Context, op *operations.Operation) error {
	tasks, err := h.store.ListTasksByOperationUID(ctx, op.UID, "")
	if err != nil {
		return err
	}
	for i := range tasks.Items {
		if !tasks.Items[i].Status.Phase.IsTerminal() {
			return conflictOperation(op.Name, "operation still has active tasks")
		}
	}
	return nil
}

func (h *handler) ensureLatestOperation(ctx context.Context, op *operations.Operation) error {
	operationList, err := h.store.ListOperations(ctx, op.Spec.TargetRef.UID, "")
	if err != nil {
		return err
	}
	sort.Slice(operationList.Items, func(i, j int) bool { return operationLess(&operationList.Items[i], &operationList.Items[j]) })
	if len(operationList.Items) == 0 || operationList.Items[len(operationList.Items)-1].UID != op.UID {
		return conflictOperation(op.Name, "only the latest operation for a target may be retried")
	}
	return nil
}

func operationLess(left, right *operations.Operation) bool {
	if !left.CreationTimestamp.Equal(&right.CreationTimestamp) {
		return left.CreationTimestamp.Before(&right.CreationTimestamp)
	}
	leftRevision, leftErr := strconv.ParseUint(left.ResourceVersion, 10, 64)
	rightRevision, rightErr := strconv.ParseUint(right.ResourceVersion, 10, 64)
	if leftErr == nil && rightErr == nil && leftRevision != rightRevision {
		return leftRevision < rightRevision
	}
	return string(left.UID) < string(right.UID)
}

func (h *handler) listTasks(req *restful.Request, resp *restful.Response) {
	options, err := parseListOptions(req)
	if err != nil {
		writeError(resp, err)
		return
	}
	agentID, isAgent := agentIDFromRequest(req)
	if options.Watch {
		watcher, watchErr := h.store.WatchTasksWithOptions(req.Request.Context(), conditionalAgentID(agentID, isAgent), &options)
		if watchErr != nil {
			writeError(resp, watchErr)
			return
		}
		restplus.ServeWatch(watcher, operations.SchemeGroupVersion.WithKind(operations.KindOperationTask), req, resp, watchTimeout(&options))
		return
	}
	list, err := h.store.ListTasksWithOptions(req.Request.Context(), conditionalAgentID(agentID, isAgent), &options)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, list); err != nil {
		return
	}
}

func (h *handler) getTask(req *restful.Request, resp *restful.Response) {
	task, err := h.store.GetTask(req.Request.Context(), req.PathParameter("name"), req.QueryParameter("resourceVersion"))
	if err != nil {
		writeError(resp, err)
		return
	}
	if agentID, isAgent := agentIDFromRequest(req); isAgent && task.Spec.NodeRef.Name != agentID {
		writeError(
			resp,
			apierrors.NewForbidden(operations.Resource(operations.ResourceTasks), task.Name, fmt.Errorf("task belongs to another agent")),
		)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, task); err != nil {
		return
	}
}

func (h *handler) getTaskLogs(req *restful.Request, resp *restful.Response) {
	task, err := h.store.GetTask(req.Request.Context(), req.PathParameter("name"), "")
	if err != nil {
		writeError(resp, err)
		return
	}
	node, err := h.clusterReader.GetNodeEx(req.Request.Context(), task.Spec.NodeRef.Name, "")
	if err != nil {
		writeError(resp, err)
		return
	}
	if node.UID != task.Spec.NodeRef.UID || node.Status.Ipv4DefaultIP == "" {
		writeError(
			resp,
			apierrors.NewConflict(
				operations.Resource(operations.ResourceTasks),
				task.Name,
				fmt.Errorf("task Node identity or management IP is unavailable"),
			),
		)
		return
	}
	query := url.Values{}
	query.Set("offset", req.QueryParameter("offset"))
	query.Set("limit", req.QueryParameter("limit"))
	endpoint, err := taskLogEndpoint(node, task, query)
	if err != nil {
		writeError(resp, apierrors.NewConflict(operations.Resource(operations.ResourceTasks), task.Name, err))
		return
	}
	request, err := http.NewRequestWithContext(req.Request.Context(), http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		writeError(resp, apierrors.NewInternalError(err))
		return
	}
	// Agents advertise their management IP in Node status, while their serving
	// certificate is identified by the stable AgentID. Keep hostname
	// verification enabled and use the Node name as the TLS ServerName.
	logClient := *h.logClient
	if transport, ok := h.logClient.Transport.(*http.Transport); ok {
		transport = transport.Clone()
		if transport.TLSClientConfig != nil {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
			transport.TLSClientConfig.ServerName = task.Spec.NodeRef.Name
		}
		logClient.Transport = transport
	}
	upstream, err := logClient.Do(request)
	if err != nil {
		writeError(resp, apierrors.NewServiceUnavailable("agent Task log endpoint is unavailable"))
		return
	}
	defer upstream.Body.Close()
	body, err := io.ReadAll(io.LimitReader(upstream.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		writeError(resp, apierrors.NewInternalError(fmt.Errorf("read agent Task log response failed or exceeded 1 MiB")))
		return
	}
	resp.Header().Set("Content-Type", upstream.Header.Get("Content-Type"))
	resp.WriteHeader(upstream.StatusCode)
	if _, err := resp.Write(body); err != nil {
		return
	}
}

func taskLogEndpoint(node *corev1.Node, task *operations.OperationTask, query url.Values) (string, error) {
	port := node.Status.AgentLogPort
	if port == 0 {
		port = constatns.DefaultAgentLogPort
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("agent log port %d is invalid", port)
	}
	return fmt.Sprintf(
		"https://%s/v1/tasks/%s/logs?%s",
		net.JoinHostPort(node.Status.Ipv4DefaultIP, strconv.Itoa(int(port))),
		url.PathEscape(string(task.UID)),
		query.Encode(),
	), nil
}

func newLogClient(caFile, certFile, keyFile string) (*http.Client, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caData, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("agent log CA contains no certificates")
	}
	return &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{certificate},
	}}}, nil
}

func (h *handler) updateTaskStatus(req *restful.Request, resp *restful.Response) {
	agentID, ok := agentIDFromRequest(req)
	if !ok {
		writeError(
			resp,
			apierrors.NewForbidden(
				operations.Resource(operations.ResourceTasks),
				req.PathParameter("name"),
				fmt.Errorf("agent certificate is required"),
			),
		)
		return
	}
	incoming := &operations.OperationTask{}
	if err := req.ReadEntity(incoming); err != nil {
		writeError(resp, apierrors.NewBadRequest(err.Error()))
		return
	}
	if incoming.Name != "" && incoming.Name != req.PathParameter("name") {
		writeError(resp, apierrors.NewBadRequest("task name does not match URL"))
		return
	}
	current, err := h.store.GetTask(req.Request.Context(), req.PathParameter("name"), "")
	if err != nil {
		writeError(resp, err)
		return
	}
	if current.Spec.NodeRef.Name != agentID {
		writeError(
			resp,
			apierrors.NewForbidden(
				operations.Resource(operations.ResourceTasks),
				current.Name,
				fmt.Errorf("task belongs to another agent"),
			),
		)
		return
	}
	if incoming.UID == "" || incoming.ResourceVersion == "" {
		writeError(resp, apierrors.NewBadRequest("task UID and resourceVersion are required"))
		return
	}
	updated, err := h.store.UpdateTaskStatus(req.Request.Context(), current.Name, incoming.UID, incoming.ResourceVersion, incoming.Status)
	if err != nil {
		writeError(resp, err)
		return
	}
	if err := resp.WriteHeaderAndEntity(http.StatusOK, updated); err != nil {
		return
	}
}

func readControlRequest(req *restful.Request, resp *restful.Response) (operations.OperationControlRequest, bool) {
	request := operations.OperationControlRequest{}
	if err := req.ReadEntity(&request); err != nil {
		writeError(resp, apierrors.NewBadRequest(err.Error()))
		return request, false
	}
	if request.UID == "" || request.ResourceVersion == "" {
		writeError(resp, apierrors.NewBadRequest("uid and resourceVersion are required"))
		return request, false
	}
	return request, true
}

func parseListOptions(req *restful.Request) (metav1.ListOptions, error) {
	values := req.Request.URL.Query()
	options := metav1.ListOptions{
		LabelSelector: values.Get("labelSelector"), FieldSelector: values.Get("fieldSelector"),
		ResourceVersion: values.Get("resourceVersion"), Continue: values.Get("continue"),
		ResourceVersionMatch: metav1.ResourceVersionMatch(values.Get("resourceVersionMatch")),
	}
	var err error
	if value := values.Get("watch"); value != "" {
		options.Watch, err = strconv.ParseBool(value)
		if err != nil {
			return options, apierrors.NewBadRequest("watch must be a boolean")
		}
	}
	if value := values.Get("allowWatchBookmarks"); value != "" {
		options.AllowWatchBookmarks, err = strconv.ParseBool(value)
		if err != nil {
			return options, apierrors.NewBadRequest("allowWatchBookmarks must be a boolean")
		}
	}
	if value := values.Get("limit"); value != "" {
		options.Limit, err = strconv.ParseInt(value, 10, 64)
		if err != nil || options.Limit < 0 {
			return options, apierrors.NewBadRequest("limit must be a non-negative integer")
		}
	}
	if value := values.Get("timeoutSeconds"); value != "" {
		parsed, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return options, apierrors.NewBadRequest("timeoutSeconds must be a positive integer")
		}
		options.TimeoutSeconds = &parsed
	}
	return options, nil
}

func watchTimeout(options *metav1.ListOptions) time.Duration {
	if options != nil && options.TimeoutSeconds != nil {
		return time.Duration(*options.TimeoutSeconds) * time.Second
	}
	return defaultWatchTimeout
}

func agentIDFromRequest(req *restful.Request) (string, bool) {
	user, ok := serverrequest.UserFrom(req.Request.Context())
	if !ok || !hasGroup(user, agentGroup) || !strings.HasPrefix(user.GetName(), agentUsernamePrefix) {
		return "", false
	}
	id := strings.TrimPrefix(user.GetName(), agentUsernamePrefix)
	return id, id != ""
}

func hasGroup(user userapi.Info, wanted string) bool {
	return slices.Contains(user.GetGroups(), wanted)
}

func conditionalAgentID(id string, ok bool) string {
	if ok {
		return id
	}
	return ""
}

func conflictOperation(name, message string) error {
	return apierrors.NewConflict(operations.Resource(operations.ResourceOperations), name, fmt.Errorf("%s", message))
}

func writeError(resp *restful.Response, err error) {
	status := apierrors.NewInternalError(err).ErrStatus
	if apiStatus, ok := err.(apierrors.APIStatus); ok {
		status = apiStatus.Status()
	}
	if status.Code == 0 {
		status.Code = http.StatusInternalServerError
	}
	if err := resp.WriteHeaderAndEntity(int(status.Code), &status); err != nil {
		return
	}
}
