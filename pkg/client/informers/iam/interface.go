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

package iam

import (
	"k8s.io/client-go/informers/internalinterfaces"

	v1 "github.com/kubeclipper/kubeclipper/pkg/client/informers/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/client/internal"
)

type Interface interface {
	V1() v1.Interface
}

type group struct {
	factor           internal.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

func New(f internal.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &group{factor: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

func (g *group) V1() v1.Interface {
	return v1.New(g.factor, g.namespace, g.tweakListOptions)
}
