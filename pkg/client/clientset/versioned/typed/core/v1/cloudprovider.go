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

	"k8s.io/client-go/rest"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

var _ CloudProvidersInterface = (*cloudProviders)(nil)

type CloudProvidersGetter interface {
	CloudProviders() CloudProvidersInterface
}

type CloudProvidersInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*corev1.CloudProvider, error)
	List(ctx context.Context, opts v1.ListOptions) (*corev1.CloudProviderList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type cloudProviders struct {
	client rest.Interface
}

func newCloudProviders(c *CoreV1Client) *cloudProviders {
	return &cloudProviders{client: c.RESTClient()}
}

func (c *cloudProviders) Get(ctx context.Context, name string, opts v1.GetOptions) (result *corev1.CloudProvider, err error) {
	result = &corev1.CloudProvider{}
	err = c.client.Get().
		Resource("cloudproviders").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *cloudProviders) List(ctx context.Context, opts v1.ListOptions) (result *corev1.CloudProviderList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &corev1.CloudProviderList{}
	err = c.client.Get().
		Resource("cloudproviders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *cloudProviders) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("cloudproviders").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
