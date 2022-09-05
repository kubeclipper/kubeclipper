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

package proxy

import (
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/cmd/kubeclipper-proxy/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/proxy/config"
)

type Server struct {
	Config *config.Config
	stopCh <-chan struct{}
}

func NewServer(opt *options.ProxyOptions, stopCh <-chan struct{}) (*Server, error) {
	server := &Server{
		Config: opt.Config,
		stopCh: stopCh,
	}
	return server, nil
}

func (s *Server) PrepareRun() error {

	return nil
}

func (s *Server) Run() error {
	return s.StartProxy()
}

func (s *Server) StartProxy() error {
	err := s.proxyServer()
	if err != nil {
		return errors.WithMessage(err, "proxy server")
	}
	err = s.proxyAgent()
	if err != nil {
		return errors.WithMessage(err, "proxy agent")
	}
	<-s.stopCh
	return nil
}

func (s *Server) proxyServer() error {
	if s.Config.ServerIP == "" {
		logger.Warn("no serverip specified")
		return nil
	}
	err := startProxy(config.Tunnel{
		LocalPort:     s.Config.DefaultMQPort,
		RemoteAddress: fmt.Sprintf("%s:%v", s.Config.ServerIP, s.Config.DefaultMQPort),
	})
	if err != nil {
		return err
	}
	err = startProxy(config.Tunnel{
		LocalPort:     s.Config.DefaultStaticServerPort,
		RemoteAddress: fmt.Sprintf("%s:%v", s.Config.ServerIP, s.Config.DefaultStaticServerPort),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) proxyAgent() error {
	for _, c := range s.Config.Config {
		for _, t := range c.Tunnels {
			err := startProxy(t)
			if err != nil {
				return errors.WithMessagef(err, "[start tunnel]: %v <---> %s", t.LocalPort, t.RemoteAddress)
			}
		}
	}
	return nil
}

func startProxy(t config.Tunnel) error {
	logger.Infof("[start tunnel]: %v <---> %s", t.LocalPort, t.RemoteAddress)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", t.LocalPort))
	if err != nil {
		return errors.WithMessagef(err, "[start tunnel]: %v <---> %s,listen port %v", t.LocalPort, t.RemoteAddress, t.LocalPort)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Errorf("ERROR: failed to accept listener: %v", err)
			}
			logger.Infof("Accepted connection from %v\n", conn.RemoteAddr().String())
			go forward(conn, t.RemoteAddress)
		}
	}()
	return nil
}

func forward(conn net.Conn, remoteAddr string) {
	client, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Errorf("dail %s err:%v", remoteAddr, err)
		return
	}
	logger.Infof("Forwarding from %v to %v\n", conn.LocalAddr(), client.RemoteAddr())

	go func() {
		defer func() {
			client.Close()
		}()
		_, _ = io.Copy(client, conn)
	}()
	go func() {
		defer func() {
			conn.Close()
		}()
		_, _ = io.Copy(conn, client)
	}()
}
