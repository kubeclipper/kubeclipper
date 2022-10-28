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

package v1

import (
	"context"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset/versioned/scheme"
)

var _ ProjectInterface = (*projects)(nil)

type ProjectsGetter interface {
	Projects() ProjectInterface
}

type ProjectInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*tenantv1.Project, error)
	List(ctx context.Context, opts v1.ListOptions) (*tenantv1.ProjectList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type projects struct {
	client rest.Interface
}

func newProjects(c *TenantV1Client) *projects {
	return &projects{client: c.RESTClient()}
}

func (c *projects) Get(ctx context.Context, name string, opts v1.GetOptions) (result *tenantv1.Project, err error) {
	result = &tenantv1.Project{}
	err = c.client.Get().
		Resource("projects").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *projects) List(ctx context.Context, opts v1.ListOptions) (result *tenantv1.ProjectList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &tenantv1.ProjectList{}
	err = c.client.Get().
		Resource("projects").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *projects) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("projects").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
