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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var _ RegionsInterface = (*regions)(nil)

type RegionsGetter interface {
	Regions() RegionsInterface
}

type RegionsInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*corev1.Region, error)
	List(ctx context.Context, opts v1.ListOptions) (*corev1.RegionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type regions struct {
	client rest.Interface
}

func newRegions(c *CoreV1Client) *regions {
	return &regions{client: c.RESTClient()}
}

func (c *regions) Get(ctx context.Context, name string, opts v1.GetOptions) (result *corev1.Region, err error) {
	result = &corev1.Region{}
	err = c.client.Get().
		Resource("regions").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *regions) List(ctx context.Context, opts v1.ListOptions) (result *corev1.RegionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &corev1.RegionList{}
	err = c.client.Get().
		Resource("regions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *regions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("regions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
