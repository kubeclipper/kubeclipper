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

package rbac

import (
	"context"
	"net/http"
	"testing"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	v12 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/server/request"

	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/kubeclipper/kubeclipper/pkg/authorization/authorizer"

	"github.com/golang/mock/gomock"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	iammock "github.com/kubeclipper/kubeclipper/pkg/models/iam/mock"
)

var (
	// non resource api
	reqGetMetrics, _ = http.NewRequest("GET", "/metrics", nil)
	reqLogin, _      = http.NewRequest("POST", "/oauth/login", nil)
	reqLogout, _     = http.NewRequest("GET", "/oauth/logout", nil)
	reqOauth, _      = http.NewRequest("POST", "/oauth/token", nil)

	// cluster API
	reqListClusters, _   = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/clusters", nil)
	reqCreateClusters, _ = http.NewRequest("POST", "/api/core.kubeclipper.io/v1/clusters", nil)
	reqGetCluster, _     = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/clusters/cluster1", nil)
	reqDeleteClusters, _ = http.NewRequest("DELETE", "/api/core.kubeclipper.io/v1/clusters/cluster1", nil)

	// operation api
	reqListOperations, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/operations", nil)

	// node api
	reqListNodes, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes", nil)
	reqGetNode, _   = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/nodes/node1", nil)

	// region api
	reqListRegions, _ = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/regions", nil)
	reqGetRegion, _   = http.NewRequest("GET", "/api/core.kubeclipper.io/v1/regions/rg1", nil)

	// iam role api
	reqListRoles, _   = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/roles", nil)
	reqCreateRoles, _ = http.NewRequest("POST", "/api/iam.kubeclipper.io/v1/roles", nil)
	reqGetRole, _     = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/roles/role1", nil)
	reqDeleteRoles, _ = http.NewRequest("DELETE", "/api/iam.kubeclipper.io/v1/roles/role1", nil)
	reqUpdateRoles, _ = http.NewRequest("PUT", "/api/iam.kubeclipper.io/v1/roles/role1", nil)

	// iam user api
	reqListUsers, _   = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/users", nil)
	reqCreateUsers, _ = http.NewRequest("POST", "/api/iam.kubeclipper.io/v1/users", nil)
	reqGetUser, _     = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/users/user1", nil)
	reqDeleteUsers, _ = http.NewRequest("DELETE", "/api/iam.kubeclipper.io/v1/users/user1", nil)
	reqUpdateUsers, _ = http.NewRequest("PUT", "/api/iam.kubeclipper.io/v1/users/user1", nil)

	reqAddNodeToCluster, _   = http.NewRequest("POST", "/api/core.kubeclipper.io/v1/clusters/cluster1/nodes", nil)
	reqAddPluginToCluster, _ = http.NewRequest("POST", "/api/core.kubeclipper.io/v1/clusters/cluster1/plugins", nil)
	reqDelPluginToCluster, _ = http.NewRequest("DELETE", "/api/core.kubeclipper.io/v1/clusters/cluster1/plugins", nil)

	reqGetUserAdmin, _          = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/users/admin", nil)
	reqGetUserClusterManager, _ = http.NewRequest("GET", "/api/iam.kubeclipper.io/v1/users/clustermanager", nil)

	reqChangeAdminPassword, _          = http.NewRequest("PUT", "/api/iam.kubeclipper.io/v1/users/admin/password", nil)
	reqChangeClusterManagerPassword, _ = http.NewRequest("PUT", "/api/iam.kubeclipper.io/v1/users/clustermanager/password", nil)
)

var (
	userAdmin = &user.DefaultInfo{
		Name:   "admin",
		UID:    "",
		Groups: []string{user.AllAuthenticated},
		Extra:  nil,
	}
	userAnonymous = &user.DefaultInfo{
		Name:   user.Anonymous,
		UID:    "",
		Groups: []string{user.AllUnauthenticated},
	}
	userIamManager = &user.DefaultInfo{
		Name:   "iammanager",
		Groups: []string{user.AllAuthenticated},
	}
	userClusterManager = &user.DefaultInfo{
		Name:   "clustermanager",
		Groups: []string{user.AllAuthenticated},
	}
	userPlatformView = &user.DefaultInfo{
		Name:   "view",
		UID:    "",
		Groups: []string{user.AllAuthenticated},
		Extra:  nil,
	}
)

