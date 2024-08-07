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

package mocklease

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	query "github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "k8s.io/api/coordination/v1"
	watch "k8s.io/apimachinery/pkg/watch"
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

// CreateLease mocks base method.
func (m *MockOperator) CreateLease(ctx context.Context, lease *v1.Lease) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLease", ctx, lease)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateLease indicates an expected call of CreateLease.
func (mr *MockOperatorMockRecorder) CreateLease(ctx, lease interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLease", reflect.TypeOf((*MockOperator)(nil).CreateLease), ctx, lease)
}

// GetLease mocks base method.
func (m *MockOperator) GetLease(ctx context.Context, name, resourceVersion string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLease", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLease indicates an expected call of GetLease.
func (mr *MockOperatorMockRecorder) GetLease(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLease", reflect.TypeOf((*MockOperator)(nil).GetLease), ctx, name, resourceVersion)
}

// GetLeaseWithNamespace mocks base method.
func (m *MockOperator) GetLeaseWithNamespace(ctx context.Context, name, namespace string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLeaseWithNamespace", ctx, name, namespace)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLeaseWithNamespace indicates an expected call of GetLeaseWithNamespace.
func (mr *MockOperatorMockRecorder) GetLeaseWithNamespace(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLeaseWithNamespace", reflect.TypeOf((*MockOperator)(nil).GetLeaseWithNamespace), ctx, name, namespace)
}

// GetLeaseWithNamespaceEx mocks base method.
func (m *MockOperator) GetLeaseWithNamespaceEx(ctx context.Context, name, namespace, resourceVersion string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLeaseWithNamespaceEx", ctx, name, namespace, resourceVersion)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLeaseWithNamespaceEx indicates an expected call of GetLeaseWithNamespaceEx.
func (mr *MockOperatorMockRecorder) GetLeaseWithNamespaceEx(ctx, name, namespace, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLeaseWithNamespaceEx", reflect.TypeOf((*MockOperator)(nil).GetLeaseWithNamespaceEx), ctx, name, namespace, resourceVersion)
}

// ListLeases mocks base method.
func (m *MockOperator) ListLeases(ctx context.Context, query *query.Query) (*v1.LeaseList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLeases", ctx, query)
	ret0, _ := ret[0].(*v1.LeaseList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLeases indicates an expected call of ListLeases.
func (mr *MockOperatorMockRecorder) ListLeases(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLeases", reflect.TypeOf((*MockOperator)(nil).ListLeases), ctx, query)
}

// UpdateLease mocks base method.
func (m *MockOperator) UpdateLease(ctx context.Context, lease *v1.Lease) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateLease", ctx, lease)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateLease indicates an expected call of UpdateLease.
func (mr *MockOperatorMockRecorder) UpdateLease(ctx, lease interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLease", reflect.TypeOf((*MockOperator)(nil).UpdateLease), ctx, lease)
}

// WatchLease mocks base method.
func (m *MockOperator) WatchLease(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchLease", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchLease indicates an expected call of WatchLease.
func (mr *MockOperatorMockRecorder) WatchLease(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchLease", reflect.TypeOf((*MockOperator)(nil).WatchLease), ctx, query)
}

// MockLeaseReader is a mock of LeaseReader interface.
type MockLeaseReader struct {
	ctrl     *gomock.Controller
	recorder *MockLeaseReaderMockRecorder
}

// MockLeaseReaderMockRecorder is the mock recorder for MockLeaseReader.
type MockLeaseReaderMockRecorder struct {
	mock *MockLeaseReader
}

// NewMockLeaseReader creates a new mock instance.
func NewMockLeaseReader(ctrl *gomock.Controller) *MockLeaseReader {
	mock := &MockLeaseReader{ctrl: ctrl}
	mock.recorder = &MockLeaseReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLeaseReader) EXPECT() *MockLeaseReaderMockRecorder {
	return m.recorder
}

// GetLease mocks base method.
func (m *MockLeaseReader) GetLease(ctx context.Context, name, resourceVersion string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLease", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLease indicates an expected call of GetLease.
func (mr *MockLeaseReaderMockRecorder) GetLease(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLease", reflect.TypeOf((*MockLeaseReader)(nil).GetLease), ctx, name, resourceVersion)
}

// GetLeaseWithNamespace mocks base method.
func (m *MockLeaseReader) GetLeaseWithNamespace(ctx context.Context, name, namespace string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLeaseWithNamespace", ctx, name, namespace)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLeaseWithNamespace indicates an expected call of GetLeaseWithNamespace.
func (mr *MockLeaseReaderMockRecorder) GetLeaseWithNamespace(ctx, name, namespace interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLeaseWithNamespace", reflect.TypeOf((*MockLeaseReader)(nil).GetLeaseWithNamespace), ctx, name, namespace)
}

// GetLeaseWithNamespaceEx mocks base method.
func (m *MockLeaseReader) GetLeaseWithNamespaceEx(ctx context.Context, name, namespace, resourceVersion string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLeaseWithNamespaceEx", ctx, name, namespace, resourceVersion)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLeaseWithNamespaceEx indicates an expected call of GetLeaseWithNamespaceEx.
func (mr *MockLeaseReaderMockRecorder) GetLeaseWithNamespaceEx(ctx, name, namespace, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLeaseWithNamespaceEx", reflect.TypeOf((*MockLeaseReader)(nil).GetLeaseWithNamespaceEx), ctx, name, namespace, resourceVersion)
}

// ListLeases mocks base method.
func (m *MockLeaseReader) ListLeases(ctx context.Context, query *query.Query) (*v1.LeaseList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLeases", ctx, query)
	ret0, _ := ret[0].(*v1.LeaseList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLeases indicates an expected call of ListLeases.
func (mr *MockLeaseReaderMockRecorder) ListLeases(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLeases", reflect.TypeOf((*MockLeaseReader)(nil).ListLeases), ctx, query)
}

// WatchLease mocks base method.
func (m *MockLeaseReader) WatchLease(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchLease", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchLease indicates an expected call of WatchLease.
func (mr *MockLeaseReaderMockRecorder) WatchLease(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchLease", reflect.TypeOf((*MockLeaseReader)(nil).WatchLease), ctx, query)
}

// MockLeaseReaderEx is a mock of LeaseReaderEx interface.
type MockLeaseReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockLeaseReaderExMockRecorder
}

// MockLeaseReaderExMockRecorder is the mock recorder for MockLeaseReaderEx.
type MockLeaseReaderExMockRecorder struct {
	mock *MockLeaseReaderEx
}

// NewMockLeaseReaderEx creates a new mock instance.
func NewMockLeaseReaderEx(ctrl *gomock.Controller) *MockLeaseReaderEx {
	mock := &MockLeaseReaderEx{ctrl: ctrl}
	mock.recorder = &MockLeaseReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLeaseReaderEx) EXPECT() *MockLeaseReaderExMockRecorder {
	return m.recorder
}

// GetLeaseWithNamespaceEx mocks base method.
func (m *MockLeaseReaderEx) GetLeaseWithNamespaceEx(ctx context.Context, name, namespace, resourceVersion string) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLeaseWithNamespaceEx", ctx, name, namespace, resourceVersion)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLeaseWithNamespaceEx indicates an expected call of GetLeaseWithNamespaceEx.
func (mr *MockLeaseReaderExMockRecorder) GetLeaseWithNamespaceEx(ctx, name, namespace, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLeaseWithNamespaceEx", reflect.TypeOf((*MockLeaseReaderEx)(nil).GetLeaseWithNamespaceEx), ctx, name, namespace, resourceVersion)
}

// MockLeaseWriter is a mock of LeaseWriter interface.
type MockLeaseWriter struct {
	ctrl     *gomock.Controller
	recorder *MockLeaseWriterMockRecorder
}

// MockLeaseWriterMockRecorder is the mock recorder for MockLeaseWriter.
type MockLeaseWriterMockRecorder struct {
	mock *MockLeaseWriter
}

// NewMockLeaseWriter creates a new mock instance.
func NewMockLeaseWriter(ctrl *gomock.Controller) *MockLeaseWriter {
	mock := &MockLeaseWriter{ctrl: ctrl}
	mock.recorder = &MockLeaseWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLeaseWriter) EXPECT() *MockLeaseWriterMockRecorder {
	return m.recorder
}

// CreateLease mocks base method.
func (m *MockLeaseWriter) CreateLease(ctx context.Context, lease *v1.Lease) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLease", ctx, lease)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateLease indicates an expected call of CreateLease.
func (mr *MockLeaseWriterMockRecorder) CreateLease(ctx, lease interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLease", reflect.TypeOf((*MockLeaseWriter)(nil).CreateLease), ctx, lease)
}

// UpdateLease mocks base method.
func (m *MockLeaseWriter) UpdateLease(ctx context.Context, lease *v1.Lease) (*v1.Lease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateLease", ctx, lease)
	ret0, _ := ret[0].(*v1.Lease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateLease indicates an expected call of UpdateLease.
func (mr *MockLeaseWriterMockRecorder) UpdateLease(ctx, lease interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateLease", reflect.TypeOf((*MockLeaseWriter)(nil).UpdateLease), ctx, lease)
}
