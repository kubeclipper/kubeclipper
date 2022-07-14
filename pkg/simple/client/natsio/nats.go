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

package natsio

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/google/uuid"
	natServer "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	certutil "k8s.io/client-go/util/cert"
)

var _ Interface = (*Client)(nil)

type Client struct {
	serverRunning    bool
	serverRunningMux sync.Mutex
	server           *natServer.Server
	conn             *nats.Conn
	serverOptions    *natServer.Options
	clientOptions    []nats.Option
	url              string
}

func NewNats(opts *NatsOptions) Interface {
	s := &Client{
		conn: nil,
		url:  opts.GetConnectionString(),
	}
	s.setupConnOptions(opts)
	s.setupServerOptions(opts)

	return s
}

func (c *Client) Close() {
	if c.server != nil {
		c.server.Shutdown()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) SetDisconnectErrHandler(handler nats.ConnErrHandler) {
	c.clientOptions = append(c.clientOptions, nats.DisconnectErrHandler(handler))
}

func (c *Client) SetReconnectHandler(handler nats.ConnHandler) {
	c.clientOptions = append(c.clientOptions, nats.ReconnectHandler(handler))
}

func (c *Client) SetErrorHandler(handler nats.ErrHandler) {
	c.clientOptions = append(c.clientOptions, nats.ErrorHandler(handler))
}

func (c *Client) SetClosedHandler(handler nats.ConnHandler) {
	c.clientOptions = append(c.clientOptions, nats.ClosedHandler(handler))
}

func (c *Client) setupConnOptions(opts *NatsOptions) {
	c.clientOptions = []nats.Option{
		nats.Name(fmt.Sprintf("server.%s", uuid.New().String())),
		nats.ReconnectWait(opts.Client.ReconnectInterval),
		nats.MaxReconnects(opts.Client.MaxReconnect),
		nats.PingInterval(opts.Client.PingInterval),
		nats.MaxPingsOutstanding(opts.Client.MaxPingsOut),
	}
	c.clientOptions = append(c.clientOptions, nats.UserInfo(opts.Auth.UserName, opts.Auth.Password))
	if opts.Client.TLSCaPath != "" {
		c.clientOptions = append(c.clientOptions, nats.ClientCert(opts.Client.TLSCertPath, opts.Client.TLSKeyPath))
		c.clientOptions = append(c.clientOptions, nats.RootCAs(opts.Client.TLSCaPath))
	}
}

func (c *Client) setTLSConfig(opts *NatsOptions) *tls.Config {
	tlsConfig := &tls.Config{}
	ca, err := certutil.CertsFromFile(opts.Client.TLSCaPath)
	if err != nil {
		return nil
	}
	serverCert, err := tls.LoadX509KeyPair(opts.Server.TLSCertPath, opts.Server.TLSKeyPath)
	if err != nil {
		return nil
	}
	clientCA := x509.NewCertPool()
	clientCA.AddCert(ca[0])
	rootCA := x509.NewCertPool()
	rootCA.AddCert(ca[0])
	tlsConfig.RootCAs = rootCA
	tlsConfig.ClientCAs = clientCA
	tlsConfig.Certificates = []tls.Certificate{serverCert}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	tlsConfig.InsecureSkipVerify = false
	return tlsConfig
}

func (c *Client) setupServerOptions(opts *NatsOptions) {
	c.serverOptions = &natServer.Options{
		Debug:    false,
		Host:     opts.Server.Host,
		Port:     opts.Server.Port,
		Username: opts.Auth.UserName,
		Password: opts.Auth.Password,
		Cluster: natServer.ClusterOpts{
			Host: opts.Server.Cluster.Host,
			Port: opts.Server.Cluster.Port,
		},
		// we use one of master ip as the leader
		// all of masters have to set message-queue-leader to this master ip will do the trick e.g. message-queue-leader: 10.0.0.100
		// when use message queue cluster client can set multiple server addresses split by command
		// e.g. 10.0.0.100:2222,10.0.0.101:2222,10.0.0.102:2222
		// won`t abort program even the ip:port not open yet
		Routes:    natServer.RoutesFromStr(fmt.Sprintf("nats://%s", opts.Server.Cluster.LeaderHost)),
		TLSCaCert: opts.Server.TLSCaPath,
		TLSCert:   opts.Server.TLSCertPath,
		TLSKey:    opts.Server.TLSKeyPath,
	}

	if opts.Server.TLSCaPath != "" {
		c.serverOptions.TLSConfig = c.setTLSConfig(opts)
		c.serverOptions.TLSVerify = true
	}
}

func (c *Client) RunServer(stopCh <-chan struct{}) error {
	c.serverRunningMux.Lock()
	defer c.serverRunningMux.Unlock()
	if c.serverRunning {
		return nil
	}
	var err error
	c.server, err = natServer.NewServer(c.serverOptions)
	if err != nil {
		return err
	}
	go c.server.Start()
	go func() {
		<-stopCh
		c.server.Shutdown()
	}()
	c.serverRunning = true
	return nil
}

func (c *Client) InitConn(stopCh <-chan struct{}) error {
	var err error
	c.conn, err = nats.Connect(c.url, c.clientOptions...)
	if err != nil {
		return err
	}
	go func() {
		<-stopCh
		c.conn.Close()
	}()
	return nil
}

func (c *Client) Publish(msg *Msg) error {
	return c.conn.Publish(msg.Subject, msg.Data)
}

func (c *Client) Subscribe(subj string, handler nats.MsgHandler) error {
	_, err := c.conn.Subscribe(subj, handler)
	return err
}

func (c *Client) QueueSubscribe(subj string, queue string, handler nats.MsgHandler) error {
	_, err := c.conn.QueueSubscribe(subj, queue, handler)
	return err
}

func (c *Client) Request(msg *Msg, timeoutHandler TimeoutHandler) ([]byte, error) {
	resp, err := c.request(msg, timeoutHandler)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) RequestWithContext(ctx context.Context, msg *Msg) ([]byte, error) {
	resp, err := c.conn.RequestWithContext(ctx, msg.Subject, msg.Data)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) RequestAsync(msg *Msg, handler ReplyHandler, timeoutHandler TimeoutHandler) error {
	resp, err := c.request(msg, timeoutHandler)
	if err != nil {
		return err
	}
	if handler != nil {
		return handler(resp)
	}
	return nil
}

func (c *Client) request(msg *Msg, timeoutHandler TimeoutHandler) (*nats.Msg, error) {
	resp, err := c.conn.Request(msg.Subject, msg.Data, msg.Timeout)
	if err != nil {
		if err == nats.ErrTimeout && timeoutHandler != nil {
			//TODO: wrapper error
			_ = timeoutHandler(msg)
			return nil, err
		}
		return nil, err
	}
	return resp, err
}
