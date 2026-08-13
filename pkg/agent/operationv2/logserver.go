/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package operationv2

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
)

const (
	DefaultAgentLogAddress      = ":10260"
	DefaultServerClientIdentity = "system:kc-server"
	maxLogReadSize              = 1 << 20
)

type TaskLogReader interface {
	GetStepLogContent(opID, stepID string, offset int64, length int) (content []byte, deliverySize int64, logSize int64, err error)
}

type LogHandler struct {
	Logs                     TaskLogReader
	ExpectedClientCommonName string
}

func (h *LogHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if !validLogClient(request, h.expectedClientCommonName()) {
		http.Error(response, "verified kc-server client certificate is required", http.StatusUnauthorized)
		return
	}
	taskUID, ok := taskUIDFromLogPath(request.URL.Path)
	if !ok {
		http.NotFound(response, request)
		return
	}
	offset, err := parseNonNegativeInt64(request.URL.Query().Get("offset"), 0)
	if err != nil {
		http.Error(response, "invalid offset", http.StatusBadRequest)
		return
	}
	limit, err := parseLogLimit(request.URL.Query().Get("limit"))
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	content, delivered, size, err := h.Logs.GetStepLogContent(taskUID, "task", offset, limit)
	if err != nil {
		if os.IsNotExist(err) {
			content, delivered, size = nil, 0, 0
		} else {
			http.Error(response, "read task log failed", http.StatusInternalServerError)
			return
		}
	}
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(oplog.LogContentResponse{
		Content:      string(content),
		LogSize:      size,
		DeliverySize: delivered,
	})
}

func (h *LogHandler) expectedClientCommonName() string {
	if h.ExpectedClientCommonName == "" {
		return DefaultServerClientIdentity
	}
	return h.ExpectedClientCommonName
}

func validLogClient(request *http.Request, expectedCommonName string) bool {
	if request.TLS == nil || len(request.TLS.PeerCertificates) == 0 || len(request.TLS.VerifiedChains) == 0 {
		return false
	}
	return request.TLS.PeerCertificates[0].Subject.CommonName == expectedCommonName
}

func taskUIDFromLogPath(requestPath string) (string, bool) {
	const prefix = "/v1/tasks/"
	if !strings.HasPrefix(requestPath, prefix) || !strings.HasSuffix(requestPath, "/logs") {
		return "", false
	}
	uid := strings.TrimSuffix(strings.TrimPrefix(requestPath, prefix), "/logs")
	if uid == "" || strings.Contains(uid, "/") || uid == "." || uid == ".." {
		return "", false
	}
	return uid, true
}

func parseNonNegativeInt64(value string, defaultValue int64) (int64, error) {
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("value must be a non-negative integer")
	}
	return parsed, nil
}

func parseLogLimit(value string) (int, error) {
	parsed, err := parseNonNegativeInt64(value, maxLogReadSize)
	if err != nil || parsed == 0 || parsed > maxLogReadSize {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxLogReadSize)
	}
	return int(parsed), nil
}

type LogServerOptions struct {
	Address                  string
	TLSCertFile              string
	TLSKeyFile               string
	ClientCAFile             string
	ExpectedClientCommonName string
	Logs                     TaskLogReader
}

type LogServer struct {
	server   *http.Server
	listener net.Listener
	close    sync.Once
}

func NewLogServer(opts LogServerOptions) (*LogServer, error) {
	if opts.Address == "" {
		opts.Address = DefaultAgentLogAddress
	}
	if opts.Logs == nil || opts.TLSCertFile == "" || opts.TLSKeyFile == "" || opts.ClientCAFile == "" {
		return nil, fmt.Errorf("Task logs, TLS certificate, key, and client CA are required")
	}
	certificate, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load agent serving certificate: %w", err)
	}
	caPEM, err := os.ReadFile(opts.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read client CA: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("client CA contains no certificates")
	}
	handler := &LogHandler{Logs: opts.Logs, ExpectedClientCommonName: opts.ExpectedClientCommonName}
	return &LogServer{server: &http.Server{
		Addr:              opts.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{certificate},
			ClientCAs:    clientCAs,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		},
	}}, nil
}

func (s *LogServer) PrepareRun(<-chan struct{}) error {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	s.listener = tls.NewListener(listener, s.server.TLSConfig)
	return nil
}

func (s *LogServer) Run(stopCh <-chan struct{}) error {
	if s.listener == nil {
		return fmt.Errorf("log server is not prepared")
	}
	go func() {
		<-stopCh
		s.Close()
	}()
	go func() {
		if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("agent Task log server stopped: %v", err)
		}
	}()
	return nil
}

func (s *LogServer) Close() {
	s.close.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	})
}
