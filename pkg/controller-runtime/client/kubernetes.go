/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type Client interface {
	Kubernetes() kubernetes.Interface
	Config() *rest.Config
}

type kubernetesClient struct {
	// kubernetes client interface
	k8s    kubernetes.Interface
	config *rest.Config
}

// NewKubernetesClient creates a KubernetesClient
func NewKubernetesClient(config *rest.Config, client *kubernetes.Clientset) Client {
	return &kubernetesClient{
		k8s:    client,
		config: config,
	}
}

func (k *kubernetesClient) Kubernetes() kubernetes.Interface {
	return k.k8s
}

func (k *kubernetesClient) Config() *rest.Config {
	return k.config
}

func FromKubeConfig(kubeConfig []byte) (*rest.Config, *kubernetes.Clientset, error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeConfig)
	if err != nil {
		logger.Error("create cluster client config failed", zap.Error(err))
		return nil, nil, err
	}
	clientcfg, err := clientConfig.ClientConfig()
	if err != nil {
		logger.Error("get cluster kubeconfig client failed", zap.Error(err))
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(clientcfg)
	if err != nil {
		logger.Error("create cluster clientset failed", zap.Error(err))
		return nil, nil, err
	}
	return clientcfg, clientset, nil
}
