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

package mockoperation

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/kubeclipper/kubeclipper/pkg/models"
	query "github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	watch "k8s.io/apimachinery/pkg/watch"
)

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

// GetOperation mocks base method.
func (m *MockReader) GetOperation(ctx context.Context, name string) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperation", ctx, name)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperation indicates an expected call of GetOperation.
func (mr *MockReaderMockRecorder) GetOperation(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperation", reflect.TypeOf((*MockReader)(nil).GetOperation), ctx, name)
}

// GetOperationEx mocks base method.
func (m *MockReader) GetOperationEx(ctx context.Context, name, resourceVersion string) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperationEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperationEx indicates an expected call of GetOperationEx.
func (mr *MockReaderMockRecorder) GetOperationEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperationEx", reflect.TypeOf((*MockReader)(nil).GetOperationEx), ctx, name, resourceVersion)
}

// ListOperations mocks base method.
func (m *MockReader) ListOperations(ctx context.Context, query *query.Query) (*v1.OperationList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperations", ctx, query)
	ret0, _ := ret[0].(*v1.OperationList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOperations indicates an expected call of ListOperations.
func (mr *MockReaderMockRecorder) ListOperations(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperations", reflect.TypeOf((*MockReader)(nil).ListOperations), ctx, query)
}

// ListOperationsEx mocks base method.
func (m *MockReader) ListOperationsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperationsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOperationsEx indicates an expected call of ListOperationsEx.
func (mr *MockReaderMockRecorder) ListOperationsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperationsEx", reflect.TypeOf((*MockReader)(nil).ListOperationsEx), ctx, query)
}

// WatchOperations mocks base method.
func (m *MockReader) WatchOperations(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchOperations", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchOperations indicates an expected call of WatchOperations.
func (mr *MockReaderMockRecorder) WatchOperations(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchOperations", reflect.TypeOf((*MockReader)(nil).WatchOperations), ctx, query)
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

// GetOperationEx mocks base method.
func (m *MockReaderEx) GetOperationEx(ctx context.Context, name, resourceVersion string) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperationEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperationEx indicates an expected call of GetOperationEx.
func (mr *MockReaderExMockRecorder) GetOperationEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperationEx", reflect.TypeOf((*MockReaderEx)(nil).GetOperationEx), ctx, name, resourceVersion)
}

// ListOperationsEx mocks base method.
func (m *MockReaderEx) ListOperationsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperationsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOperationsEx indicates an expected call of ListOperationsEx.
func (mr *MockReaderExMockRecorder) ListOperationsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperationsEx", reflect.TypeOf((*MockReaderEx)(nil).ListOperationsEx), ctx, query)
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

// CreateOperation mocks base method.
func (m *MockWriter) CreateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOperation", ctx, operation)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOperation indicates an expected call of CreateOperation.
func (mr *MockWriterMockRecorder) CreateOperation(ctx, operation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOperation", reflect.TypeOf((*MockWriter)(nil).CreateOperation), ctx, operation)
}

// DeleteOperation mocks base method.
func (m *MockWriter) DeleteOperation(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOperation", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOperation indicates an expected call of DeleteOperation.
func (mr *MockWriterMockRecorder) DeleteOperation(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOperation", reflect.TypeOf((*MockWriter)(nil).DeleteOperation), ctx, name)
}

// DeleteOperationCollection mocks base method.
func (m *MockWriter) DeleteOperationCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOperationCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOperationCollection indicates an expected call of DeleteOperationCollection.
func (mr *MockWriterMockRecorder) DeleteOperationCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOperationCollection", reflect.TypeOf((*MockWriter)(nil).DeleteOperationCollection), ctx, query)
}

// UpdateOperation mocks base method.
func (m *MockWriter) UpdateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOperation", ctx, operation)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOperation indicates an expected call of UpdateOperation.
func (mr *MockWriterMockRecorder) UpdateOperation(ctx, operation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOperation", reflect.TypeOf((*MockWriter)(nil).UpdateOperation), ctx, operation)
}

// UpdateOperationStatus mocks base method.
func (m *MockWriter) UpdateOperationStatus(ctx context.Context, name string, status *v1.OperationStatus) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOperationStatus", ctx, name, status)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOperationStatus indicates an expected call of UpdateOperationStatus.
func (mr *MockWriterMockRecorder) UpdateOperationStatus(ctx, name, status interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOperationStatus", reflect.TypeOf((*MockWriter)(nil).UpdateOperationStatus), ctx, name, status)
}

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

// CreateOperation mocks base method.
func (m *MockOperator) CreateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOperation", ctx, operation)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOperation indicates an expected call of CreateOperation.
func (mr *MockOperatorMockRecorder) CreateOperation(ctx, operation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOperation", reflect.TypeOf((*MockOperator)(nil).CreateOperation), ctx, operation)
}

// DeleteOperation mocks base method.
func (m *MockOperator) DeleteOperation(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOperation", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOperation indicates an expected call of DeleteOperation.
func (mr *MockOperatorMockRecorder) DeleteOperation(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOperation", reflect.TypeOf((*MockOperator)(nil).DeleteOperation), ctx, name)
}

// DeleteOperationCollection mocks base method.
func (m *MockOperator) DeleteOperationCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOperationCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOperationCollection indicates an expected call of DeleteOperationCollection.
func (mr *MockOperatorMockRecorder) DeleteOperationCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOperationCollection", reflect.TypeOf((*MockOperator)(nil).DeleteOperationCollection), ctx, query)
}

// GetOperation mocks base method.
func (m *MockOperator) GetOperation(ctx context.Context, name string) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperation", ctx, name)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperation indicates an expected call of GetOperation.
func (mr *MockOperatorMockRecorder) GetOperation(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperation", reflect.TypeOf((*MockOperator)(nil).GetOperation), ctx, name)
}

// GetOperationEx mocks base method.
func (m *MockOperator) GetOperationEx(ctx context.Context, name, resourceVersion string) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperationEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOperationEx indicates an expected call of GetOperationEx.
func (mr *MockOperatorMockRecorder) GetOperationEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperationEx", reflect.TypeOf((*MockOperator)(nil).GetOperationEx), ctx, name, resourceVersion)
}

// ListOperations mocks base method.
func (m *MockOperator) ListOperations(ctx context.Context, query *query.Query) (*v1.OperationList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperations", ctx, query)
	ret0, _ := ret[0].(*v1.OperationList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOperations indicates an expected call of ListOperations.
func (mr *MockOperatorMockRecorder) ListOperations(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperations", reflect.TypeOf((*MockOperator)(nil).ListOperations), ctx, query)
}

// ListOperationsEx mocks base method.
func (m *MockOperator) ListOperationsEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOperationsEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOperationsEx indicates an expected call of ListOperationsEx.
func (mr *MockOperatorMockRecorder) ListOperationsEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOperationsEx", reflect.TypeOf((*MockOperator)(nil).ListOperationsEx), ctx, query)
}

// UpdateOperation mocks base method.
func (m *MockOperator) UpdateOperation(ctx context.Context, operation *v1.Operation) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOperation", ctx, operation)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOperation indicates an expected call of UpdateOperation.
func (mr *MockOperatorMockRecorder) UpdateOperation(ctx, operation interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOperation", reflect.TypeOf((*MockOperator)(nil).UpdateOperation), ctx, operation)
}

// UpdateOperationStatus mocks base method.
func (m *MockOperator) UpdateOperationStatus(ctx context.Context, name string, status *v1.OperationStatus) (*v1.Operation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOperationStatus", ctx, name, status)
	ret0, _ := ret[0].(*v1.Operation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOperationStatus indicates an expected call of UpdateOperationStatus.
func (mr *MockOperatorMockRecorder) UpdateOperationStatus(ctx, name, status interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOperationStatus", reflect.TypeOf((*MockOperator)(nil).UpdateOperationStatus), ctx, name, status)
}

// WatchOperations mocks base method.
func (m *MockOperator) WatchOperations(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchOperations", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchOperations indicates an expected call of WatchOperations.
func (mr *MockOperatorMockRecorder) WatchOperations(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchOperations", reflect.TypeOf((*MockOperator)(nil).WatchOperations), ctx, query)
}
