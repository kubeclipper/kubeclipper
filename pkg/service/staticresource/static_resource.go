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

package staticresource

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/simple/staticserver"
)

var _ service.Interface = (*Service)(nil)

const resourceDirectoryMode = os.FileMode(0755)

type Service struct {
	server  *http.Server
	path    string
	secure  bool
	running atomic.Bool
}

func NewService(opts *staticserver.Options) (*Service, error) {
	httpSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", opts.BindAddress, opts.InsecurePort),
	}
	if opts.SecurePort != 0 {
		certificate, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSPrivateKey)
		if err != nil {
			return nil, err
		}
		httpSrv.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{certificate},
		}
		httpSrv.Addr = fmt.Sprintf("%s:%d", opts.BindAddress, opts.SecurePort)
	}
	return &Service{
		server: httpSrv,
		path:   opts.Path,
		secure: opts.SecurePort != 0,
	}, nil
}

func (s *Service) PrepareRun(stopCh <-chan struct{}) error {
	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			if mkdirErr := os.MkdirAll(s.path, os.ModeDir|resourceDirectoryMode); mkdirErr != nil {
				return mkdirErr
			}
		} else {
			return err
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(s.path))))
	s.server.Handler = mux
	return nil
}

func (s *Service) Run(stopCh <-chan struct{}) error {
	logger.Info("Static resource server start", zap.String("addr", s.server.Addr), zap.String("path", s.path))
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", s.server.Addr)
	if err != nil {
		return err
	}
	s.server.Addr = listener.Addr().String()
	s.running.Store(true)
	go func() {
		<-stopCh
		_ = s.server.Shutdown(context.TODO())
	}()
	go func() {
		var serveErr error
		if s.secure {
			serveErr = s.server.ServeTLS(listener, "", "")
		} else {
			serveErr = s.server.Serve(listener)
		}
		s.running.Store(false)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Error("static resource server exit", zap.Error(serveErr))
		}
	}()

	return nil
}

func (s *Service) Close() {
	s.running.Store(false)
	_ = s.server.Close()
}

func (s *Service) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	if !s.running.Load() {
		http.Error(writer, "static resource server is not running", http.StatusServiceUnavailable)
		return
	}
	info, err := os.Stat(s.path)
	if err != nil || !info.IsDir() {
		http.Error(writer, "static resource directory is unavailable", http.StatusServiceUnavailable)
		return
	}
	dir, err := os.Open(s.path)
	if err != nil {
		http.Error(writer, "static resource directory is unreadable", http.StatusServiceUnavailable)
		return
	}
	_ = dir.Close()
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := writer.Write([]byte("ok")); err != nil {
		logger.Error("write static resource health response", zap.Error(err))
	}
}

func (s *Service) Health(ctx context.Context) error {
	host, port, err := net.SplitHostPort(s.server.Addr)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	scheme := "http"
	transport := http.DefaultTransport
	if s.secure {
		scheme = "https"
		// The request only reaches this process through its listener address, which may not match the certificate host.
		//nolint:gosec // TLS authenticity is established by the in-process listener, not a remote endpoint.
		transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s://%s/healthz", scheme, net.JoinHostPort(host, port)), http.NoBody)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("static resource health endpoint returned %s", response.Status)
	}
	return nil
}
