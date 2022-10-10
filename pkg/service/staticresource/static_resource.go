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
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/simple/staticserver"
)

var _ service.Interface = (*Service)(nil)

type Service struct {
	server *http.Server
	path   string
}

func NewService(opts *staticserver.Options) (service.Interface, error) {
	httpSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", opts.BindAddress, opts.InsecurePort),
	}
	if opts.SecurePort != 0 {
		certificate, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSPrivateKey)
		if err != nil {
			return nil, err
		}
		httpSrv.TLSConfig.Certificates = []tls.Certificate{certificate}
		httpSrv.Addr = fmt.Sprintf("%s:%d", opts.BindAddress, opts.SecurePort)
	}
	return &Service{
		server: httpSrv,
		path:   opts.Path,
	}, nil
}

func (s *Service) PrepareRun(stopCh <-chan struct{}) error {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return os.MkdirAll(s.path, os.ModeDir|0755)
	}
	s.server.Handler = http.StripPrefix("/", http.FileServer(http.Dir(s.path)))
	return nil
}

func (s *Service) Run(stopCh <-chan struct{}) error {
	logger.Info("Static resource server start", zap.String("addr", s.server.Addr), zap.String("path", s.path))
	go func() {
		<-stopCh
		_ = s.server.Shutdown(context.TODO())
	}()
	go func() {
		var err error
		if s.server.TLSConfig != nil {
			err = s.server.ListenAndServeTLS("", "")
		} else {
			err = s.server.ListenAndServe()
		}
		logger.Error("static resource server exit", zap.Error(err))
	}()

	return nil
}

func (s *Service) Close() {
	s.server.Close()
}
