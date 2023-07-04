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

package auditing

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	auditoptions "github.com/kubeclipper/kubeclipper/pkg/auditing/option"

	"k8s.io/apimachinery/pkg/util/sets"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/models/platform"

	"github.com/google/uuid"
	authnv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"k8s.io/apiserver/pkg/apis/audit"

	"github.com/kubeclipper/kubeclipper/pkg/server/request"
)

type Interface interface {
	Enabled() bool
	AddBackend(backend Backend)
	LogRequestObject(req *http.Request, info *request.Info) *audit.Event
	LogResponseObject(e *audit.Event, resp *ResponseCapture)
}

type Backend interface {
	SendEvent(e audit.Event)
}

type ResponseCapture struct {
	http.ResponseWriter
	wroteHeader bool
	status      int
	body        *bytes.Buffer
}

func NewResponseCapture(w http.ResponseWriter) *ResponseCapture {
	return &ResponseCapture{
		ResponseWriter: w,
		wroteHeader:    false,
		body:           new(bytes.Buffer),
	}
}

func (c *ResponseCapture) Header() http.Header {
	return c.ResponseWriter.Header()
}

func (c *ResponseCapture) Write(data []byte) (int, error) {

	c.WriteHeader(http.StatusOK)
	c.body.Write(data)
	return c.ResponseWriter.Write(data)
}

func (c *ResponseCapture) WriteHeader(statusCode int) {
	if !c.wroteHeader {
		c.status = statusCode
		c.ResponseWriter.WriteHeader(statusCode)
		c.wroteHeader = true
	}
}

func (c *ResponseCapture) Bytes() []byte {
	return c.body.Bytes()
}

func (c *ResponseCapture) StatusCode() int {
	return c.status
}

// Hijack implements the http.Hijacker interface.  This expands
// the Response to fulfill http.Hijacker if the underlying
// http.ResponseWriter supports it.
func (c *ResponseCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := c.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter doesn't support Hijacker interface")
	}
	return hijacker.Hijack()
}

func (c *ResponseCapture) Flush() {
	c.ResponseWriter.(http.Flusher).Flush()
}

// CloseNotify is part of http.CloseNotifier interface
func (c *ResponseCapture) CloseNotify() <-chan bool {
	return c.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

type Object struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

var _ Interface = (*auditing)(nil)

func NewAuditing(options *auditoptions.AuditOptions) Interface {
	return &auditing{
		backends:     nil,
		auditOptions: options,
	}
}

type auditing struct {
	backends     []Backend
	auditOptions *auditoptions.AuditOptions
}

func (a *auditing) AddBackend(backend Backend) {
	if backend != nil {
		a.backends = append(a.backends, backend)
	}
}

func (a *auditing) Enabled() bool {
	return !a.auditOptions.AuditLevel.Less(audit.LevelMetadata)
}

func (a *auditing) LogRequestObject(req *http.Request, info *request.Info) *audit.Event {
	if len(req.URL.Query()["dryRun"]) != 0 {
		logger.Debug("ignore dryRun request", zap.String("url", req.URL.Path))
		return nil
	}
	e := &audit.Event{
		RequestURI: info.Path,
		Verb:       info.Verb,
		// Level:                    a.level,
		Level:                    a.auditOptions.AuditLevel,
		AuditID:                  types.UID(uuid.New().String()),
		Stage:                    audit.StageResponseComplete,
		ImpersonatedUser:         nil,
		UserAgent:                req.UserAgent(),
		RequestReceivedTimestamp: metav1.NowMicro(),
		Annotations:              nil,
		ObjectRef: &audit.ObjectReference{
			Resource: info.Resource,
			Name:     info.Name,
			UID:      "",
			// APIGroup:    info.APIGroup,
			// APIVersion:  info.APIVersion,
			// Subresource: info.Subresource,
		},
	}

	ips := make([]string, 1)
	ips[0] = RemoteIP(req)
	e.SourceIPs = ips

	user, ok := request.UserFrom(req.Context())
	if ok {
		e.User.Username = user.GetName()
		e.User.UID = user.GetUID()
		e.User.Groups = user.GetGroups()

		e.User.Extra = make(map[string]authnv1.ExtraValue)
		for k, v := range user.GetExtra() {
			e.User.Extra[k] = v
		}
	}

	if (e.Level.GreaterOrEqual(audit.LevelRequest) || e.Verb == "create") && req.ContentLength > 0 && req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Error("read request body failed", zap.Error(err))
			return e
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(body))

		if e.Level.GreaterOrEqual(audit.LevelRequest) {
			if info.Path == "/oauth/login" {
				obj := &LoginRequest{}
				if err := json.Unmarshal(body, obj); err == nil {
					e.User.Username = obj.Username
				}
			} else {
				e.RequestObject = &runtime.Unknown{Raw: body}
			}
		}

		if info.Verb == "create" {
			obj := &Object{}
			if err := json.Unmarshal(body, obj); err == nil {
				e.ObjectRef.Name = obj.Name
			}
		}
	}
	return e
}