func setupIAMMock(op *iammock.MockOperator) {
	// Role platform-admin
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("platform-admin"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"kubeclipper.io/aggregation-roles": "[\"role-template-manage-clusters\",\"role-template-view-clusters\",\"role-template-view-roles\",\"role-template-manage-roles\",\"role-template-manage-users\",\"role-template-view-users\"]",
				},
				Name: "platform-admin",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					NonResourceURLs: []string{"*"},
					Verbs:           []string{"*"},
				},
			},
		},
		nil).AnyTimes()
	// Role for platform-view
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("platform-view"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"kubeclipper.io/aggregation-roles": "[\"role-template-view-clusters\",\"role-template-view-roles\",\"role-template-view-users\"]",
				},
				Name: "platform-view",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		nil).AnyTimes()
	// Role for authenticated
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("authenticated"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"kubeclipper.io/rego-override": "package authz\ndefault allow = false\nallow = true {\n  allowedResources := [\"users\"]\n  allowedResources[_] == input.Resource\n  input.User.Name == input.Name\n}",
				},
				Name: "authenticated",
			},
			Rules: []rbacv1.PolicyRule{
				{
					NonResourceURLs: []string{"*"},
					Verbs:           []string{"*"},
				},
			},
		},
		nil).AnyTimes()
	// Role for anonymous
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("anonymous"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "anonymous",
			},
			Rules: []rbacv1.PolicyRule{
				{
					NonResourceURLs: []string{"*"},
					Verbs:           []string{"*"},
				},
			},
		},
		nil).AnyTimes()
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("iam-manager"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"kubeclipper.io/aggregation-roles": "[\"role-template-view-users\",\"role-template-view-roles\",\"role-template-manage-roles\",\"role-template-view-roles\"]",
				},
				Name: "iam-manager",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"iam.kubeclipper.io"},
					Resources: []string{"users"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"iam.kubeclipper.io"},
					Resources: []string{"users", "users/password"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{"iam.kubeclipper.io"},
					Resources: []string{"roles"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"iam.kubeclipper.io"},
					Resources: []string{"roles"},
					Verbs:     []string{"*"},
				},
			},
		},
		nil).AnyTimes()
	op.EXPECT().GetRoleEx(gomock.Any(), gomock.Eq("cluster-manager"), gomock.Any()).Return(
		&v12.GlobalRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       v1.ResourceKindGlobalRole,
				APIVersion: "core.kubeclipper.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"kubeclipper.io/aggregation-roles": "[\"role-template-manage-clusters\",\"role-template-view-clusters\"]",
				},
				Name: "cluster-manager",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"core.kubeclipper.io"},
					Resources: []string{"clusters", "nodes", "regions", "operations"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"core.kubeclipper.io"},
					Resources: []string{"clusters", "clusters/nodes", "clusters/plugins", "nodes", "regions", "operations"},
					Verbs:     []string{"*"},
				},
			},
		},
		nil).AnyTimes()
	// ListRoleBinding
	op.EXPECT().ListRoleBindings(gomock.Any(), gomock.Any()).Return(
		&v12.GlobalRoleBindingList{
			TypeMeta: metav1.TypeMeta{},
			Items: []v12.GlobalRoleBinding{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "anonymous",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "anonymous",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "system:unauthenticated",
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "authenticated",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "authenticated",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "Group",
							Name:     "system:authenticated",
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "platform-admin",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "platform-admin",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "User",
							Name:     "admin",
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "iam-manager",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "iam-manager",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "User",
							Name:     "iammanager",
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster-manager",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "cluster-manager",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "User",
							Name:     "clustermanager",
						},
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "GlobalRoleBinding",
						APIVersion: "core.kubeclipper.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "platform-view",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "core.kubeclipper.io",
						Kind:     "GlobalRole",
						Name:     "platform-view",
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "User",
							Name:     "view",
						},
					},
				},
			},
		},
		nil).AnyTimes()
}

