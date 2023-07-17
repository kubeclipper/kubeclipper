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

package iam

import (
	"context"
	"sort"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/pkg/errors"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"

	"k8s.io/apimachinery/pkg/runtime"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeclipper/kubeclipper/pkg/models"
	"github.com/kubeclipper/kubeclipper/pkg/query"
)

var _ Operator = (*iamOperator)(nil)

type iamOperator struct {
	userStorage        rest.StandardStorage
	roleStorage        rest.StandardStorage
	roleBindingStorage rest.StandardStorage
	tokenStorage       rest.StandardStorage
	loginRecordStorage rest.StandardStorage
}

func NewOperator(userStorage rest.StandardStorage, roleStorage rest.StandardStorage,
	roleBindingStorage rest.StandardStorage, tokenStorage rest.StandardStorage, loginRecordStorage rest.StandardStorage) Operator {
	return &iamOperator{
		userStorage:        userStorage,
		roleStorage:        roleStorage,
		roleBindingStorage: roleBindingStorage,
		tokenStorage:       tokenStorage,
		loginRecordStorage: loginRecordStorage,
	}
}

func (i *iamOperator) ListUsers(ctx context.Context, query *query.Query) (*iamv1.UserList, error) {
	list, err := models.List(ctx, i.userStorage, query)
	if err != nil {
		return nil, err
	}
	result, _ := list.(*iamv1.UserList)
	return result, nil
}

func (i *iamOperator) WatchUsers(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, i.userStorage, query)
}

func (i *iamOperator) GetUser(ctx context.Context, name string) (*iamv1.User, error) {
	return i.GetUserEx(ctx, name, "", false, false)
}

func (i *iamOperator) CreateUser(ctx context.Context, user *iamv1.User) (*iamv1.User, error) {
	ctx = genericapirequest.WithNamespace(ctx, user.Namespace)
	obj, err := i.userStorage.Create(ctx, user, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.User), nil
}

