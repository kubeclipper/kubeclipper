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

package mockiam

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	models "github.com/kubeclipper/kubeclipper/pkg/models"
	query "github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
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

// CreateLoginRecord mocks base method.
func (m *MockOperator) CreateLoginRecord(ctx context.Context, record *v1.LoginRecord) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLoginRecord", ctx, record)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateLoginRecord indicates an expected call of CreateLoginRecord.
func (mr *MockOperatorMockRecorder) CreateLoginRecord(ctx, record interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLoginRecord", reflect.TypeOf((*MockOperator)(nil).CreateLoginRecord), ctx, record)
}

// CreateRole mocks base method.
func (m *MockOperator) CreateRole(ctx context.Context, role *v1.GlobalRole) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRole", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRole indicates an expected call of CreateRole.
func (mr *MockOperatorMockRecorder) CreateRole(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRole", reflect.TypeOf((*MockOperator)(nil).CreateRole), ctx, role)
}

// CreateRoleBinding mocks base method.
func (m *MockOperator) CreateRoleBinding(ctx context.Context, role *v1.GlobalRoleBinding) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRoleBinding", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRoleBinding indicates an expected call of CreateRoleBinding.
func (mr *MockOperatorMockRecorder) CreateRoleBinding(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRoleBinding", reflect.TypeOf((*MockOperator)(nil).CreateRoleBinding), ctx, role)
}

// CreateToken mocks base method.
func (m *MockOperator) CreateToken(ctx context.Context, token *v1.Token) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateToken", ctx, token)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateToken indicates an expected call of CreateToken.
func (mr *MockOperatorMockRecorder) CreateToken(ctx, token interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateToken", reflect.TypeOf((*MockOperator)(nil).CreateToken), ctx, token)
}

// CreateUser mocks base method.
func (m *MockOperator) CreateUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, user)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockOperatorMockRecorder) CreateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockOperator)(nil).CreateUser), ctx, user)
}

// DeleteLoginRecord mocks base method.
func (m *MockOperator) DeleteLoginRecord(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLoginRecord", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteLoginRecord indicates an expected call of DeleteLoginRecord.
func (mr *MockOperatorMockRecorder) DeleteLoginRecord(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLoginRecord", reflect.TypeOf((*MockOperator)(nil).DeleteLoginRecord), ctx, name)
}

// DeleteLoginRecordCollection mocks base method.
func (m *MockOperator) DeleteLoginRecordCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLoginRecordCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteLoginRecordCollection indicates an expected call of DeleteLoginRecordCollection.
func (mr *MockOperatorMockRecorder) DeleteLoginRecordCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLoginRecordCollection", reflect.TypeOf((*MockOperator)(nil).DeleteLoginRecordCollection), ctx, query)
}

// DeleteRole mocks base method.
func (m *MockOperator) DeleteRole(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRole", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRole indicates an expected call of DeleteRole.
func (mr *MockOperatorMockRecorder) DeleteRole(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRole", reflect.TypeOf((*MockOperator)(nil).DeleteRole), ctx, name)
}

// DeleteRoleBinding mocks base method.
func (m *MockOperator) DeleteRoleBinding(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRoleBinding", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRoleBinding indicates an expected call of DeleteRoleBinding.
func (mr *MockOperatorMockRecorder) DeleteRoleBinding(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRoleBinding", reflect.TypeOf((*MockOperator)(nil).DeleteRoleBinding), ctx, name)
}

// DeleteToken mocks base method.
func (m *MockOperator) DeleteToken(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteToken", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteToken indicates an expected call of DeleteToken.
func (mr *MockOperatorMockRecorder) DeleteToken(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteToken", reflect.TypeOf((*MockOperator)(nil).DeleteToken), ctx, name)
}

// DeleteTokenCollection mocks base method.
func (m *MockOperator) DeleteTokenCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTokenCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTokenCollection indicates an expected call of DeleteTokenCollection.
func (mr *MockOperatorMockRecorder) DeleteTokenCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTokenCollection", reflect.TypeOf((*MockOperator)(nil).DeleteTokenCollection), ctx, query)
}

// DeleteUser mocks base method.
func (m *MockOperator) DeleteUser(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockOperatorMockRecorder) DeleteUser(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockOperator)(nil).DeleteUser), ctx, name)
}

// GetLoginRecord mocks base method.
func (m *MockOperator) GetLoginRecord(ctx context.Context, name string) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoginRecord", ctx, name)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoginRecord indicates an expected call of GetLoginRecord.
func (mr *MockOperatorMockRecorder) GetLoginRecord(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoginRecord", reflect.TypeOf((*MockOperator)(nil).GetLoginRecord), ctx, name)
}

// GetLoginRecordEx mocks base method.
func (m *MockOperator) GetLoginRecordEx(ctx context.Context, name, resourceVersion string) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoginRecordEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoginRecordEx indicates an expected call of GetLoginRecordEx.
func (mr *MockOperatorMockRecorder) GetLoginRecordEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoginRecordEx", reflect.TypeOf((*MockOperator)(nil).GetLoginRecordEx), ctx, name, resourceVersion)
}

// GetRole mocks base method.
func (m *MockOperator) GetRole(ctx context.Context, name string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRole", ctx, name)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRole indicates an expected call of GetRole.
func (mr *MockOperatorMockRecorder) GetRole(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRole", reflect.TypeOf((*MockOperator)(nil).GetRole), ctx, name)
}

// GetRoleBinding mocks base method.
func (m *MockOperator) GetRoleBinding(ctx context.Context, name string) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleBinding", ctx, name)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleBinding indicates an expected call of GetRoleBinding.
func (mr *MockOperatorMockRecorder) GetRoleBinding(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleBinding", reflect.TypeOf((*MockOperator)(nil).GetRoleBinding), ctx, name)
}

// GetRoleBindingEx mocks base method.
func (m *MockOperator) GetRoleBindingEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleBindingEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleBindingEx indicates an expected call of GetRoleBindingEx.
func (mr *MockOperatorMockRecorder) GetRoleBindingEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleBindingEx", reflect.TypeOf((*MockOperator)(nil).GetRoleBindingEx), ctx, name, resourceVersion)
}

