/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package mocknatsio

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	natsio "github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
	nats "github.com/nats-io/nats.go"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockInterface) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockInterfaceMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockInterface)(nil).Close))
}

// InitConn mocks base method.
func (m *MockInterface) InitConn(stopCh <-chan struct{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitConn", stopCh)
	ret0, _ := ret[0].(error)
	return ret0
}

// InitConn indicates an expected call of InitConn.
func (mr *MockInterfaceMockRecorder) InitConn(stopCh interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitConn", reflect.TypeOf((*MockInterface)(nil).InitConn), stopCh)
}

// Publish mocks base method.
func (m *MockInterface) Publish(msg *natsio.Msg) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", msg)
	ret0, _ := ret[0].(error)
	return ret0
}

// Publish indicates an expected call of Publish.
func (mr *MockInterfaceMockRecorder) Publish(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Publish", reflect.TypeOf((*MockInterface)(nil).Publish), msg)
}

// QueueSubscribe mocks base method.
func (m *MockInterface) QueueSubscribe(subj, queue string, handler nats.MsgHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueSubscribe", subj, queue, handler)
	ret0, _ := ret[0].(error)
	return ret0
}

// QueueSubscribe indicates an expected call of QueueSubscribe.
func (mr *MockInterfaceMockRecorder) QueueSubscribe(subj, queue, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueSubscribe", reflect.TypeOf((*MockInterface)(nil).QueueSubscribe), subj, queue, handler)
}

// Request mocks base method.
func (m *MockInterface) Request(msg *natsio.Msg, timeoutHandler natsio.TimeoutHandler) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Request", msg, timeoutHandler)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Request indicates an expected call of Request.
func (mr *MockInterfaceMockRecorder) Request(msg, timeoutHandler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Request", reflect.TypeOf((*MockInterface)(nil).Request), msg, timeoutHandler)
}

// RequestAsync mocks base method.
func (m *MockInterface) RequestAsync(msg *natsio.Msg, handler natsio.ReplyHandler, timeoutHandler natsio.TimeoutHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestAsync", msg, handler, timeoutHandler)
	ret0, _ := ret[0].(error)
	return ret0
}

// RequestAsync indicates an expected call of RequestAsync.
func (mr *MockInterfaceMockRecorder) RequestAsync(msg, handler, timeoutHandler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestAsync", reflect.TypeOf((*MockInterface)(nil).RequestAsync), msg, handler, timeoutHandler)
}

// RequestWithContext mocks base method.
func (m *MockInterface) RequestWithContext(ctx context.Context, msg *natsio.Msg) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestWithContext", ctx, msg)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RequestWithContext indicates an expected call of RequestWithContext.
func (mr *MockInterfaceMockRecorder) RequestWithContext(ctx, msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestWithContext", reflect.TypeOf((*MockInterface)(nil).RequestWithContext), ctx, msg)
}

// RunServer mocks base method.
func (m *MockInterface) RunServer(stopCh <-chan struct{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunServer", stopCh)
	ret0, _ := ret[0].(error)
	return ret0
}

// RunServer indicates an expected call of RunServer.
func (mr *MockInterfaceMockRecorder) RunServer(stopCh interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunServer", reflect.TypeOf((*MockInterface)(nil).RunServer), stopCh)
}

// SetClosedHandler mocks base method.
func (m *MockInterface) SetClosedHandler(handler nats.ConnHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetClosedHandler", handler)
}

// SetClosedHandler indicates an expected call of SetClosedHandler.
func (mr *MockInterfaceMockRecorder) SetClosedHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetClosedHandler", reflect.TypeOf((*MockInterface)(nil).SetClosedHandler), handler)
}

// SetDisconnectErrHandler mocks base method.
func (m *MockInterface) SetDisconnectErrHandler(handler nats.ConnErrHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDisconnectErrHandler", handler)
}

// SetDisconnectErrHandler indicates an expected call of SetDisconnectErrHandler.
func (mr *MockInterfaceMockRecorder) SetDisconnectErrHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDisconnectErrHandler", reflect.TypeOf((*MockInterface)(nil).SetDisconnectErrHandler), handler)
}

// SetErrorHandler mocks base method.
func (m *MockInterface) SetErrorHandler(handler nats.ErrHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetErrorHandler", handler)
}

// SetErrorHandler indicates an expected call of SetErrorHandler.
func (mr *MockInterfaceMockRecorder) SetErrorHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetErrorHandler", reflect.TypeOf((*MockInterface)(nil).SetErrorHandler), handler)
}

// SetReconnectHandler mocks base method.
func (m *MockInterface) SetReconnectHandler(handler nats.ConnHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetReconnectHandler", handler)
}

// SetReconnectHandler indicates an expected call of SetReconnectHandler.
func (mr *MockInterfaceMockRecorder) SetReconnectHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetReconnectHandler", reflect.TypeOf((*MockInterface)(nil).SetReconnectHandler), handler)
}

// Subscribe mocks base method.
func (m *MockInterface) Subscribe(subj string, handler nats.MsgHandler) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Subscribe", subj, handler)
	ret0, _ := ret[0].(error)
	return ret0
}

// Subscribe indicates an expected call of Subscribe.
func (mr *MockInterfaceMockRecorder) Subscribe(subj, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Subscribe", reflect.TypeOf((*MockInterface)(nil).Subscribe), subj, handler)
}