func (i *iamOperator) UpdateUser(ctx context.Context, user *iamv1.User) (*iamv1.User, error) {
	ctx = genericapirequest.WithNamespace(ctx, user.Namespace)
	obj, wasCreated, err := i.userStorage.Update(ctx, user.Name, rest.DefaultUpdatedObjectInfo(user),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if wasCreated {
		logger.Debug("user not exist, use create instead of update", zap.String("user", user.Name))
	}
	return obj.(*iamv1.User), nil
}

func (i *iamOperator) DeleteUser(ctx context.Context, name string) error {
	var err error
	_, _, err = i.userStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (i *iamOperator) GetUserEx(ctx context.Context, name string, resourceVersion string, desensitization bool, includeRole bool) (*iamv1.User, error) {
	var mutatingFunc func(obj runtime.Object) runtime.Object
	if desensitization || includeRole {
		mutatingFunc = func(obj runtime.Object) runtime.Object {
			user, ok := obj.(*iamv1.User)
			if !ok {
				return obj
			}

			if desensitization {
				DesensitizationUserPassword(user)
			}
			if includeRole {
				role, err := i.GetRoleOfUser(ctx, user.Name)
				if err != nil || role == nil {
					logger.Error("failed to get role of user", zap.String("usernmae", user.Name), zap.Error(err))
				} else {
					appendGlobalRoleAnnotation(user, role.Name)
				}
			}
			return user
		}
	}
	obj, err := models.GetV2(ctx, i.userStorage, name, resourceVersion, mutatingFunc)
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.User), nil
}

func appendGlobalRoleAnnotation(user *iamv1.User, globalRole string) {
	if user.Annotations == nil {
		user.Annotations = make(map[string]string)
	}
	user.Annotations[common.RoleAnnotation] = globalRole
}

func (i *iamOperator) GetRoleOfUser(ctx context.Context, username string) (*iamv1.GlobalRole, error) {
	list, err := i.roleBindingStorage.List(ctx, &metainternalversion.ListOptions{
		LabelSelector:        labels.Everything(),
		FieldSelector:        fields.Everything(),
		ResourceVersion:      "0",
		ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
	})
	// TODO: make error standard
	if err != nil {
		return nil, errors.WithMessage(err, "list roleBinding")
	}
	rolebindings, _ := list.(*iamv1.GlobalRoleBindingList)
	result := make([]iamv1.GlobalRoleBinding, 0)
	for idx, item := range rolebindings.Items {
		if contains(item.Subjects, username, nil) {
			result = append(result, rolebindings.Items[idx])
		}
	}
	// Usually, only one globalRoleBinding will be found which is created by front-end
	// in v4.3.2 we add tenant,and will add a internal role to user for view some global resource
	// so,we need exclude internal role
	for _, v := range result {
		if v.Annotations != nil && v.Annotations[common.AnnotationHidden] == "true" {
			continue
		}
		return i.GetRoleEx(ctx, v.RoleRef.Name, "0")
	}
	return nil, nil
}

func contains(subjects []rbacv1.Subject, username string, groups []string) bool {
	// if username is nil means list all role bindings
	if username == "" {
		return true
	}
	for _, subject := range subjects {
		if subject.Kind == rbacv1.UserKind && subject.Name == username {
			return true
		}
		if subject.Kind == rbacv1.GroupKind && sliceutil.HasString(groups, subject.Name) {
			return true
		}
	}
	return false
}

func (i *iamOperator) ListUserEx(ctx context.Context, query *query.Query, desensitization bool, includeRole bool) (*models.PageableResponse, error) {
	mutatingFunc := func(obj runtime.Object) runtime.Object {
		user, ok := obj.(*iamv1.User)
		if !ok {
			return obj
		}

		if desensitization {
			DesensitizationUserPassword(user)
		}
		if includeRole {
			role, err := i.GetRoleOfUser(ctx, user.Name)
			if err != nil || role == nil {
				logger.Error("failed to get role of user", zap.String("usernmae", user.Name), zap.Error(err))
			} else {
				appendGlobalRoleAnnotation(user, role.Name)
			}
		}
		return user
	}

	return models.ListExV2(ctx, i.userStorage, query, UserFuzzyFilter, nil, mutatingFunc)
}

// ListUsersByRole
// TODO: Need Refactor
func (i *iamOperator) ListUsersByRole(ctx context.Context, query *query.Query, role string) (*models.PageableResponse, error) {
	globalrole, err := i.GetRoleBindingEx(ctx, role, "0")
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			return &models.PageableResponse{Items: []interface{}{}, TotalCount: 0}, nil
		}
		return nil, err
	}
	var users []iamv1.User
	userMap := make(map[string]struct{})
	queryByLabelField := false
	if query.LabelSelector != "" || query.FieldSelector != "" {
		queryByLabelField = true
		list, err := i.userStorage.List(ctx, &metainternalversion.ListOptions{
			LabelSelector:        query.GetLabelSelector(),
			FieldSelector:        query.GetFieldSelector(),
			ResourceVersion:      "0",
			ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
		})
		if err != nil {
			return nil, err
		}
		result, _ := list.(*iamv1.UserList)
		if len(result.Items) > 0 {
			for j := range result.Items {
				userMap[result.Items[j].Name] = struct{}{}
			}
		}
	}
	for _, subject := range globalrole.Subjects {
		if subject.Kind == rbacv1.UserKind {
			user, err := i.GetUserEx(ctx, subject.Name, "0", true, false)
			if err != nil {
				if apimachineryErrors.IsNotFound(err) {
					continue
				}
				return nil, err
			}
			if queryByLabelField {
				if _, ok := userMap[user.Name]; !ok {
					continue
				}
			}
			if user.Annotations == nil {
				user.Annotations = make(map[string]string)
			}
			user.Annotations[common.RoleAnnotation] = role
			users = append(users, *user)
		}
	}
	if query.Reverse {
		sort.Sort(sort.Reverse(userList(users)))
	} else {
		sort.Sort(userList(users))
	}

	totalCount := len(users)
	start, end := query.Pagination.GetValidPagination(totalCount)
	users = users[start:end]
	items := make([]interface{}, 0, len(users))
	for index := range users {
		items = append(items, users[index])
	}
	return &models.PageableResponse{
		Items:      items,
		TotalCount: totalCount,
	}, nil
}

func (i *iamOperator) ListRoles(ctx context.Context, query *query.Query) (*iamv1.GlobalRoleList, error) {
	list, err := models.List(ctx, i.roleStorage, query)
	if err != nil {
		return nil, err
	}
	return list.(*iamv1.GlobalRoleList), nil
}

func (i *iamOperator) WatchRoles(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, i.roleStorage, query)
}

func (i *iamOperator) GetRole(ctx context.Context, name string) (*iamv1.GlobalRole, error) {
	return i.GetRoleEx(ctx, name, "")
}

func (i *iamOperator) CreateRole(ctx context.Context, role *iamv1.GlobalRole) (*iamv1.GlobalRole, error) {
	ctx = genericapirequest.WithNamespace(ctx, role.Namespace)
	obj, err := i.roleStorage.Create(ctx, role, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.GlobalRole), nil
}