// GetRoleEx mocks base method.
func (m *MockOperator) GetRoleEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleEx indicates an expected call of GetRoleEx.
func (mr *MockOperatorMockRecorder) GetRoleEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleEx", reflect.TypeOf((*MockOperator)(nil).GetRoleEx), ctx, name, resourceVersion)
}

// GetRoleOfUser mocks base method.
func (m *MockOperator) GetRoleOfUser(ctx context.Context, username string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleOfUser", ctx, username)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleOfUser indicates an expected call of GetRoleOfUser.
func (mr *MockOperatorMockRecorder) GetRoleOfUser(ctx, username interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleOfUser", reflect.TypeOf((*MockOperator)(nil).GetRoleOfUser), ctx, username)
}

// GetToken mocks base method.
func (m *MockOperator) GetToken(ctx context.Context, name string) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetToken", ctx, name)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetToken indicates an expected call of GetToken.
func (mr *MockOperatorMockRecorder) GetToken(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetToken", reflect.TypeOf((*MockOperator)(nil).GetToken), ctx, name)
}

// GetTokenEx mocks base method.
func (m *MockOperator) GetTokenEx(ctx context.Context, name, resourceVersion string) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTokenEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTokenEx indicates an expected call of GetTokenEx.
func (mr *MockOperatorMockRecorder) GetTokenEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTokenEx", reflect.TypeOf((*MockOperator)(nil).GetTokenEx), ctx, name, resourceVersion)
}

// GetUser mocks base method.
func (m *MockOperator) GetUser(ctx context.Context, name string) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, name)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockOperatorMockRecorder) GetUser(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockOperator)(nil).GetUser), ctx, name)
}

// GetUserEx mocks base method.
func (m *MockOperator) GetUserEx(ctx context.Context, name, resourceVersion string, desensitization, includeRole bool) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserEx", ctx, name, resourceVersion, desensitization, includeRole)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserEx indicates an expected call of GetUserEx.
func (mr *MockOperatorMockRecorder) GetUserEx(ctx, name, resourceVersion, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserEx", reflect.TypeOf((*MockOperator)(nil).GetUserEx), ctx, name, resourceVersion, desensitization, includeRole)
}

// ListLoginRecordEx mocks base method.
func (m *MockOperator) ListLoginRecordEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLoginRecordEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLoginRecordEx indicates an expected call of ListLoginRecordEx.
func (mr *MockOperatorMockRecorder) ListLoginRecordEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLoginRecordEx", reflect.TypeOf((*MockOperator)(nil).ListLoginRecordEx), ctx, query)
}

// ListLoginRecords mocks base method.
func (m *MockOperator) ListLoginRecords(ctx context.Context, query *query.Query) (*v1.LoginRecordList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLoginRecords", ctx, query)
	ret0, _ := ret[0].(*v1.LoginRecordList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLoginRecords indicates an expected call of ListLoginRecords.
func (mr *MockOperatorMockRecorder) ListLoginRecords(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLoginRecords", reflect.TypeOf((*MockOperator)(nil).ListLoginRecords), ctx, query)
}

// ListRoleBindingEx mocks base method.
func (m *MockOperator) ListRoleBindingEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleBindingEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleBindingEx indicates an expected call of ListRoleBindingEx.
func (mr *MockOperatorMockRecorder) ListRoleBindingEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleBindingEx", reflect.TypeOf((*MockOperator)(nil).ListRoleBindingEx), ctx, query)
}

// ListRoleBindings mocks base method.
func (m *MockOperator) ListRoleBindings(ctx context.Context, query *query.Query) (*v1.GlobalRoleBindingList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleBindings", ctx, query)
	ret0, _ := ret[0].(*v1.GlobalRoleBindingList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleBindings indicates an expected call of ListRoleBindings.
func (mr *MockOperatorMockRecorder) ListRoleBindings(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleBindings", reflect.TypeOf((*MockOperator)(nil).ListRoleBindings), ctx, query)
}

// ListRoleEx mocks base method.
func (m *MockOperator) ListRoleEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleEx indicates an expected call of ListRoleEx.
func (mr *MockOperatorMockRecorder) ListRoleEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleEx", reflect.TypeOf((*MockOperator)(nil).ListRoleEx), ctx, query)
}

// ListRoles mocks base method.
func (m *MockOperator) ListRoles(ctx context.Context, query *query.Query) (*v1.GlobalRoleList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoles", ctx, query)
	ret0, _ := ret[0].(*v1.GlobalRoleList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoles indicates an expected call of ListRoles.
func (mr *MockOperatorMockRecorder) ListRoles(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoles", reflect.TypeOf((*MockOperator)(nil).ListRoles), ctx, query)
}

// ListTokenEx mocks base method.
func (m *MockOperator) ListTokenEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTokenEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTokenEx indicates an expected call of ListTokenEx.
func (mr *MockOperatorMockRecorder) ListTokenEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTokenEx", reflect.TypeOf((*MockOperator)(nil).ListTokenEx), ctx, query)
}

// ListTokens mocks base method.
func (m *MockOperator) ListTokens(ctx context.Context, query *query.Query) (*v1.TokenList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTokens", ctx, query)
	ret0, _ := ret[0].(*v1.TokenList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTokens indicates an expected call of ListTokens.
func (mr *MockOperatorMockRecorder) ListTokens(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTokens", reflect.TypeOf((*MockOperator)(nil).ListTokens), ctx, query)
}

// ListUserEx mocks base method.
func (m *MockOperator) ListUserEx(ctx context.Context, query *query.Query, desensitization, includeRole bool) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUserEx", ctx, query, desensitization, includeRole)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUserEx indicates an expected call of ListUserEx.
func (mr *MockOperatorMockRecorder) ListUserEx(ctx, query, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUserEx", reflect.TypeOf((*MockOperator)(nil).ListUserEx), ctx, query, desensitization, includeRole)
}

// ListUsers mocks base method.
func (m *MockOperator) ListUsers(ctx context.Context, query *query.Query) (*v1.UserList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", ctx, query)
	ret0, _ := ret[0].(*v1.UserList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockOperatorMockRecorder) ListUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockOperator)(nil).ListUsers), ctx, query)
}

// ListUsersByRole mocks base method.
func (m *MockOperator) ListUsersByRole(ctx context.Context, query *query.Query, role string) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByRole", ctx, query, role)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByRole indicates an expected call of ListUsersByRole.
func (mr *MockOperatorMockRecorder) ListUsersByRole(ctx, query, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByRole", reflect.TypeOf((*MockOperator)(nil).ListUsersByRole), ctx, query, role)
}

// UpdateRole mocks base method.
func (m *MockOperator) UpdateRole(ctx context.Context, role *v1.GlobalRole) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRole", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRole indicates an expected call of UpdateRole.
func (mr *MockOperatorMockRecorder) UpdateRole(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRole", reflect.TypeOf((*MockOperator)(nil).UpdateRole), ctx, role)
}

// UpdateRoleBinding mocks base method.
func (m *MockOperator) UpdateRoleBinding(ctx context.Context, role *v1.GlobalRoleBinding) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRoleBinding", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRoleBinding indicates an expected call of UpdateRoleBinding.
func (mr *MockOperatorMockRecorder) UpdateRoleBinding(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRoleBinding", reflect.TypeOf((*MockOperator)(nil).UpdateRoleBinding), ctx, role)
}

// UpdateToken mocks base method.
func (m *MockOperator) UpdateToken(ctx context.Context, token *v1.Token) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateToken", ctx, token)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateToken indicates an expected call of UpdateToken.
func (mr *MockOperatorMockRecorder) UpdateToken(ctx, token interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateToken", reflect.TypeOf((*MockOperator)(nil).UpdateToken), ctx, token)
}

// UpdateUser mocks base method.
func (m *MockOperator) UpdateUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, user)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockOperatorMockRecorder) UpdateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockOperator)(nil).UpdateUser), ctx, user)
}

