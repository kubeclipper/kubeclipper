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

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset/versioned/scheme"
	iamv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

var _ LoginRecordInterface = (*loginRecords)(nil)

type LoginRecordsGetter interface {
	LoginRecords() LoginRecordInterface
}

type LoginRecordInterface interface {
	Get(ctx context.Context, name string, opts v1.GetOptions) (*iamv1.LoginRecord, error)
	List(ctx context.Context, opts v1.ListOptions) (*iamv1.LoginRecordList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
}

type loginRecords struct {
	client rest.Interface
}

func newLoginRecords(c *IamV1Client) *loginRecords {
	return &loginRecords{client: c.RESTClient()}
}

func (c *loginRecords) Get(ctx context.Context, name string, opts v1.GetOptions) (result *iamv1.LoginRecord, err error) {
	result = &iamv1.LoginRecord{}
	err = c.client.Get().
		Resource("loginrecords").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

func (c *loginRecords) List(ctx context.Context, opts v1.ListOptions) (result *iamv1.LoginRecordList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &iamv1.LoginRecordList{}
	err = c.client.Get().
		Resource("loginrecords").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

func (c *loginRecords) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("loginrecords").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