func (i *iamOperator) UpdateRole(ctx context.Context, role *iamv1.GlobalRole) (*iamv1.GlobalRole, error) {
	ctx = genericapirequest.WithNamespace(ctx, role.Namespace)
	obj, wasCreated, err := i.roleStorage.Update(ctx, role.Name, rest.DefaultUpdatedObjectInfo(role),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if wasCreated {
		logger.Debug("role not exist, use create instead of update", zap.String("role", role.Name))
	}
	return obj.(*iamv1.GlobalRole), nil
}

func (i *iamOperator) DeleteRole(ctx context.Context, name string) error {
	var err error
	_, _, err = i.roleStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (i *iamOperator) GetRoleEx(ctx context.Context, name string, resourceVersion string) (*iamv1.GlobalRole, error) {
	role, err := models.GetV2(ctx, i.roleStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return role.(*iamv1.GlobalRole), nil
}

func (i *iamOperator) ListRoleEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, i.roleStorage, query, i.roleFuzzyFilter, nil, nil)
}

func (i *iamOperator) ListRoleBindings(ctx context.Context, query *query.Query) (*iamv1.GlobalRoleBindingList, error) {
	list, err := models.List(ctx, i.roleBindingStorage, query)
	if err != nil {
		return nil, err
	}
	return list.(*iamv1.GlobalRoleBindingList), nil
}

func (i *iamOperator) WatchRoleBindings(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, i.roleBindingStorage, query)
}

func (i *iamOperator) GetRoleBinding(ctx context.Context, name string) (*iamv1.GlobalRoleBinding, error) {
	return i.GetRoleBindingEx(ctx, name, "")
}

func (i *iamOperator) CreateRoleBinding(ctx context.Context, role *iamv1.GlobalRoleBinding) (*iamv1.GlobalRoleBinding, error) {
	ctx = genericapirequest.WithNamespace(ctx, role.Namespace)
	obj, err := i.roleBindingStorage.Create(ctx, role, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.GlobalRoleBinding), nil
}

func (i *iamOperator) UpdateRoleBinding(ctx context.Context, role *iamv1.GlobalRoleBinding) (*iamv1.GlobalRoleBinding, error) {
	ctx = genericapirequest.WithNamespace(ctx, role.Namespace)
	obj, wasCreated, err := i.roleBindingStorage.Update(ctx, role.Name, rest.DefaultUpdatedObjectInfo(role),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if wasCreated {
		logger.Debug("role binding not exist, use create instead of update", zap.String("role", role.Name))
	}
	return obj.(*iamv1.GlobalRoleBinding), nil
}

func (i *iamOperator) DeleteRoleBinding(ctx context.Context, name string) error {
	var err error
	_, _, err = i.roleBindingStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (i *iamOperator) GetRoleBindingEx(ctx context.Context, name string, resourceVersion string) (*iamv1.GlobalRoleBinding, error) {
	obj, err := models.GetV2(ctx, i.roleBindingStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.GlobalRoleBinding), nil
}

func (i *iamOperator) ListRoleBindingEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, i.roleBindingStorage, query, i.roleBindingFuzzyFilter, nil, nil)
}

func (i *iamOperator) ListTokens(ctx context.Context, query *query.Query) (*iamv1.TokenList, error) {
	list, err := models.List(ctx, i.tokenStorage, query)
	if err != nil {
		return nil, err
	}
	list.GetObjectKind().SetGroupVersionKind(iamv1.SchemeGroupVersion.WithKind("TokenList"))
	return list.(*iamv1.TokenList), nil
}

func (i *iamOperator) WatchTokens(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, i.tokenStorage, query)
}

func (i *iamOperator) GetToken(ctx context.Context, name string) (*iamv1.Token, error) {
	return i.GetTokenEx(ctx, name, "")
}

func (i *iamOperator) CreateToken(ctx context.Context, token *iamv1.Token) (*iamv1.Token, error) {
	ctx = genericapirequest.WithNamespace(ctx, token.Namespace)
	obj, err := i.tokenStorage.Create(ctx, token, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.Token), nil
}