// WatchLoginRecords mocks base method.
func (m *MockOperator) WatchLoginRecords(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchLoginRecords", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchLoginRecords indicates an expected call of WatchLoginRecords.
func (mr *MockOperatorMockRecorder) WatchLoginRecords(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchLoginRecords", reflect.TypeOf((*MockOperator)(nil).WatchLoginRecords), ctx, query)
}

// WatchRoleBindings mocks base method.
func (m *MockOperator) WatchRoleBindings(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRoleBindings", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRoleBindings indicates an expected call of WatchRoleBindings.
func (mr *MockOperatorMockRecorder) WatchRoleBindings(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRoleBindings", reflect.TypeOf((*MockOperator)(nil).WatchRoleBindings), ctx, query)
}

// WatchRoles mocks base method.
func (m *MockOperator) WatchRoles(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRoles", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRoles indicates an expected call of WatchRoles.
func (mr *MockOperatorMockRecorder) WatchRoles(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRoles", reflect.TypeOf((*MockOperator)(nil).WatchRoles), ctx, query)
}

// WatchTokens mocks base method.
func (m *MockOperator) WatchTokens(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchTokens", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchTokens indicates an expected call of WatchTokens.
func (mr *MockOperatorMockRecorder) WatchTokens(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchTokens", reflect.TypeOf((*MockOperator)(nil).WatchTokens), ctx, query)
}

// WatchUsers mocks base method.
func (m *MockOperator) WatchUsers(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchUsers", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchUsers indicates an expected call of WatchUsers.
func (mr *MockOperatorMockRecorder) WatchUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchUsers", reflect.TypeOf((*MockOperator)(nil).WatchUsers), ctx, query)
}

// MockUserReader is a mock of UserReader interface.
type MockUserReader struct {
	ctrl     *gomock.Controller
	recorder *MockUserReaderMockRecorder
}

// MockUserReaderMockRecorder is the mock recorder for MockUserReader.
type MockUserReaderMockRecorder struct {
	mock *MockUserReader
}

// NewMockUserReader creates a new mock instance.
func NewMockUserReader(ctrl *gomock.Controller) *MockUserReader {
	mock := &MockUserReader{ctrl: ctrl}
	mock.recorder = &MockUserReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserReader) EXPECT() *MockUserReaderMockRecorder {
	return m.recorder
}

// GetUser mocks base method.
func (m *MockUserReader) GetUser(ctx context.Context, name string) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, name)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserReaderMockRecorder) GetUser(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserReader)(nil).GetUser), ctx, name)
}

// GetUserEx mocks base method.
func (m *MockUserReader) GetUserEx(ctx context.Context, name, resourceVersion string, desensitization, includeRole bool) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserEx", ctx, name, resourceVersion, desensitization, includeRole)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserEx indicates an expected call of GetUserEx.
func (mr *MockUserReaderMockRecorder) GetUserEx(ctx, name, resourceVersion, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserEx", reflect.TypeOf((*MockUserReader)(nil).GetUserEx), ctx, name, resourceVersion, desensitization, includeRole)
}

// ListUserEx mocks base method.
func (m *MockUserReader) ListUserEx(ctx context.Context, query *query.Query, desensitization, includeRole bool) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUserEx", ctx, query, desensitization, includeRole)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUserEx indicates an expected call of ListUserEx.
func (mr *MockUserReaderMockRecorder) ListUserEx(ctx, query, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUserEx", reflect.TypeOf((*MockUserReader)(nil).ListUserEx), ctx, query, desensitization, includeRole)
}

// ListUsers mocks base method.
func (m *MockUserReader) ListUsers(ctx context.Context, query *query.Query) (*v1.UserList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", ctx, query)
	ret0, _ := ret[0].(*v1.UserList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockUserReaderMockRecorder) ListUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockUserReader)(nil).ListUsers), ctx, query)
}

// ListUsersByRole mocks base method.
func (m *MockUserReader) ListUsersByRole(ctx context.Context, query *query.Query, role string) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByRole", ctx, query, role)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByRole indicates an expected call of ListUsersByRole.
func (mr *MockUserReaderMockRecorder) ListUsersByRole(ctx, query, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByRole", reflect.TypeOf((*MockUserReader)(nil).ListUsersByRole), ctx, query, role)
}

