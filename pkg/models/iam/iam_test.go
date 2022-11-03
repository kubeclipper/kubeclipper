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
	"reflect"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

var i = &iamOperator{
	userStorage:        nil,
	roleStorage:        nil,
	roleBindingStorage: nil,
	tokenStorage:       nil,
	loginRecordStorage: nil,
}

func Test_iamOperator_userFuzzyFilter(t *testing.T) {
	type args struct {
		obj runtime.Object
		q   *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			name: "user fuzzy filter test",
			args: args{
				obj: &iamv1.UserList{
					Items: []iamv1.User{
						{
							Spec: iamv1.UserSpec{
								Email: "email1",
							},
						},
						{
							Spec: iamv1.UserSpec{
								Email: "email2",
							},
						},
						{
							Spec: iamv1.UserSpec{
								Email: "email3",
							},
						},
					},
				},
				q: &query.Query{
					FuzzySearch: map[string]string{
						"query1": "test1",
						"query2": "test2",
						"query3": "test3",
					},
				},
			},
			want: []runtime.Object{
				&iamv1.User{
					Spec: iamv1.UserSpec{
						Email: "email1",
					},
				},
				&iamv1.User{
					Spec: iamv1.UserSpec{
						Email: "email2",
					},
				},
				&iamv1.User{
					Spec: iamv1.UserSpec{
						Email: "email3",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UserFuzzyFilter(tt.args.obj, tt.args.q); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("userFuzzyFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_iamOperator_roleFuzzyFilter1(t *testing.T) {
	type args struct {
		obj runtime.Object
		q   *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			name: "role fuzzy filter test",
			args: args{
				obj: &iamv1.GlobalRoleList{
					Items: []iamv1.GlobalRole{
						{
							Rules: []rbacv1.PolicyRule{
								{
									Verbs: []string{"test_verb_1"},
								},
							},
						},
						{
							Rules: []rbacv1.PolicyRule{
								{
									Verbs: []string{"test_verb_1"},
								},
							},
						},
					},
				},
				q: &query.Query{
					FuzzySearch: map[string]string{
						"query1": "test1",
						"query2": "test2",
						"query3": "test3",
					},
				},
			},
			want: []runtime.Object{
				&iamv1.GlobalRole{
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{"test_verb_1"},
						},
					},
				},
				&iamv1.GlobalRole{
					Rules: []rbacv1.PolicyRule{
						{
							Verbs: []string{"test_verb_1"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := i.roleFuzzyFilter(tt.args.obj, tt.args.q); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("roleFuzzyFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_iamOperator_roleBindingFuzzyFilter(t *testing.T) {
	type args struct {
		obj runtime.Object
		in1 *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			args: args{
				obj: &iamv1.GlobalRoleBindingList{
					Items: []iamv1.GlobalRoleBinding{
						{
							Subjects: []rbacv1.Subject{
								{
									Name: "test1",
								},
							},
						},
						{
							Subjects: []rbacv1.Subject{
								{
									Name: "test2",
								},
							},
						},
					},
				},
				in1: nil,
			},
			want: []runtime.Object{
				&iamv1.GlobalRoleBinding{
					Subjects: []rbacv1.Subject{
						{
							Name: "test1",
						},
					},
				},
				&iamv1.GlobalRoleBinding{
					Subjects: []rbacv1.Subject{
						{
							Name: "test2",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := i.roleBindingFuzzyFilter(tt.args.obj, tt.args.in1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("roleBindingFuzzyFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_iamOperator_loginRecordFuzzyFilter(t *testing.T) {
	type args struct {
		obj runtime.Object
		in1 *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			name: "login record fuzzy filter test",
			args: args{
				obj: &iamv1.LoginRecordList{
					Items: []iamv1.LoginRecord{
						{
							Spec: iamv1.LoginRecordSpec{
								Type: iamv1.TokenLogin,
							},
						},
					},
				},
				in1: nil,
			},
			want: []runtime.Object{
				&iamv1.LoginRecord{
					Spec: iamv1.LoginRecordSpec{
						Type: iamv1.TokenLogin,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := i.loginRecordFuzzyFilter(tt.args.obj, tt.args.in1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loginRecordFuzzyFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_iamOperator_tokenFuzzyFilter(t *testing.T) {
	type args struct {
		obj runtime.Object
		in1 *query.Query
	}
	tests := []struct {
		name string
		args args
		want []runtime.Object
	}{
		{
			name: "token fuzzy filter test",
			args: args{
				obj: &iamv1.TokenList{
					Items: []iamv1.Token{
						{
							Spec: iamv1.TokenSpec{
								TokenType: iamv1.AccessToken,
							},
						},
					},
				},
				in1: nil,
			},
			want: []runtime.Object{
				&iamv1.Token{
					Spec: iamv1.TokenSpec{
						TokenType: iamv1.AccessToken,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := i.tokenFuzzyFilter(tt.args.obj, tt.args.in1); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenFuzzyFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
