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

package utils

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mockcluster "github.com/kubeclipper/kubeclipper/pkg/models/cluster/mock"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestGetClusterCRIRegistries(t *testing.T) {
	ctrl := gomock.NewController(t)
	op := mockcluster.NewMockOperator(ctrl)
	auth := &v1.RegistryAuth{Username: "user", Password: "secret"}
	op.EXPECT().GetRegistry(gomock.Any(), "business").Return(&v1.Registry{
		ObjectMeta:   metav1.ObjectMeta{Name: "business"},
		RegistrySpec: v1.RegistrySpec{Scheme: "https", Host: "registry.business.example.com", CA: "ca", RegistryAuth: auth},
	}, nil)
	op.EXPECT().GetRegistry(gomock.Any(), "image-auth").Return(&v1.Registry{
		ObjectMeta:   metav1.ObjectMeta{Name: "image-auth"},
		RegistrySpec: v1.RegistrySpec{Scheme: "https", Host: "harbor.example.com", SkipVerify: true, RegistryAuth: auth},
	}, nil)
	business := "business"
	cluster := &v1.Cluster{
		ImageRegistry: "image-auth",
		ContainerRuntime: v1.ContainerRuntime{Registries: []v1.CRIRegistry{
			{RegistryRef: &business},
		}},
	}
	got, err := GetClusterCRIRegistriesWithContext(context.Background(), cluster, op)
	if err != nil {
		t.Fatal(err)
	}
	want := []v1.RegistrySpec{
		{Scheme: "https", Host: "harbor.example.com", SkipVerify: true, RegistryAuth: auth},
		{Scheme: "https", Host: "registry.business.example.com", CA: "ca", RegistryAuth: auth},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("registries = %#v, want %#v", got, want)
	}
	if cluster.ResolvedImageRegistry != "harbor.example.com" {
		t.Fatalf("resolved image registry = %q", cluster.ResolvedImageRegistry)
	}
}

func TestGetClusterCRIRegistriesRejectsConflictingResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	op := mockcluster.NewMockOperator(ctrl)
	op.EXPECT().GetRegistry(gomock.Any(), "first").Return(&v1.Registry{RegistrySpec: v1.RegistrySpec{Scheme: "https", Host: "registry.example.com"}}, nil)
	op.EXPECT().GetRegistry(gomock.Any(), "second").Return(&v1.Registry{RegistrySpec: v1.RegistrySpec{Scheme: "http", Host: "registry.example.com"}}, nil)
	first, second := "first", "second"
	cluster := &v1.Cluster{ContainerRuntime: v1.ContainerRuntime{Registries: []v1.CRIRegistry{{RegistryRef: &first}, {RegistryRef: &second}}}}
	if _, err := GetClusterCRIRegistriesWithContext(context.Background(), cluster, op); err == nil {
		t.Fatal("expected conflicting registry configuration error")
	}
}

func TestLoadImage(t *testing.T) {

	type args struct {
		ctx     context.Context
		dryRun  bool
		file    string
		criType string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "load a docker image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "docker",
			},
		},
		{
			name: "load a containerd image",
			args: args{
				ctx:     context.TODO(),
				dryRun:  true,
				file:    "/demoPath",
				criType: "containerd",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadImage(tt.args.ctx, tt.args.dryRun, tt.args.file, tt.args.criType); err != nil {
				t.Errorf("LoadImage() error = %v", err)
			}
		})
	}
}