// WatchUsers mocks base method.
func (m *MockUserReader) WatchUsers(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchUsers", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchUsers indicates an expected call of WatchUsers.
func (mr *MockUserReaderMockRecorder) WatchUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchUsers", reflect.TypeOf((*MockUserReader)(nil).WatchUsers), ctx, query)
}

// MockUserReaderEx is a mock of UserReaderEx interface.
type MockUserReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockUserReaderExMockRecorder
}

// MockUserReaderExMockRecorder is the mock recorder for MockUserReaderEx.
type MockUserReaderExMockRecorder struct {
	mock *MockUserReaderEx
}

// NewMockUserReaderEx creates a new mock instance.
func NewMockUserReaderEx(ctrl *gomock.Controller) *MockUserReaderEx {
	mock := &MockUserReaderEx{ctrl: ctrl}
	mock.recorder = &MockUserReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserReaderEx) EXPECT() *MockUserReaderExMockRecorder {
	return m.recorder
}

// GetUserEx mocks base method.
func (m *MockUserReaderEx) GetUserEx(ctx context.Context, name, resourceVersion string, desensitization, includeRole bool) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserEx", ctx, name, resourceVersion, desensitization, includeRole)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserEx indicates an expected call of GetUserEx.
func (mr *MockUserReaderExMockRecorder) GetUserEx(ctx, name, resourceVersion, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserEx", reflect.TypeOf((*MockUserReaderEx)(nil).GetUserEx), ctx, name, resourceVersion, desensitization, includeRole)
}

// ListUserEx mocks base method.
func (m *MockUserReaderEx) ListUserEx(ctx context.Context, query *query.Query, desensitization, includeRole bool) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUserEx", ctx, query, desensitization, includeRole)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUserEx indicates an expected call of ListUserEx.
func (mr *MockUserReaderExMockRecorder) ListUserEx(ctx, query, desensitization, includeRole interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUserEx", reflect.TypeOf((*MockUserReaderEx)(nil).ListUserEx), ctx, query, desensitization, includeRole)
}

// ListUsersByRole mocks base method.
func (m *MockUserReaderEx) ListUsersByRole(ctx context.Context, query *query.Query, role string) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByRole", ctx, query, role)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByRole indicates an expected call of ListUsersByRole.
func (mr *MockUserReaderExMockRecorder) ListUsersByRole(ctx, query, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByRole", reflect.TypeOf((*MockUserReaderEx)(nil).ListUsersByRole), ctx, query, role)
}

// MockUserWriter is a mock of UserWriter interface.
type MockUserWriter struct {
	ctrl     *gomock.Controller
	recorder *MockUserWriterMockRecorder
}

// MockUserWriterMockRecorder is the mock recorder for MockUserWriter.
type MockUserWriterMockRecorder struct {
	mock *MockUserWriter
}

// NewMockUserWriter creates a new mock instance.
func NewMockUserWriter(ctrl *gomock.Controller) *MockUserWriter {
	mock := &MockUserWriter{ctrl: ctrl}
	mock.recorder = &MockUserWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserWriter) EXPECT() *MockUserWriterMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockUserWriter) CreateUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, user)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockUserWriterMockRecorder) CreateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockUserWriter)(nil).CreateUser), ctx, user)
}

// DeleteUser mocks base method.
func (m *MockUserWriter) DeleteUser(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockUserWriterMockRecorder) DeleteUser(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockUserWriter)(nil).DeleteUser), ctx, name)
}

// UpdateUser mocks base method.
func (m *MockUserWriter) UpdateUser(ctx context.Context, user *v1.User) (*v1.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, user)
	ret0, _ := ret[0].(*v1.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserWriterMockRecorder) UpdateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserWriter)(nil).UpdateUser), ctx, user)
}

// MockRoleReader is a mock of RoleReader interface.
type MockRoleReader struct {
	ctrl     *gomock.Controller
	recorder *MockRoleReaderMockRecorder
}

// MockRoleReaderMockRecorder is the mock recorder for MockRoleReader.
type MockRoleReaderMockRecorder struct {
	mock *MockRoleReader
}

// NewMockRoleReader creates a new mock instance.
func NewMockRoleReader(ctrl *gomock.Controller) *MockRoleReader {
	mock := &MockRoleReader{ctrl: ctrl}
	mock.recorder = &MockRoleReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleReader) EXPECT() *MockRoleReaderMockRecorder {
	return m.recorder
}

// GetRole mocks base method.
func (m *MockRoleReader) GetRole(ctx context.Context, name string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRole", ctx, name)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRole indicates an expected call of GetRole.
func (mr *MockRoleReaderMockRecorder) GetRole(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRole", reflect.TypeOf((*MockRoleReader)(nil).GetRole), ctx, name)
}

// GetRoleEx mocks base method.
func (m *MockRoleReader) GetRoleEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleEx indicates an expected call of GetRoleEx.
func (mr *MockRoleReaderMockRecorder) GetRoleEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleEx", reflect.TypeOf((*MockRoleReader)(nil).GetRoleEx), ctx, name, resourceVersion)
}

// GetRoleOfUser mocks base method.
func (m *MockRoleReader) GetRoleOfUser(ctx context.Context, username string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleOfUser", ctx, username)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleOfUser indicates an expected call of GetRoleOfUser.
func (mr *MockRoleReaderMockRecorder) GetRoleOfUser(ctx, username interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleOfUser", reflect.TypeOf((*MockRoleReader)(nil).GetRoleOfUser), ctx, username)
}

// ListRoleEx mocks base method.
func (m *MockRoleReader) ListRoleEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleEx indicates an expected call of ListRoleEx.
func (mr *MockRoleReaderMockRecorder) ListRoleEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleEx", reflect.TypeOf((*MockRoleReader)(nil).ListRoleEx), ctx, query)
}

