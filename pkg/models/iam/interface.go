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

//go:generate mockgen -destination mock/mock_iam.go -source interface.go Operator

package iam

import (
	"context"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
)

type Operator interface {
	UserReader
	UserWriter

	RoleReader
	RoleWriter

	RoleBindingReader
	RoleBindingWriter

	TokenReader
	TokenWriter

	LoginRecordReader
	LoginRecordWriter
}

type UserReader interface {
	ListUsers(ctx context.Context, query *query.Query) (*iamv1.UserList, error)
	WatchUsers(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetUser(ctx context.Context, name string) (*iamv1.User, error)
	UserReaderEx
}

type UserReaderEx interface {
	GetUserEx(ctx context.Context, name string, resourceVersion string, desensitization bool, includeRole bool) (*iamv1.User, error)
	ListUserEx(ctx context.Context, query *query.Query, desensitization bool, includeRole bool) (*models.PageableResponse, error)
	ListUsersByRole(ctx context.Context, query *query.Query, role string) (*models.PageableResponse, error)
}

type UserWriter interface {
	CreateUser(ctx context.Context, user *iamv1.User) (*iamv1.User, error)
	UpdateUser(ctx context.Context, user *iamv1.User) (*iamv1.User, error)
	DeleteUser(ctx context.Context, name string) error
}

type RoleReader interface {
	ListRoles(ctx context.Context, query *query.Query) (*iamv1.GlobalRoleList, error)
	WatchRoles(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetRole(ctx context.Context, name string) (*iamv1.GlobalRole, error)
	RoleReaderEx
}

type RoleReaderEx interface {
	GetRoleEx(ctx context.Context, name string, resourceVersion string) (*iamv1.GlobalRole, error)
	ListRoleEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
	GetRoleOfUser(ctx context.Context, username string) (*iamv1.GlobalRole, error)
}

type RoleWriter interface {
	CreateRole(ctx context.Context, role *iamv1.GlobalRole) (*iamv1.GlobalRole, error)
	UpdateRole(ctx context.Context, role *iamv1.GlobalRole) (*iamv1.GlobalRole, error)
	DeleteRole(ctx context.Context, name string) error
}

type RoleBindingReader interface {
	ListRoleBindings(ctx context.Context, query *query.Query) (*iamv1.GlobalRoleBindingList, error)
	WatchRoleBindings(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetRoleBinding(ctx context.Context, name string) (*iamv1.GlobalRoleBinding, error)
	RoleBindingReaderEx
}

type RoleBindingReaderEx interface {
	GetRoleBindingEx(ctx context.Context, name string, resourceVersion string) (*iamv1.GlobalRoleBinding, error)
	ListRoleBindingEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type RoleBindingWriter interface {
	CreateRoleBinding(ctx context.Context, role *iamv1.GlobalRoleBinding) (*iamv1.GlobalRoleBinding, error)
	UpdateRoleBinding(ctx context.Context, role *iamv1.GlobalRoleBinding) (*iamv1.GlobalRoleBinding, error)
	DeleteRoleBinding(ctx context.Context, name string) error
}

type TokenReader interface {
	ListTokens(ctx context.Context, query *query.Query) (*iamv1.TokenList, error)
	WatchTokens(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetToken(ctx context.Context, name string) (*iamv1.Token, error)
	TokenReaderEx
}

type TokenReaderEx interface {
	GetTokenEx(ctx context.Context, name string, resourceVersion string) (*iamv1.Token, error)
	ListTokenEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
}

type TokenWriter interface {
	CreateToken(ctx context.Context, token *iamv1.Token) (*iamv1.Token, error)
	UpdateToken(ctx context.Context, token *iamv1.Token) (*iamv1.Token, error)
	DeleteToken(ctx context.Context, name string) error
	DeleteTokenCollection(ctx context.Context, query *query.Query) error
}

type LoginRecordReader interface {
	ListLoginRecords(ctx context.Context, query *query.Query) (*iamv1.LoginRecordList, error)
	WatchLoginRecords(ctx context.Context, query *query.Query) (watch.Interface, error)
	GetLoginRecord(ctx context.Context, name string) (*iamv1.LoginRecord, error)
	LoginRecordReaderEx
}

type LoginRecordReaderEx interface {
	ListLoginRecordEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error)
	GetLoginRecordEx(ctx context.Context, name string, resourceVersion string) (*iamv1.LoginRecord, error)
}

type LoginRecordWriter interface {
	CreateLoginRecord(ctx context.Context, record *iamv1.LoginRecord) (*iamv1.LoginRecord, error)
	DeleteLoginRecord(ctx context.Context, name string) error
	DeleteLoginRecordCollection(ctx context.Context, query *query.Query) error
}
