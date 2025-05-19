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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

// LoadImage decompress and load image
func LoadImage(ctx context.Context, dryRun bool, file, criType string) error {
	//_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "gzip", "-df", file)
	//if err != nil {
	//	return err
	//}
	//
	//file = strings.ReplaceAll(file, ".gz", "")

	switch criType {
	case "containerd":
		// ctr --namespace k8s.io image import --all-platforms xxx/images.tar
		// Why need set `--all-platforms` flag?
		// This prevents unexpected errors due to the incorrect schema type declaration of the mirroring system that cannot be imported normally.
		_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "nerdctl", "-n", "k8s.io", "load", "-i", file)
		if err != nil {
			return err
		}
	case "docker":
		// docker load -i xxx/images.tar
		_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "docker", "load", "-i", file)
		if err != nil {
			return err
		}
	}

	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "rm", "-rf", file)

	return err
}

func RetryFunc(ctx context.Context, opts component.Options, intervalTime time.Duration, funcName string, fn func(ctx context.Context, opts component.Options) error) error {
	for {
		select {
		case <-ctx.Done():
			logger.Warnf("retry function '%s' timeout...", funcName)
			return ctx.Err()
		case <-time.After(intervalTime):
			err := fn(ctx, opts)
			if err == nil {
				return nil
			}
			logger.Warnf("function '%s' running error: %s. about to enter retry", funcName, err.Error())
		}
	}
}

func UnwrapNodeList(nl component.NodeList) (nodes []v1.StepNode) {
	for _, v := range nl {
		nodes = append(nodes, v1.StepNode{
			ID:       v.ID,
			IPv4:     v.IPv4,
			NodeIPv4: v.NodeIPv4,
			Hostname: v.Hostname,
		})
	}
	return nodes
}

func BuildStepNode(nodes []*v1.Node) (stepNodes []v1.StepNode) {
	for _, node := range nodes {
		stepNodes = append(stepNodes, v1.StepNode{
			ID:       node.Name,
			IPv4:     node.Status.Ipv4DefaultIP,
			NodeIPv4: node.Status.NodeIpv4DefaultIP,
			Hostname: node.Labels[common.LabelHostname],
		})
	}
	return stepNodes
}

// NamespacedKey returns ${namespace}/${name} string(namespaced key) of a Kubernetes object.
func NamespacedKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func BuildKubeClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	kubeCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
	}
	return kubernetes.NewForConfig(kubeCfg)
}

func NewMetadata(c *v1.Cluster) *component.ExtraMetadata {
	return &component.ExtraMetadata{
		ClusterName:        c.Name,
		ClusterStatus:      c.Status.Phase,
		Offline:            c.Offline(),
		LocalRegistry:      c.LocalRegistry,
		CRI:                c.ContainerRuntime.Type,
		KubeVersion:        c.KubernetesVersion,
		KubeletDataDir:     c.Kubelet.RootDir,
		ControlPlaneStatus: c.Status.ControlPlaneHealth,
		CNI:                c.CNI.Type,
		CNINamespace:       c.CNI.Namespace,
	}
}

func GetClusterCRIRegistries(c *v1.Cluster, op cluster.Operator) ([]v1.RegistrySpec, error) {
	/*
			NOTE: there are 3 way to set registry
			1.cri registry,use to generate containerd config
			2.insecure registry(local registry),use to pull image for create k8s cluster
			3.mirror registry,use to pull image for deploy k8s addons
		insecure registry and mirror registry only used to config insecure for pull image
		cri registry can use to config insecure and auth
	*/

	registries := make([]v1.RegistrySpec, 0)

	// First deal with cri registry(with auth) in Registries
	for _, reg := range c.ContainerRuntime.Registries {
		// cri registry
		if reg.RegistryRef != nil && *reg.RegistryRef != "" {
			//registry, err := r.RegistryLister.Get(*reg.RegistryRef)
			// use operator not lister
			registry, err := op.GetRegistry(context.Background(), *reg.RegistryRef)
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return nil, fmt.Errorf("get cri registry %s:%w", *reg.RegistryRef, err)
			}
			logger.Debugf("【cluster-controller】 getClusterCRIRegistries registry:%#v  auth:%#v", registry, registry.RegistryAuth)
			registries = appendUniqueRegistry(registries, registry.RegistrySpec)
		} else {
			// insecure registry in cri registry
			registries = appendUniqueRegistry(registries,
				v1.RegistrySpec{Scheme: "http", Host: reg.InsecureRegistry},
				v1.RegistrySpec{Scheme: "https", Host: reg.InsecureRegistry, SkipVerify: true})
			continue
		}
	}

	// Then,deal with insecure registry and mirror registry
	type mirror struct {
		ImageRepoMirror string `json:"imageRepoMirror"`
	}
	insecureRegistry := append([]string{}, c.ContainerRuntime.InsecureRegistry...)
	sort.Strings(insecureRegistry)
	// add addons mirror registry
	for _, a := range c.Addons {
		var m mirror
		err := json.Unmarshal(a.Config.Raw, &m)
		if err != nil {
			continue
		}
		if m.ImageRepoMirror != "" {
			idx, ok := sort.Find(len(insecureRegistry), func(i int) int {
				return strings.Compare(m.ImageRepoMirror, insecureRegistry[i])
			})
			if !ok {
				if idx == len(insecureRegistry) {
					insecureRegistry = append(insecureRegistry, m.ImageRepoMirror)
				} else {
					insecureRegistry = append(insecureRegistry[:idx+1], insecureRegistry[idx:]...)
					insecureRegistry[idx] = m.ImageRepoMirror
				}
			}
		}
	}

	// insecure registry
	for _, host := range insecureRegistry {
		registries = appendUniqueRegistry(registries,
			v1.RegistrySpec{Scheme: "http", Host: host},
			v1.RegistrySpec{Scheme: "https", Host: host, SkipVerify: true})
	}

	return registries, nil
}

// Use scheme and host as unique key
func appendUniqueRegistry(s []v1.RegistrySpec, items ...v1.RegistrySpec) []v1.RegistrySpec {
	for _, r := range items {
		key := r.Scheme + r.Host
		idx, ok := sort.Find(len(s), func(i int) int {
			return strings.Compare(key, s[i].Scheme+s[i].Host)
		})
		if !ok {
			if idx == len(s) {
				s = append(s, r)
			} else {
				s = append(s[:idx+1], s[idx:]...)
				s[idx] = r
			}
		}
	}
	return s
}
