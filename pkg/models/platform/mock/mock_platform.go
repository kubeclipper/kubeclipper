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

package mockplatform

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/kubeclipper/kubeclipper/pkg/models"
	query "github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

// MockOperator is a mock of Operator interface.
type MockOperator struct {
	ctrl     *gomock.Controller
	recorder *MockOperatorMockRecorder
}

// MockOperatorMockRecorder is the mock recorder for MockOperator.
type MockOperatorMockRecorder struct {
	mock *MockOperator
}

// NewMockOperator creates a new mock instance.
func NewMockOperator(ctrl *gomock.Controller) *MockOperator {
	mock := &MockOperator{ctrl: ctrl}
	mock.recorder = &MockOperatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOperator) EXPECT() *MockOperatorMockRecorder {
	return m.recorder
}

// CreateEvent mocks base method.
func (m *MockOperator) CreateEvent(ctx context.Context, Event *v1.Event) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEvent", ctx, Event)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEvent indicates an expected call of CreateEvent.
func (mr *MockOperatorMockRecorder) CreateEvent(ctx, Event interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEvent", reflect.TypeOf((*MockOperator)(nil).CreateEvent), ctx, Event)
}

// CreatePlatformSetting mocks base method.
func (m *MockOperator) CreatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePlatformSetting", ctx, platformSetting)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePlatformSetting indicates an expected call of CreatePlatformSetting.
func (mr *MockOperatorMockRecorder) CreatePlatformSetting(ctx, platformSetting interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePlatformSetting", reflect.TypeOf((*MockOperator)(nil).CreatePlatformSetting), ctx, platformSetting)
}

// DeleteEvent mocks base method.
func (m *MockOperator) DeleteEvent(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEvent", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEvent indicates an expected call of DeleteEvent.
func (mr *MockOperatorMockRecorder) DeleteEvent(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEvent", reflect.TypeOf((*MockOperator)(nil).DeleteEvent), ctx, name)
}

// DeleteEventCollection mocks base method.
func (m *MockOperator) DeleteEventCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEventCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEventCollection indicates an expected call of DeleteEventCollection.
func (mr *MockOperatorMockRecorder) DeleteEventCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEventCollection", reflect.TypeOf((*MockOperator)(nil).DeleteEventCollection), ctx, query)
}

// GetEvent mocks base method.
func (m *MockOperator) GetEvent(ctx context.Context, name string) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEvent", ctx, name)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEvent indicates an expected call of GetEvent.
func (mr *MockOperatorMockRecorder) GetEvent(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEvent", reflect.TypeOf((*MockOperator)(nil).GetEvent), ctx, name)
}

// GetEventEx mocks base method.
func (m *MockOperator) GetEventEx(ctx context.Context, name, resourceVersion string) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventEx indicates an expected call of GetEventEx.
func (mr *MockOperatorMockRecorder) GetEventEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventEx", reflect.TypeOf((*MockOperator)(nil).GetEventEx), ctx, name, resourceVersion)
}

// GetPlatformSetting mocks base method.
func (m *MockOperator) GetPlatformSetting(ctx context.Context) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlatformSetting", ctx)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPlatformSetting indicates an expected call of GetPlatformSetting.
func (mr *MockOperatorMockRecorder) GetPlatformSetting(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlatformSetting", reflect.TypeOf((*MockOperator)(nil).GetPlatformSetting), ctx)
}