// ListRoles mocks base method.
func (m *MockRoleReader) ListRoles(ctx context.Context, query *query.Query) (*v1.GlobalRoleList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoles", ctx, query)
	ret0, _ := ret[0].(*v1.GlobalRoleList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoles indicates an expected call of ListRoles.
func (mr *MockRoleReaderMockRecorder) ListRoles(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoles", reflect.TypeOf((*MockRoleReader)(nil).ListRoles), ctx, query)
}

// WatchRoles mocks base method.
func (m *MockRoleReader) WatchRoles(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRoles", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRoles indicates an expected call of WatchRoles.
func (mr *MockRoleReaderMockRecorder) WatchRoles(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRoles", reflect.TypeOf((*MockRoleReader)(nil).WatchRoles), ctx, query)
}

// MockRoleReaderEx is a mock of RoleReaderEx interface.
type MockRoleReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockRoleReaderExMockRecorder
}

// MockRoleReaderExMockRecorder is the mock recorder for MockRoleReaderEx.
type MockRoleReaderExMockRecorder struct {
	mock *MockRoleReaderEx
}

// NewMockRoleReaderEx creates a new mock instance.
func NewMockRoleReaderEx(ctrl *gomock.Controller) *MockRoleReaderEx {
	mock := &MockRoleReaderEx{ctrl: ctrl}
	mock.recorder = &MockRoleReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleReaderEx) EXPECT() *MockRoleReaderExMockRecorder {
	return m.recorder
}

// GetRoleEx mocks base method.
func (m *MockRoleReaderEx) GetRoleEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleEx indicates an expected call of GetRoleEx.
func (mr *MockRoleReaderExMockRecorder) GetRoleEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleEx", reflect.TypeOf((*MockRoleReaderEx)(nil).GetRoleEx), ctx, name, resourceVersion)
}

// GetRoleOfUser mocks base method.
func (m *MockRoleReaderEx) GetRoleOfUser(ctx context.Context, username string) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleOfUser", ctx, username)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleOfUser indicates an expected call of GetRoleOfUser.
func (mr *MockRoleReaderExMockRecorder) GetRoleOfUser(ctx, username interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleOfUser", reflect.TypeOf((*MockRoleReaderEx)(nil).GetRoleOfUser), ctx, username)
}

// ListRoleEx mocks base method.
func (m *MockRoleReaderEx) ListRoleEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleEx indicates an expected call of ListRoleEx.
func (mr *MockRoleReaderExMockRecorder) ListRoleEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleEx", reflect.TypeOf((*MockRoleReaderEx)(nil).ListRoleEx), ctx, query)
}

// MockRoleWriter is a mock of RoleWriter interface.
type MockRoleWriter struct {
	ctrl     *gomock.Controller
	recorder *MockRoleWriterMockRecorder
}

// MockRoleWriterMockRecorder is the mock recorder for MockRoleWriter.
type MockRoleWriterMockRecorder struct {
	mock *MockRoleWriter
}

// NewMockRoleWriter creates a new mock instance.
func NewMockRoleWriter(ctrl *gomock.Controller) *MockRoleWriter {
	mock := &MockRoleWriter{ctrl: ctrl}
	mock.recorder = &MockRoleWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleWriter) EXPECT() *MockRoleWriterMockRecorder {
	return m.recorder
}

// CreateRole mocks base method.
func (m *MockRoleWriter) CreateRole(ctx context.Context, role *v1.GlobalRole) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRole", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRole indicates an expected call of CreateRole.
func (mr *MockRoleWriterMockRecorder) CreateRole(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRole", reflect.TypeOf((*MockRoleWriter)(nil).CreateRole), ctx, role)
}

// DeleteRole mocks base method.
func (m *MockRoleWriter) DeleteRole(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRole", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRole indicates an expected call of DeleteRole.
func (mr *MockRoleWriterMockRecorder) DeleteRole(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRole", reflect.TypeOf((*MockRoleWriter)(nil).DeleteRole), ctx, name)
}

// UpdateRole mocks base method.
func (m *MockRoleWriter) UpdateRole(ctx context.Context, role *v1.GlobalRole) (*v1.GlobalRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRole", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRole indicates an expected call of UpdateRole.
func (mr *MockRoleWriterMockRecorder) UpdateRole(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRole", reflect.TypeOf((*MockRoleWriter)(nil).UpdateRole), ctx, role)
}

// MockRoleBindingReader is a mock of RoleBindingReader interface.
type MockRoleBindingReader struct {
	ctrl     *gomock.Controller
	recorder *MockRoleBindingReaderMockRecorder
}

// MockRoleBindingReaderMockRecorder is the mock recorder for MockRoleBindingReader.
type MockRoleBindingReaderMockRecorder struct {
	mock *MockRoleBindingReader
}

// NewMockRoleBindingReader creates a new mock instance.
func NewMockRoleBindingReader(ctrl *gomock.Controller) *MockRoleBindingReader {
	mock := &MockRoleBindingReader{ctrl: ctrl}
	mock.recorder = &MockRoleBindingReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleBindingReader) EXPECT() *MockRoleBindingReaderMockRecorder {
	return m.recorder
}

// GetRoleBinding mocks base method.
func (m *MockRoleBindingReader) GetRoleBinding(ctx context.Context, name string) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleBinding", ctx, name)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleBinding indicates an expected call of GetRoleBinding.
func (mr *MockRoleBindingReaderMockRecorder) GetRoleBinding(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleBinding", reflect.TypeOf((*MockRoleBindingReader)(nil).GetRoleBinding), ctx, name)
}

// GetRoleBindingEx mocks base method.
func (m *MockRoleBindingReader) GetRoleBindingEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleBindingEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleBindingEx indicates an expected call of GetRoleBindingEx.
func (mr *MockRoleBindingReaderMockRecorder) GetRoleBindingEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleBindingEx", reflect.TypeOf((*MockRoleBindingReader)(nil).GetRoleBindingEx), ctx, name, resourceVersion)
}

// ListRoleBindingEx mocks base method.
func (m *MockRoleBindingReader) ListRoleBindingEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleBindingEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleBindingEx indicates an expected call of ListRoleBindingEx.
func (mr *MockRoleBindingReaderMockRecorder) ListRoleBindingEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleBindingEx", reflect.TypeOf((*MockRoleBindingReader)(nil).ListRoleBindingEx), ctx, query)
}

