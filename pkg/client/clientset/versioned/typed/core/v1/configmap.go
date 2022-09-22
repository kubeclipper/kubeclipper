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

var _ ConfigMapsInterface = (*configMaps)(nil)

type ConfigMapsGetter interface {
	ConfigMaps() ConfigMapsInterface
}

type ConfigMapsInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*corev1.ConfigMap, error)
	List(ctx context.Context, opts v1.ListOptions) (*corev1.ConfigMapList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type configMaps struct {
	client rest.Interface
}

func newConfigMaps(c *CoreV1Client) *configMaps {
	return &configMaps{client: c.RESTClient()}
}

func (c *configMaps) Get(ctx context.Context, name string, opts v1.GetOptions) (result *corev1.ConfigMap, err error) {
	result = &corev1.ConfigMap{}
	err = c.client.Get().
		Resource("configmaps").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *configMaps) List(ctx context.Context, opts v1.ListOptions) (result *corev1.ConfigMapList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &corev1.ConfigMapList{}
	err = c.client.Get().
		Resource("configmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *configMaps) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("configmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
