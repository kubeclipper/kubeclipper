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

//go:generate mockgen -destination mock/mock_nats.go -source interface.go Interface

package natsio

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
)

type Msg struct {
	Subject string
	From    string
	To      string
	Step    string
	Timeout time.Duration
	Data    []byte
}

type ReplyHandler func(msg *nats.Msg) error
type TimeoutHandler func(msg *Msg) error

type Interface interface {
	SetDisconnectErrHandler(handler nats.ConnErrHandler)
	SetReconnectHandler(handler nats.ConnHandler)
	SetErrorHandler(handler nats.ErrHandler)
	SetClosedHandler(handler nats.ConnHandler)
	RunServer(stopCh <-chan struct{}) error
	InitConn(stopCh <-chan struct{}) error
	Publish(msg *Msg) error
	Subscribe(subj string, handler nats.MsgHandler) error
	QueueSubscribe(subj string, queue string, handler nats.MsgHandler) error
	Request(msg *Msg, timeoutHandler TimeoutHandler) ([]byte, error)
	RequestWithContext(ctx context.Context, msg *Msg) ([]byte, error)
	RequestAsync(msg *Msg, handler ReplyHandler, timeoutHandler TimeoutHandler) error
	Close()
}