// ListRoleBindings mocks base method.
func (m *MockRoleBindingReader) ListRoleBindings(ctx context.Context, query *query.Query) (*v1.GlobalRoleBindingList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleBindings", ctx, query)
	ret0, _ := ret[0].(*v1.GlobalRoleBindingList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleBindings indicates an expected call of ListRoleBindings.
func (mr *MockRoleBindingReaderMockRecorder) ListRoleBindings(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleBindings", reflect.TypeOf((*MockRoleBindingReader)(nil).ListRoleBindings), ctx, query)
}

// WatchRoleBindings mocks base method.
func (m *MockRoleBindingReader) WatchRoleBindings(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchRoleBindings", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchRoleBindings indicates an expected call of WatchRoleBindings.
func (mr *MockRoleBindingReaderMockRecorder) WatchRoleBindings(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchRoleBindings", reflect.TypeOf((*MockRoleBindingReader)(nil).WatchRoleBindings), ctx, query)
}

// MockRoleBindingReaderEx is a mock of RoleBindingReaderEx interface.
type MockRoleBindingReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockRoleBindingReaderExMockRecorder
}

// MockRoleBindingReaderExMockRecorder is the mock recorder for MockRoleBindingReaderEx.
type MockRoleBindingReaderExMockRecorder struct {
	mock *MockRoleBindingReaderEx
}

// NewMockRoleBindingReaderEx creates a new mock instance.
func NewMockRoleBindingReaderEx(ctrl *gomock.Controller) *MockRoleBindingReaderEx {
	mock := &MockRoleBindingReaderEx{ctrl: ctrl}
	mock.recorder = &MockRoleBindingReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleBindingReaderEx) EXPECT() *MockRoleBindingReaderExMockRecorder {
	return m.recorder
}

// GetRoleBindingEx mocks base method.
func (m *MockRoleBindingReaderEx) GetRoleBindingEx(ctx context.Context, name, resourceVersion string) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleBindingEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleBindingEx indicates an expected call of GetRoleBindingEx.
func (mr *MockRoleBindingReaderExMockRecorder) GetRoleBindingEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleBindingEx", reflect.TypeOf((*MockRoleBindingReaderEx)(nil).GetRoleBindingEx), ctx, name, resourceVersion)
}

// ListRoleBindingEx mocks base method.
func (m *MockRoleBindingReaderEx) ListRoleBindingEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoleBindingEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoleBindingEx indicates an expected call of ListRoleBindingEx.
func (mr *MockRoleBindingReaderExMockRecorder) ListRoleBindingEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoleBindingEx", reflect.TypeOf((*MockRoleBindingReaderEx)(nil).ListRoleBindingEx), ctx, query)
}

// MockRoleBindingWriter is a mock of RoleBindingWriter interface.
type MockRoleBindingWriter struct {
	ctrl     *gomock.Controller
	recorder *MockRoleBindingWriterMockRecorder
}

// MockRoleBindingWriterMockRecorder is the mock recorder for MockRoleBindingWriter.
type MockRoleBindingWriterMockRecorder struct {
	mock *MockRoleBindingWriter
}

// NewMockRoleBindingWriter creates a new mock instance.
func NewMockRoleBindingWriter(ctrl *gomock.Controller) *MockRoleBindingWriter {
	mock := &MockRoleBindingWriter{ctrl: ctrl}
	mock.recorder = &MockRoleBindingWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRoleBindingWriter) EXPECT() *MockRoleBindingWriterMockRecorder {
	return m.recorder
}

// CreateRoleBinding mocks base method.
func (m *MockRoleBindingWriter) CreateRoleBinding(ctx context.Context, role *v1.GlobalRoleBinding) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRoleBinding", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRoleBinding indicates an expected call of CreateRoleBinding.
func (mr *MockRoleBindingWriterMockRecorder) CreateRoleBinding(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRoleBinding", reflect.TypeOf((*MockRoleBindingWriter)(nil).CreateRoleBinding), ctx, role)
}

// DeleteRoleBinding mocks base method.
func (m *MockRoleBindingWriter) DeleteRoleBinding(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRoleBinding", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRoleBinding indicates an expected call of DeleteRoleBinding.
func (mr *MockRoleBindingWriterMockRecorder) DeleteRoleBinding(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRoleBinding", reflect.TypeOf((*MockRoleBindingWriter)(nil).DeleteRoleBinding), ctx, name)
}

// UpdateRoleBinding mocks base method.
func (m *MockRoleBindingWriter) UpdateRoleBinding(ctx context.Context, role *v1.GlobalRoleBinding) (*v1.GlobalRoleBinding, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRoleBinding", ctx, role)
	ret0, _ := ret[0].(*v1.GlobalRoleBinding)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRoleBinding indicates an expected call of UpdateRoleBinding.
func (mr *MockRoleBindingWriterMockRecorder) UpdateRoleBinding(ctx, role interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRoleBinding", reflect.TypeOf((*MockRoleBindingWriter)(nil).UpdateRoleBinding), ctx, role)
}

// MockTokenReader is a mock of TokenReader interface.
type MockTokenReader struct {
	ctrl     *gomock.Controller
	recorder *MockTokenReaderMockRecorder
}

// MockTokenReaderMockRecorder is the mock recorder for MockTokenReader.
type MockTokenReaderMockRecorder struct {
	mock *MockTokenReader
}

// NewMockTokenReader creates a new mock instance.
func NewMockTokenReader(ctrl *gomock.Controller) *MockTokenReader {
	mock := &MockTokenReader{ctrl: ctrl}
	mock.recorder = &MockTokenReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenReader) EXPECT() *MockTokenReaderMockRecorder {
	return m.recorder
}

// GetToken mocks base method.
func (m *MockTokenReader) GetToken(ctx context.Context, name string) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetToken", ctx, name)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetToken indicates an expected call of GetToken.
func (mr *MockTokenReaderMockRecorder) GetToken(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetToken", reflect.TypeOf((*MockTokenReader)(nil).GetToken), ctx, name)
}

// GetTokenEx mocks base method.
func (m *MockTokenReader) GetTokenEx(ctx context.Context, name, resourceVersion string) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTokenEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTokenEx indicates an expected call of GetTokenEx.
func (mr *MockTokenReaderMockRecorder) GetTokenEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTokenEx", reflect.TypeOf((*MockTokenReader)(nil).GetTokenEx), ctx, name, resourceVersion)
}

// ListTokenEx mocks base method.
func (m *MockTokenReader) ListTokenEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTokenEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTokenEx indicates an expected call of ListTokenEx.
func (mr *MockTokenReaderMockRecorder) ListTokenEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTokenEx", reflect.TypeOf((*MockTokenReader)(nil).ListTokenEx), ctx, query)
}

