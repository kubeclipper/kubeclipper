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
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestVerbMatches(t *testing.T) {
	type args struct {
		rule          *rbacv1.PolicyRule
		requestedVerb string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "all verb match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Verbs: []string{rbacv1.VerbAll},
				},
				requestedVerb: "testVerb",
			},
			want: true,
		},
		{
			name: "spec verb match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Verbs: []string{"testVerb"},
				},
				requestedVerb: "testVerb",
			},
			want: true,
		},
		{
			name: "spec wrong verb match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Verbs: []string{"test verb wrong"},
				},
				requestedVerb: "testVerb",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerbMatches(tt.args.rule, tt.args.requestedVerb); got != tt.want {
				t.Errorf("VerbMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIGroupMatches(t *testing.T) {
	type args struct {
		rule           *rbacv1.PolicyRule
		requestedGroup string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "group all test",
			args: args{
				rule: &rbacv1.PolicyRule{
					APIGroups: []string{rbacv1.APIGroupAll},
				},
				requestedGroup: "empty",
			},
			want: true,
		},
		{
			name: "spec right group test",
			args: args{
				rule: &rbacv1.PolicyRule{
					APIGroups: []string{"testGroup"},
				},
				requestedGroup: "testGroup",
			},
			want: true,
		},
		{
			name: "spec wrong group test",
			args: args{
				rule: &rbacv1.PolicyRule{
					APIGroups: []string{"test group wrong"},
				},
				requestedGroup: "testGroup",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := APIGroupMatches(tt.args.rule, tt.args.requestedGroup); got != tt.want {
				t.Errorf("APIGroupMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceMatches(t *testing.T) {
	type args struct {
		rule                      *rbacv1.PolicyRule
		combinedRequestedResource string
		requestedSubresource      string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "all resource match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Resources: []string{rbacv1.ResourceAll},
				},
				combinedRequestedResource: "test",
				requestedSubresource:      "test",
			},
			want: true,
		},
		{
			name: "spec resource match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Resources: []string{"testResource"},
				},
				combinedRequestedResource: "testResource",
				requestedSubresource:      "empty",
			},
			want: true,
		},
		{
			name: "spec wrong group test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Resources: []string{"test resource"},
				},
				combinedRequestedResource: "testResource",
				requestedSubresource:      "empty",
			},
			want: false,
		},
		{
			name: "requestedSubresource test",
			args: args{
				rule: &rbacv1.PolicyRule{
					Resources: []string{"*/", "test1", "test2"},
				},
				combinedRequestedResource: "empty",
				requestedSubresource:      "test2",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResourceMatches(tt.args.rule, tt.args.combinedRequestedResource, tt.args.requestedSubresource); got != tt.want {
				t.Errorf("ResourceMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceNameMatches(t *testing.T) {
	type args struct {
		rule          *rbacv1.PolicyRule
		requestedName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "resourceName 0 match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					ResourceNames: []string{},
				},
				requestedName: "empty",
			},
			want: true,
		},
		{
			name: "resourceName 0 match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					ResourceNames: []string{"test"},
				},
				requestedName: "test",
			},
			want: true,
		},
		{
			name: "resourceName 0 match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					ResourceNames: []string{"test"},
				},
				requestedName: "empty",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResourceNameMatches(tt.args.rule, tt.args.requestedName); got != tt.want {
				t.Errorf("ResourceNameMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNonResourceURLMatches(t *testing.T) {
	type args struct {
		rule         *rbacv1.PolicyRule
		requestedURL string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "all resource url match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					NonResourceURLs: []string{rbacv1.NonResourceAll},
				},
				requestedURL: "empty",
			},
			want: true,
		},
		{
			name: "spec resource url match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					NonResourceURLs: []string{"testNonResourceURLs"},
				},
				requestedURL: "testNonResourceURLs",
			},
			want: true,
		},
		{
			name: "contain resource url match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					NonResourceURLs: []string{"test1", "*"},
				},
				requestedURL: "test1",
			},
			want: true,
		},
		{
			name: "wrong resource url match test",
			args: args{
				rule: &rbacv1.PolicyRule{
					NonResourceURLs: []string{"testNonResourceURLs"},
				},
				requestedURL: "empty",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NonResourceURLMatches(tt.args.rule, tt.args.requestedURL); got != tt.want {
				t.Errorf("NonResourceURLMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