func (a *auditing) LogResponseObject(e *audit.Event, resp *ResponseCapture) {
	e.StageTimestamp = metav1.NowMicro()
	e.ResponseStatus = &metav1.Status{Code: int32(resp.StatusCode())}
	if e.Level.GreaterOrEqual(audit.LevelRequestResponse) {
		e.ResponseObject = &runtime.Unknown{Raw: resp.Bytes()}
	}
	a.sendEvent(e)
}

func (a *auditing) sendEvent(e *audit.Event) {
	for _, backend := range a.backends {
		backend.SendEvent(*e)
	}
}

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
	XClientIP     = "x-client-ip"
)

func RemoteIP(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XClientIP); ip != "" {
		remoteAddr = ip
	} else if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

var _ Backend = (*ConsoleBackend)(nil)

type ConsoleBackend struct {
}

func (c ConsoleBackend) SendEvent(e audit.Event) {
	logger.Info("audit", zap.Any("event", e))
}

type DatabaseBackend struct {
	eventOp     platform.EventWriter
	eventCh     chan *audit.Event
	stopCh      <-chan struct{}
	ignoreVerbs sets.Set[string]
}

func NewDatabaseBackend(operator platform.EventWriter, stopCh <-chan struct{}) Backend {
	b := &DatabaseBackend{
		eventOp:     operator,
		eventCh:     make(chan *audit.Event, 1000),
		stopCh:      stopCh,
		ignoreVerbs: sets.New("get", "list", "watch"),
	}
	go b.worker()
	return b
}

func (c *DatabaseBackend) SendEvent(e audit.Event) {
	if c.ignoreVerbs.Has(e.Verb) {
		return
	}
	select {
	case c.eventCh <- &e:
		return
	case <-time.After(time.Second):
		logger.Warn("send event to database backend timeout", zap.String("audit_id", string(e.AuditID)))
		break
	}
}

func (c *DatabaseBackend) worker() {
	for {
		select {
		case <-c.stopCh:
			return
		case event, ok := <-c.eventCh:
			if !ok {
				// End of results.
				return
			}
			go c.CreateEvent(event)
		}
	}
}

func (c *DatabaseBackend) CreateEvent(e *audit.Event) {
	internalEv := v1.Event{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Event",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "audit-",
		},
		AuditID:                  string(e.AuditID),
		RequestURI:               e.RequestURI,
		UserID:                   e.User.UID,
		Username:                 e.User.Username,
		SourceIP:                 e.SourceIPs[0],
		UserAgent:                e.UserAgent,
		Verb:                     e.Verb,
		Success:                  true,
		RequestReceivedTimestamp: e.RequestReceivedTimestamp,
		StageTimestamp:           e.StageTimestamp,
		Resource:                 e.ObjectRef.Resource,
		ResourceName:             e.ObjectRef.Name,
	}
	if e.RequestURI == "/oauth/login" {
		internalEv.Type = "login"
	} else if e.RequestURI == "/oauth/logout" {
		internalEv.Type = "logout"
	}
	if e.ResponseStatus != nil && e.ResponseStatus.Code >= http.StatusBadRequest {
		internalEv.Success = false
	}
	_, err := c.eventOp.CreateEvent(context.TODO(), &internalEv)
	if err != nil {
		logger.Error("create audit event error", zap.Error(err))
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
