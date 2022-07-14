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

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset/versioned/scheme"

	coordinationv1 "k8s.io/api/coordination/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

var _ LeasesInterface = (*leases)(nil)

type LeasesGetter interface {
	Leases() LeasesInterface
}

type LeasesInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*coordinationv1.Lease, error)
	List(ctx context.Context, opts v1.ListOptions) (*coordinationv1.LeaseList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type leases struct {
	client rest.Interface
}

func newLeases(c *CoreV1Client) *leases {
	return &leases{client: c.RESTClient()}
}

func (c *leases) Get(ctx context.Context, name string, opts v1.GetOptions) (result *coordinationv1.Lease, err error) {
	result = &coordinationv1.Lease{}
	err = c.client.Get().
		Resource("leases").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *leases) List(ctx context.Context, opts v1.ListOptions) (result *coordinationv1.LeaseList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &coordinationv1.LeaseList{}
	err = c.client.Get().
		Resource("leases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *leases) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("leases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