// ListTokens mocks base method.
func (m *MockTokenReader) ListTokens(ctx context.Context, query *query.Query) (*v1.TokenList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTokens", ctx, query)
	ret0, _ := ret[0].(*v1.TokenList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTokens indicates an expected call of ListTokens.
func (mr *MockTokenReaderMockRecorder) ListTokens(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTokens", reflect.TypeOf((*MockTokenReader)(nil).ListTokens), ctx, query)
}

// WatchTokens mocks base method.
func (m *MockTokenReader) WatchTokens(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchTokens", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchTokens indicates an expected call of WatchTokens.
func (mr *MockTokenReaderMockRecorder) WatchTokens(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchTokens", reflect.TypeOf((*MockTokenReader)(nil).WatchTokens), ctx, query)
}

// MockTokenReaderEx is a mock of TokenReaderEx interface.
type MockTokenReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockTokenReaderExMockRecorder
}

// MockTokenReaderExMockRecorder is the mock recorder for MockTokenReaderEx.
type MockTokenReaderExMockRecorder struct {
	mock *MockTokenReaderEx
}

// NewMockTokenReaderEx creates a new mock instance.
func NewMockTokenReaderEx(ctrl *gomock.Controller) *MockTokenReaderEx {
	mock := &MockTokenReaderEx{ctrl: ctrl}
	mock.recorder = &MockTokenReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenReaderEx) EXPECT() *MockTokenReaderExMockRecorder {
	return m.recorder
}

// GetTokenEx mocks base method.
func (m *MockTokenReaderEx) GetTokenEx(ctx context.Context, name, resourceVersion string) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTokenEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTokenEx indicates an expected call of GetTokenEx.
func (mr *MockTokenReaderExMockRecorder) GetTokenEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTokenEx", reflect.TypeOf((*MockTokenReaderEx)(nil).GetTokenEx), ctx, name, resourceVersion)
}

// ListTokenEx mocks base method.
func (m *MockTokenReaderEx) ListTokenEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTokenEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTokenEx indicates an expected call of ListTokenEx.
func (mr *MockTokenReaderExMockRecorder) ListTokenEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTokenEx", reflect.TypeOf((*MockTokenReaderEx)(nil).ListTokenEx), ctx, query)
}

// MockTokenWriter is a mock of TokenWriter interface.
type MockTokenWriter struct {
	ctrl     *gomock.Controller
	recorder *MockTokenWriterMockRecorder
}

// MockTokenWriterMockRecorder is the mock recorder for MockTokenWriter.
type MockTokenWriterMockRecorder struct {
	mock *MockTokenWriter
}

// NewMockTokenWriter creates a new mock instance.
func NewMockTokenWriter(ctrl *gomock.Controller) *MockTokenWriter {
	mock := &MockTokenWriter{ctrl: ctrl}
	mock.recorder = &MockTokenWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenWriter) EXPECT() *MockTokenWriterMockRecorder {
	return m.recorder
}

// CreateToken mocks base method.
func (m *MockTokenWriter) CreateToken(ctx context.Context, token *v1.Token) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateToken", ctx, token)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateToken indicates an expected call of CreateToken.
func (mr *MockTokenWriterMockRecorder) CreateToken(ctx, token interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateToken", reflect.TypeOf((*MockTokenWriter)(nil).CreateToken), ctx, token)
}

// DeleteToken mocks base method.
func (m *MockTokenWriter) DeleteToken(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteToken", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteToken indicates an expected call of DeleteToken.
func (mr *MockTokenWriterMockRecorder) DeleteToken(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteToken", reflect.TypeOf((*MockTokenWriter)(nil).DeleteToken), ctx, name)
}

// DeleteTokenCollection mocks base method.
func (m *MockTokenWriter) DeleteTokenCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTokenCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTokenCollection indicates an expected call of DeleteTokenCollection.
func (mr *MockTokenWriterMockRecorder) DeleteTokenCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTokenCollection", reflect.TypeOf((*MockTokenWriter)(nil).DeleteTokenCollection), ctx, query)
}

// UpdateToken mocks base method.
func (m *MockTokenWriter) UpdateToken(ctx context.Context, token *v1.Token) (*v1.Token, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateToken", ctx, token)
	ret0, _ := ret[0].(*v1.Token)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateToken indicates an expected call of UpdateToken.
func (mr *MockTokenWriterMockRecorder) UpdateToken(ctx, token interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateToken", reflect.TypeOf((*MockTokenWriter)(nil).UpdateToken), ctx, token)
}

// MockLoginRecordReader is a mock of LoginRecordReader interface.
type MockLoginRecordReader struct {
	ctrl     *gomock.Controller
	recorder *MockLoginRecordReaderMockRecorder
}

// MockLoginRecordReaderMockRecorder is the mock recorder for MockLoginRecordReader.
type MockLoginRecordReaderMockRecorder struct {
	mock *MockLoginRecordReader
}

// NewMockLoginRecordReader creates a new mock instance.
func NewMockLoginRecordReader(ctrl *gomock.Controller) *MockLoginRecordReader {
	mock := &MockLoginRecordReader{ctrl: ctrl}
	mock.recorder = &MockLoginRecordReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoginRecordReader) EXPECT() *MockLoginRecordReaderMockRecorder {
	return m.recorder
}

// GetLoginRecord mocks base method.
func (m *MockLoginRecordReader) GetLoginRecord(ctx context.Context, name string) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoginRecord", ctx, name)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoginRecord indicates an expected call of GetLoginRecord.
func (mr *MockLoginRecordReaderMockRecorder) GetLoginRecord(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoginRecord", reflect.TypeOf((*MockLoginRecordReader)(nil).GetLoginRecord), ctx, name)
}

// GetLoginRecordEx mocks base method.
func (m *MockLoginRecordReader) GetLoginRecordEx(ctx context.Context, name, resourceVersion string) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoginRecordEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoginRecordEx indicates an expected call of GetLoginRecordEx.
func (mr *MockLoginRecordReaderMockRecorder) GetLoginRecordEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoginRecordEx", reflect.TypeOf((*MockLoginRecordReader)(nil).GetLoginRecordEx), ctx, name, resourceVersion)
}