// ListEvents mocks base method.
func (m *MockOperator) ListEvents(ctx context.Context, query *query.Query) (*v1.EventList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEvents", ctx, query)
	ret0, _ := ret[0].(*v1.EventList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEvents indicates an expected call of ListEvents.
func (mr *MockOperatorMockRecorder) ListEvents(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEvents", reflect.TypeOf((*MockOperator)(nil).ListEvents), ctx, query)
}

// ListEventsEx mocks base method.
func (m *MockOperator) ListEventsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEventsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEventsEx indicates an expected call of ListEventsEx.
func (mr *MockOperatorMockRecorder) ListEventsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEventsEx", reflect.TypeOf((*MockOperator)(nil).ListEventsEx), ctx, query)
}

// UpdatePlatformSetting mocks base method.
func (m *MockOperator) UpdatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePlatformSetting", ctx, platformSetting)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdatePlatformSetting indicates an expected call of UpdatePlatformSetting.
func (mr *MockOperatorMockRecorder) UpdatePlatformSetting(ctx, platformSetting interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePlatformSetting", reflect.TypeOf((*MockOperator)(nil).UpdatePlatformSetting), ctx, platformSetting)
}

// MockReader is a mock of Reader interface.
type MockReader struct {
	ctrl     *gomock.Controller
	recorder *MockReaderMockRecorder
}

// MockReaderMockRecorder is the mock recorder for MockReader.
type MockReaderMockRecorder struct {
	mock *MockReader
}

// NewMockReader creates a new mock instance.
func NewMockReader(ctrl *gomock.Controller) *MockReader {
	mock := &MockReader{ctrl: ctrl}
	mock.recorder = &MockReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReader) EXPECT() *MockReaderMockRecorder {
	return m.recorder
}

// GetPlatformSetting mocks base method.
func (m *MockReader) GetPlatformSetting(ctx context.Context) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlatformSetting", ctx)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPlatformSetting indicates an expected call of GetPlatformSetting.
func (mr *MockReaderMockRecorder) GetPlatformSetting(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlatformSetting", reflect.TypeOf((*MockReader)(nil).GetPlatformSetting), ctx)
}

// MockWriter is a mock of Writer interface.
type MockWriter struct {
	ctrl     *gomock.Controller
	recorder *MockWriterMockRecorder
}

// MockWriterMockRecorder is the mock recorder for MockWriter.
type MockWriterMockRecorder struct {
	mock *MockWriter
}

// NewMockWriter creates a new mock instance.
func NewMockWriter(ctrl *gomock.Controller) *MockWriter {
	mock := &MockWriter{ctrl: ctrl}
	mock.recorder = &MockWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWriter) EXPECT() *MockWriterMockRecorder {
	return m.recorder
}

// CreatePlatformSetting mocks base method.
func (m *MockWriter) CreatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePlatformSetting", ctx, platformSetting)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePlatformSetting indicates an expected call of CreatePlatformSetting.
func (mr *MockWriterMockRecorder) CreatePlatformSetting(ctx, platformSetting interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePlatformSetting", reflect.TypeOf((*MockWriter)(nil).CreatePlatformSetting), ctx, platformSetting)
}

// UpdatePlatformSetting mocks base method.
func (m *MockWriter) UpdatePlatformSetting(ctx context.Context, platformSetting *v1.PlatformSetting) (*v1.PlatformSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePlatformSetting", ctx, platformSetting)
	ret0, _ := ret[0].(*v1.PlatformSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdatePlatformSetting indicates an expected call of UpdatePlatformSetting.
func (mr *MockWriterMockRecorder) UpdatePlatformSetting(ctx, platformSetting interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePlatformSetting", reflect.TypeOf((*MockWriter)(nil).UpdatePlatformSetting), ctx, platformSetting)
}

// MockReaderEx is a mock of ReaderEx interface.
type MockReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockReaderExMockRecorder
}

// MockReaderExMockRecorder is the mock recorder for MockReaderEx.
type MockReaderExMockRecorder struct {
	mock *MockReaderEx
}

// NewMockReaderEx creates a new mock instance.
func NewMockReaderEx(ctrl *gomock.Controller) *MockReaderEx {
	mock := &MockReaderEx{ctrl: ctrl}
	mock.recorder = &MockReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReaderEx) EXPECT() *MockReaderExMockRecorder {
	return m.recorder
}

// MockEventReader is a mock of EventReader interface.
type MockEventReader struct {
	ctrl     *gomock.Controller
	recorder *MockEventReaderMockRecorder
}

// MockEventReaderMockRecorder is the mock recorder for MockEventReader.
type MockEventReaderMockRecorder struct {
	mock *MockEventReader
}

// NewMockEventReader creates a new mock instance.
func NewMockEventReader(ctrl *gomock.Controller) *MockEventReader {
	mock := &MockEventReader{ctrl: ctrl}
	mock.recorder = &MockEventReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventReader) EXPECT() *MockEventReaderMockRecorder {
	return m.recorder
}

// GetEvent mocks base method.
func (m *MockEventReader) GetEvent(ctx context.Context, name string) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEvent", ctx, name)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEvent indicates an expected call of GetEvent.
func (mr *MockEventReaderMockRecorder) GetEvent(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEvent", reflect.TypeOf((*MockEventReader)(nil).GetEvent), ctx, name)
}

// GetEventEx mocks base method.
func (m *MockEventReader) GetEventEx(ctx context.Context, name, resourceVersion string) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventEx indicates an expected call of GetEventEx.
func (mr *MockEventReaderMockRecorder) GetEventEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventEx", reflect.TypeOf((*MockEventReader)(nil).GetEventEx), ctx, name, resourceVersion)
}

// ListEvents mocks base method.
func (m *MockEventReader) ListEvents(ctx context.Context, query *query.Query) (*v1.EventList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEvents", ctx, query)
	ret0, _ := ret[0].(*v1.EventList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEvents indicates an expected call of ListEvents.
func (mr *MockEventReaderMockRecorder) ListEvents(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEvents", reflect.TypeOf((*MockEventReader)(nil).ListEvents), ctx, query)
}

// ListEventsEx mocks base method.
func (m *MockEventReader) ListEventsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEventsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEventsEx indicates an expected call of ListEventsEx.
func (mr *MockEventReaderMockRecorder) ListEventsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEventsEx", reflect.TypeOf((*MockEventReader)(nil).ListEventsEx), ctx, query)
}

// MockEventReaderEx is a mock of EventReaderEx interface.
type MockEventReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockEventReaderExMockRecorder
}

// MockEventReaderExMockRecorder is the mock recorder for MockEventReaderEx.
type MockEventReaderExMockRecorder struct {
	mock *MockEventReaderEx
}

// NewMockEventReaderEx creates a new mock instance.
func NewMockEventReaderEx(ctrl *gomock.Controller) *MockEventReaderEx {
	mock := &MockEventReaderEx{ctrl: ctrl}
	mock.recorder = &MockEventReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventReaderEx) EXPECT() *MockEventReaderExMockRecorder {
	return m.recorder
}

// GetEventEx mocks base method.
func (m *MockEventReaderEx) GetEventEx(ctx context.Context, name, resourceVersion string) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEventEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEventEx indicates an expected call of GetEventEx.
func (mr *MockEventReaderExMockRecorder) GetEventEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEventEx", reflect.TypeOf((*MockEventReaderEx)(nil).GetEventEx), ctx, name, resourceVersion)
}

