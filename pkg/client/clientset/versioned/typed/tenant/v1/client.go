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

// Package v1 implements tenant/v1 client.
package v1

import (
	"k8s.io/client-go/rest"

	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientrest"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset/versioned/scheme"
)

var _ TenantV1Interface = (*TenantV1Client)(nil)

// TenantV1Interface include the clientset about tenant resource
type TenantV1Interface interface {
	RESTClient() rest.Interface
	ProjectsGetter
}

// TenantV1Client implement TenantV1Interface
type TenantV1Client struct {
	restClient rest.Interface
}

// New return a TenantV1Interface
func New(c rest.Interface) TenantV1Interface {
	return &TenantV1Client{restClient: c}
}

// RESTClient return a RESTClient
func (c *TenantV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}

// Projects return an obj which implement ProjectInterface
func (c *TenantV1Client) Projects() ProjectInterface {
	return newProjects(c)
}

// NewForConfig return a TenantV1Client from config.
func NewForConfig(c *rest.Config) (*TenantV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := clientrest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &TenantV1Client{client}, nil
}

func setConfigDefaults(config *rest.Config) error {
	gv := tenantv1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/api"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	// if config.UserAgent == "" {
	//	config.UserAgent = rest.DefaultKubernetesUserAgent()
	// }

	return nil
}