// ListLoginRecordEx mocks base method.
func (m *MockLoginRecordReader) ListLoginRecordEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLoginRecordEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLoginRecordEx indicates an expected call of ListLoginRecordEx.
func (mr *MockLoginRecordReaderMockRecorder) ListLoginRecordEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLoginRecordEx", reflect.TypeOf((*MockLoginRecordReader)(nil).ListLoginRecordEx), ctx, query)
}

// ListLoginRecords mocks base method.
func (m *MockLoginRecordReader) ListLoginRecords(ctx context.Context, query *query.Query) (*v1.LoginRecordList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLoginRecords", ctx, query)
	ret0, _ := ret[0].(*v1.LoginRecordList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLoginRecords indicates an expected call of ListLoginRecords.
func (mr *MockLoginRecordReaderMockRecorder) ListLoginRecords(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLoginRecords", reflect.TypeOf((*MockLoginRecordReader)(nil).ListLoginRecords), ctx, query)
}

// WatchLoginRecords mocks base method.
func (m *MockLoginRecordReader) WatchLoginRecords(ctx context.Context, query *query.Query) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WatchLoginRecords", ctx, query)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WatchLoginRecords indicates an expected call of WatchLoginRecords.
func (mr *MockLoginRecordReaderMockRecorder) WatchLoginRecords(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WatchLoginRecords", reflect.TypeOf((*MockLoginRecordReader)(nil).WatchLoginRecords), ctx, query)
}

// MockLoginRecordReaderEx is a mock of LoginRecordReaderEx interface.
type MockLoginRecordReaderEx struct {
	ctrl     *gomock.Controller
	recorder *MockLoginRecordReaderExMockRecorder
}

// MockLoginRecordReaderExMockRecorder is the mock recorder for MockLoginRecordReaderEx.
type MockLoginRecordReaderExMockRecorder struct {
	mock *MockLoginRecordReaderEx
}

// NewMockLoginRecordReaderEx creates a new mock instance.
func NewMockLoginRecordReaderEx(ctrl *gomock.Controller) *MockLoginRecordReaderEx {
	mock := &MockLoginRecordReaderEx{ctrl: ctrl}
	mock.recorder = &MockLoginRecordReaderExMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoginRecordReaderEx) EXPECT() *MockLoginRecordReaderExMockRecorder {
	return m.recorder
}

// GetLoginRecordEx mocks base method.
func (m *MockLoginRecordReaderEx) GetLoginRecordEx(ctx context.Context, name, resourceVersion string) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLoginRecordEx", ctx, name, resourceVersion)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLoginRecordEx indicates an expected call of GetLoginRecordEx.
func (mr *MockLoginRecordReaderExMockRecorder) GetLoginRecordEx(ctx, name, resourceVersion interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLoginRecordEx", reflect.TypeOf((*MockLoginRecordReaderEx)(nil).GetLoginRecordEx), ctx, name, resourceVersion)
}

// ListLoginRecordEx mocks base method.
func (m *MockLoginRecordReaderEx) ListLoginRecordEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListLoginRecordEx", ctx, query)
	ret0, _ := ret[0].(*models.PageableResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLoginRecordEx indicates an expected call of ListLoginRecordEx.
func (mr *MockLoginRecordReaderExMockRecorder) ListLoginRecordEx(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLoginRecordEx", reflect.TypeOf((*MockLoginRecordReaderEx)(nil).ListLoginRecordEx), ctx, query)
}

// MockLoginRecordWriter is a mock of LoginRecordWriter interface.
type MockLoginRecordWriter struct {
	ctrl     *gomock.Controller
	recorder *MockLoginRecordWriterMockRecorder
}

// MockLoginRecordWriterMockRecorder is the mock recorder for MockLoginRecordWriter.
type MockLoginRecordWriterMockRecorder struct {
	mock *MockLoginRecordWriter
}

// NewMockLoginRecordWriter creates a new mock instance.
func NewMockLoginRecordWriter(ctrl *gomock.Controller) *MockLoginRecordWriter {
	mock := &MockLoginRecordWriter{ctrl: ctrl}
	mock.recorder = &MockLoginRecordWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLoginRecordWriter) EXPECT() *MockLoginRecordWriterMockRecorder {
	return m.recorder
}

// CreateLoginRecord mocks base method.
func (m *MockLoginRecordWriter) CreateLoginRecord(ctx context.Context, record *v1.LoginRecord) (*v1.LoginRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateLoginRecord", ctx, record)
	ret0, _ := ret[0].(*v1.LoginRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateLoginRecord indicates an expected call of CreateLoginRecord.
func (mr *MockLoginRecordWriterMockRecorder) CreateLoginRecord(ctx, record interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateLoginRecord", reflect.TypeOf((*MockLoginRecordWriter)(nil).CreateLoginRecord), ctx, record)
}

// DeleteLoginRecord mocks base method.
func (m *MockLoginRecordWriter) DeleteLoginRecord(ctx context.Context, name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLoginRecord", ctx, name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteLoginRecord indicates an expected call of DeleteLoginRecord.
func (mr *MockLoginRecordWriterMockRecorder) DeleteLoginRecord(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLoginRecord", reflect.TypeOf((*MockLoginRecordWriter)(nil).DeleteLoginRecord), ctx, name)
}

// DeleteLoginRecordCollection mocks base method.
func (m *MockLoginRecordWriter) DeleteLoginRecordCollection(ctx context.Context, query *query.Query) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteLoginRecordCollection", ctx, query)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteLoginRecordCollection indicates an expected call of DeleteLoginRecordCollection.
func (mr *MockLoginRecordWriterMockRecorder) DeleteLoginRecordCollection(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteLoginRecordCollection", reflect.TypeOf((*MockLoginRecordWriter)(nil).DeleteLoginRecordCollection), ctx, query)
}