func TestAuthorizer_Authorize(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	iamMockOp := iammock.NewMockOperator(ctrl)

	setupIAMMock(iamMockOp)

	reqInfoFactory := &request.InfoFactory{APIPrefixes: sets.New("api")}

	auth := &Authorizer{am: iamMockOp}

	type args struct {
		user user.Info
		req  *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    authorizer.Decision
		reason  string
		wantErr bool
	}{
		// Anonymous non resource api.
		{
			name: "Anonymous GetMetrics",
			args: args{
				user: userAnonymous,
				req:  reqGetMetrics,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "Anonymous Login",
			args: args{
				user: userAnonymous,
				req:  reqLogin,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "Anonymous Logout",
			args: args{
				user: userAnonymous,
				req:  reqLogout,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "Anonymous oauth",
			args: args{
				user: userAnonymous,
				req:  reqOauth,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		// Anonymous
		{
			name: "Anonymous ListClusters",
			args: args{
				user: userAnonymous,
				req:  reqListClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous CreateClusters",
			args: args{
				user: userAnonymous,
				req:  reqCreateClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous GetCluster",
			args: args{
				user: userAnonymous,
				req:  reqGetCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous DeleteCluster",
			args: args{
				user: userAnonymous,
				req:  reqDeleteClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous ListOperations",
			args: args{
				user: userAnonymous,
				req:  reqListOperations,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous ListNodes",
			args: args{
				user: userAnonymous,
				req:  reqListNodes,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous GetNode",
			args: args{
				user: userAnonymous,
				req:  reqGetNode,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous ListRegions",
			args: args{
				user: userAnonymous,
				req:  reqListRegions,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous GetRegion",
			args: args{
				user: userAnonymous,
				req:  reqGetRegion,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous ListRoles",
			args: args{
				user: userAnonymous,
				req:  reqListRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous CreateRoles",
			args: args{
				user: userAnonymous,
				req:  reqCreateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous GetRole",
			args: args{
				user: userAnonymous,
				req:  reqGetRole,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous UpdateRole",
			args: args{
				user: userAnonymous,
				req:  reqUpdateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous DeleteRole",
			args: args{
				user: userAnonymous,
				req:  reqDeleteRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous ListUsers",
			args: args{
				user: userAnonymous,
				req:  reqListUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous CreateUsers",
			args: args{
				user: userAnonymous,
				req:  reqCreateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous GetUser",
			args: args{
				user: userAnonymous,
				req:  reqGetUser,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous UpdateUser",
			args: args{
				user: userAnonymous,
				req:  reqUpdateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "Anonymous DeleteUser",
			args: args{
				user: userAnonymous,
				req:  reqDeleteUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		// admin non resource api.
		{
			name: "PlatformAdmin GetMetrics",
			args: args{
				user: userAdmin,
				req:  reqGetMetrics,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin Login",
			args: args{
				user: userAdmin,
				req:  reqLogin,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin Logout",
			args: args{
				user: userAdmin,
				req:  reqLogout,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin oauth",
			args: args{
				user: userAdmin,
				req:  reqOauth,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListClusters",
			args: args{
				user: userAdmin,
				req:  reqListClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin CreateClusters",
			args: args{
				user: userAdmin,
				req:  reqCreateClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin GetCluster",
			args: args{
				user: userAdmin,
				req:  reqGetCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin DeleteCluster",
			args: args{
				user: userAdmin,
				req:  reqDeleteClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListOperations",
			args: args{
				user: userAdmin,
				req:  reqListOperations,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListNodes",
			args: args{
				user: userAdmin,
				req:  reqListNodes,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin GetNode",
			args: args{
				user: userAdmin,
				req:  reqGetNode,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListRegions",
			args: args{
				user: userAdmin,
				req:  reqListRegions,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin GetRegion",
			args: args{
				user: userAdmin,
				req:  reqGetRegion,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListRoles",
			args: args{
				user: userAdmin,
				req:  reqListRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin CreateRoles",
			args: args{
				user: userAdmin,
				req:  reqCreateRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin GetRole",
			args: args{
				user: userAdmin,
				req:  reqGetRole,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin UpdateRole",
			args: args{
				user: userAdmin,
				req:  reqUpdateRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin DeleteRole",
			args: args{
				user: userAdmin,
				req:  reqDeleteRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin ListUsers",
			args: args{
				user: userAdmin,
				req:  reqListUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin CreateUsers",
			args: args{
				user: userAdmin,
				req:  reqCreateUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin GetUser",
			args: args{
				user: userAdmin,
				req:  reqGetUser,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin UpdateUser",
			args: args{
				user: userAdmin,
				req:  reqUpdateUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin DeleteUser",
			args: args{
				user: userAdmin,
				req:  reqDeleteUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin AddNodeToCluster",
			args: args{
				user: userAdmin,
				req:  reqAddNodeToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin AddPluginToCluster",
			args: args{
				user: userAdmin,
				req:  reqAddPluginToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformAdmin DelPluginToCluster",
			args: args{
				user: userAdmin,
				req:  reqDelPluginToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		// user iam manager
		{
			name: "IAMManager ListClusters",
			args: args{
				user: userIamManager,
				req:  reqListClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager CreateClusters",
			args: args{
				user: userIamManager,
				req:  reqCreateClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager GetCluster",
			args: args{
				user: userIamManager,
				req:  reqGetCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager DeleteCluster",
			args: args{
				user: userIamManager,
				req:  reqDeleteClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager ListOperations",
			args: args{
				user: userIamManager,
				req:  reqListOperations,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager ListNodes",
			args: args{
				user: userIamManager,
				req:  reqListNodes,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager GetNode",
			args: args{
				user: userIamManager,
				req:  reqGetNode,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager ListRegions",
			args: args{
				user: userIamManager,
				req:  reqListRegions,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager GetRegion",
			args: args{
				user: userIamManager,
				req:  reqGetRegion,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager ListRoles",
			args: args{
				user: userIamManager,
				req:  reqListRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager CreateRoles",
			args: args{
				user: userIamManager,
				req:  reqCreateRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager GetRole",
			args: args{
				user: userIamManager,
				req:  reqGetRole,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager UpdateRole",
			args: args{
				user: userIamManager,
				req:  reqUpdateRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager DeleteRole",
			args: args{
				user: userIamManager,
				req:  reqDeleteRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager ListUsers",
			args: args{
				user: userIamManager,
				req:  reqListUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager CreateUsers",
			args: args{
				user: userIamManager,
				req:  reqCreateUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager GetUser",
			args: args{
				user: userIamManager,
				req:  reqGetUser,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager UpdateUser",
			args: args{
				user: userIamManager,
				req:  reqUpdateUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager DeleteUser",
			args: args{
				user: userIamManager,
				req:  reqDeleteUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "IAMManager AddNodeToCluster",
			args: args{
				user: userIamManager,
				req:  reqAddNodeToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager AddPluginToCluster",
			args: args{
				user: userIamManager,
				req:  reqAddPluginToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager DelPluginToCluster",
			args: args{
				user: userIamManager,
				req:  reqDelPluginToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "IAMManager ChangeUserPassword",
			args: args{
				user: userIamManager,
				req:  reqChangeAdminPassword,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		// cluster manager
		{
			name: "ClusterManager ListClusters",
			args: args{
				user: userClusterManager,
				req:  reqListClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager CreateClusters",
			args: args{
				user: userClusterManager,
				req:  reqCreateClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager GetCluster",
			args: args{
				user: userClusterManager,
				req:  reqGetCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager DeleteCluster",
			args: args{
				user: userClusterManager,
				req:  reqDeleteClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager ListOperations",
			args: args{
				user: userClusterManager,
				req:  reqListOperations,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager ListNodes",
			args: args{
				user: userClusterManager,
				req:  reqListNodes,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager GetNode",
			args: args{
				user: userClusterManager,
				req:  reqGetNode,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager ListRegions",
			args: args{
				user: userClusterManager,
				req:  reqListRegions,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager GetRegion",
			args: args{
				user: userClusterManager,
				req:  reqGetRegion,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager ListRoles",
			args: args{
				user: userClusterManager,
				req:  reqListRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager CreateRoles",
			args: args{
				user: userClusterManager,
				req:  reqCreateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager GetRole",
			args: args{
				user: userClusterManager,
				req:  reqGetRole,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager UpdateRole",
			args: args{
				user: userClusterManager,
				req:  reqUpdateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager DeleteRole",
			args: args{
				user: userClusterManager,
				req:  reqDeleteRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager ListUsers",
			args: args{
				user: userClusterManager,
				req:  reqListUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager CreateUsers",
			args: args{
				user: userClusterManager,
				req:  reqCreateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager GetUserAdmin",
			args: args{
				user: userClusterManager,
				req:  reqGetUserAdmin,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager GetUserClusterManager",
			args: args{
				user: userClusterManager,
				req:  reqGetUserClusterManager,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager ChangeAdminPassword",
			args: args{
				user: userClusterManager,
				req:  reqChangeAdminPassword,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager ChangeClusterManagerPassword",
			args: args{
				user: userClusterManager,
				req:  reqChangeClusterManagerPassword,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager UpdateUser",
			args: args{
				user: userClusterManager,
				req:  reqUpdateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager DeleteUser",
			args: args{
				user: userClusterManager,
				req:  reqDeleteUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "ClusterManager AddNodeToCluster",
			args: args{
				user: userClusterManager,
				req:  reqAddNodeToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager AddPluginToCluster",
			args: args{
				user: userClusterManager,
				req:  reqAddPluginToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "ClusterManager DelPluginToCluster",
			args: args{
				user: userClusterManager,
				req:  reqDelPluginToCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		// platform view
		{
			name: "PlatformView ListClusters",
			args: args{
				user: userPlatformView,
				req:  reqListClusters,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView CreateClusters",
			args: args{
				user: userPlatformView,
				req:  reqCreateClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView GetCluster",
			args: args{
				user: userPlatformView,
				req:  reqGetCluster,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView DeleteCluster",
			args: args{
				user: userPlatformView,
				req:  reqDeleteClusters,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView ListOperations",
			args: args{
				user: userPlatformView,
				req:  reqListOperations,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView ListNodes",
			args: args{
				user: userPlatformView,
				req:  reqListNodes,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView GetNode",
			args: args{
				user: userPlatformView,
				req:  reqGetNode,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView ListRegions",
			args: args{
				user: userPlatformView,
				req:  reqListRegions,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView GetRegion",
			args: args{
				user: userPlatformView,
				req:  reqGetRegion,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView ListRoles",
			args: args{
				user: userPlatformView,
				req:  reqListRoles,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView CreateRoles",
			args: args{
				user: userPlatformView,
				req:  reqCreateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView GetRole",
			args: args{
				user: userPlatformView,
				req:  reqGetRole,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView UpdateRole",
			args: args{
				user: userPlatformView,
				req:  reqUpdateRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView DeleteRole",
			args: args{
				user: userPlatformView,
				req:  reqDeleteRoles,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView ListUsers",
			args: args{
				user: userPlatformView,
				req:  reqListUsers,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView CreateUsers",
			args: args{
				user: userPlatformView,
				req:  reqCreateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView GetUser",
			args: args{
				user: userPlatformView,
				req:  reqGetUser,
			},
			want:    authorizer.DecisionAllow,
			wantErr: false,
		},
		{
			name: "PlatformView UpdateUser",
			args: args{
				user: userPlatformView,
				req:  reqUpdateUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView DeleteUser",
			args: args{
				user: userPlatformView,
				req:  reqDeleteUsers,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView AddNodeToCluster",
			args: args{
				user: userPlatformView,
				req:  reqAddNodeToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView AddPluginToCluster",
			args: args{
				user: userPlatformView,
				req:  reqAddPluginToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
		{
			name: "PlatformView DelPluginToCluster",
			args: args{
				user: userPlatformView,
				req:  reqDelPluginToCluster,
			},
			want:    authorizer.DecisionNoOpinion,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := reqInfoFactory.NewRequestInfo(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, got1, err := auth.Authorize(context.TODO(), getAuthorizerAttributes(tt.args.user, info))
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Authorize() got = %v, want %v", got, tt.want)
			}
			if got1 != "" {
				t.Logf("Authorize() reason = %v", got1)
			}
		})
	}
}

func getAuthorizerAttributes(userInfo user.Info, reqInfo *request.Info) *authorizer.AttributesRecord {
	attribs := authorizer.AttributesRecord{}

	attribs.User = userInfo
	attribs.ResourceRequest = reqInfo.IsResourceRequest
	attribs.Path = reqInfo.Path
	attribs.Verb = reqInfo.Verb
	attribs.APIGroup = reqInfo.APIGroup
	attribs.APIVersion = reqInfo.APIVersion
	attribs.Resource = reqInfo.Resource
	attribs.Subresource = reqInfo.Subresource
	attribs.Name = reqInfo.Name

	return &attribs
}