func (i *iamOperator) UpdateToken(ctx context.Context, token *iamv1.Token) (*iamv1.Token, error) {
	ctx = genericapirequest.WithNamespace(ctx, token.Namespace)
	obj, wasCreated, err := i.tokenStorage.Update(ctx, token.Name, rest.DefaultUpdatedObjectInfo(token),
		nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if wasCreated {
		logger.Debug("token not exist, use create instead of update", zap.String("role", token.Name))
	}
	return obj.(*iamv1.Token), nil
}

func (i *iamOperator) DeleteToken(ctx context.Context, name string) error {
	var err error
	_, _, err = i.tokenStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (i *iamOperator) DeleteTokenCollection(ctx context.Context, query *query.Query) error {
	if _, err := i.tokenStorage.DeleteCollection(ctx, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{}, &metainternalversion.ListOptions{
		LabelSelector: query.GetLabelSelector(),
		FieldSelector: query.GetFieldSelector(),
	}); err != nil {
		return err
	}
	return nil
}

func (i *iamOperator) GetTokenEx(ctx context.Context, name string, resourceVersion string) (*iamv1.Token, error) {
	token, err := models.GetV2(ctx, i.tokenStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return token.(*iamv1.Token), nil
}

func (i *iamOperator) ListTokenEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, i.tokenStorage, query, i.tokenFuzzyFilter, nil, nil)
}

func (i *iamOperator) ListLoginRecords(ctx context.Context, query *query.Query) (*iamv1.LoginRecordList, error) {
	list, err := models.List(ctx, i.loginRecordStorage, query)
	if err != nil {
		return nil, err
	}
	return list.(*iamv1.LoginRecordList), nil
}

func (i *iamOperator) WatchLoginRecords(ctx context.Context, query *query.Query) (watch.Interface, error) {
	return models.Watch(ctx, i.loginRecordStorage, query)
}

func (i *iamOperator) GetLoginRecord(ctx context.Context, name string) (*iamv1.LoginRecord, error) {
	return i.GetLoginRecordEx(ctx, name, "")
}

func (i *iamOperator) CreateLoginRecord(ctx context.Context, record *iamv1.LoginRecord) (*iamv1.LoginRecord, error) {
	obj, err := i.loginRecordStorage.Create(ctx, record, nil, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*iamv1.LoginRecord), nil
}

func (i *iamOperator) DeleteLoginRecord(ctx context.Context, name string) error {
	var err error
	_, _, err = i.loginRecordStorage.Delete(ctx, name, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{})
	return err
}

func (i *iamOperator) DeleteLoginRecordCollection(ctx context.Context, query *query.Query) error {
	if _, err := i.loginRecordStorage.DeleteCollection(ctx, func(ctx context.Context, obj runtime.Object) error {
		return nil
	}, &metav1.DeleteOptions{}, &metainternalversion.ListOptions{
		LabelSelector: query.GetLabelSelector(),
		FieldSelector: query.GetFieldSelector(),
	}); err != nil {
		return err
	}
	return nil
}

func (i *iamOperator) ListLoginRecordEx(ctx context.Context, query *query.Query) (*models.PageableResponse, error) {
	return models.ListExV2(ctx, i.loginRecordStorage, query, i.loginRecordFuzzyFilter, nil, nil)
}

func (i *iamOperator) GetLoginRecordEx(ctx context.Context, name string, resourceVersion string) (*iamv1.LoginRecord, error) {
	token, err := models.GetV2(ctx, i.loginRecordStorage, name, resourceVersion, nil)
	if err != nil {
		return nil, err
	}
	return token.(*iamv1.LoginRecord), nil
}

// UserFuzzyFilter func for filter user.
func UserFuzzyFilter(obj runtime.Object, q *query.Query) []runtime.Object {
	users, ok := obj.(*iamv1.UserList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(users.Items))
	for index, user := range users.Items {
		selected := true
		for k, v := range q.FuzzySearch {
			if !models.ObjectMetaFilter(user.ObjectMeta, k, v) {
				selected = false
			}
		}
		if selected {
			objs = append(objs, &users.Items[index])
		}
	}
	return objs
}

func (i *iamOperator) roleFuzzyFilter(obj runtime.Object, q *query.Query) []runtime.Object {
	roles, ok := obj.(*iamv1.GlobalRoleList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(roles.Items))
	for index, role := range roles.Items {
		selected := true
		for k, v := range q.FuzzySearch {
			if !models.ObjectMetaFilter(role.ObjectMeta, k, v) {
				selected = false
			}
		}
		if selected {
			objs = append(objs, &roles.Items[index])
		}
	}
	return objs
}

func (i *iamOperator) roleBindingFuzzyFilter(obj runtime.Object, _ *query.Query) []runtime.Object {
	rbs, ok := obj.(*iamv1.GlobalRoleBindingList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(rbs.Items))
	for index := range rbs.Items {
		objs = append(objs, &rbs.Items[index])
	}
	return objs
}

func (i *iamOperator) loginRecordFuzzyFilter(obj runtime.Object, _ *query.Query) []runtime.Object {
	records, ok := obj.(*iamv1.LoginRecordList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(records.Items))
	for index := range records.Items {
		objs = append(objs, &records.Items[index])
	}
	return objs
}

func (i *iamOperator) tokenFuzzyFilter(obj runtime.Object, _ *query.Query) []runtime.Object {
	tokens, ok := obj.(*iamv1.TokenList)
	if !ok {
		return nil
	}
	objs := make([]runtime.Object, 0, len(tokens.Items))
	for index := range tokens.Items {
		objs = append(objs, &tokens.Items[index])
	}
	return objs
}

// var _ Operator = (*iamOperatorV2)(nil)
// type iamOperatorV2 struct {
// 	userModel models.Operator[*iamv1.User, *iamv1.UserList]
// }