// ListEventsEx mocks base method.
func (m *MockEventReaderEx) ListEventsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEventsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEventsEx indicates an expected call of ListEventsEx.
func (mr *MockEventReaderExMockRecorder) ListEventsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEventsEx", reflect.TypeOf((*MockEventReaderEx)(nil).ListEventsEx), ctx, query)
}

// MockEventWriter is a mock of EventWriter interface.
type MockEventWriter struct {
	ctrl     *gomock.Controller
	recorder *MockEventWriterMockRecorder
}

// MockEventWriterMockRecorder is the mock recorder for MockEventWriter.
type MockEventWriterMockRecorder struct {
	mock *MockEventWriter
}

// NewMockEventWriter creates a new mock instance.
func NewMockEventWriter(ctrl *gomock.Controller) *MockEventWriter {
	mock := &MockEventWriter{ctrl: ctrl}
	mock.recorder = &MockEventWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventWriter) EXPECT() *MockEventWriterMockRecorder {
	return m.recorder
}

// CreateEvent mocks base method.
func (m *MockEventWriter) CreateEvent(ctx context.Context, Event *v1.Event) (*v1.Event, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEvent", ctx, Event)
	ret0, _ := ret[0].(*v1.Event)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEvent indicates an expected call of CreateEvent.
func (mr *MockEventWriterMockRecorder) CreateEvent(ctx, Event interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEvent", reflect.TypeOf((*MockEventWriter)(nil).CreateEvent), ctx, Event)
}

// DeleteEvent mocks base method.
func (m *MockEventWriter) DeleteEvent(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEvent", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEvent indicates an expected call of DeleteEvent.
func (mr *MockEventWriterMockRecorder) DeleteEvent(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEvent", reflect.TypeOf((*MockEventWriter)(nil).DeleteEvent), ctx, name)
}

// DeleteEventCollection mocks base method.
func (m *MockEventWriter) DeleteEventCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEventCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEventCollection indicates an expected call of DeleteEventCollection.
func (mr *MockEventWriterMockRecorder) DeleteEventCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEventCollection", reflect.TypeOf((*MockEventWriter)(nil).DeleteEventCollection), ctx, query)
}
