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
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

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
		ImageRepository:    c.ImageRepository,
		CRI:                c.ContainerRuntime.Type,
		KubeVersion:        c.KubernetesVersion,
		KubeletDataDir:     c.Kubelet.RootDir,
		ControlPlaneStatus: c.Status.ControlPlaneHealth,
		CNI:                c.CNI.Type,
		CNINamespace:       c.CNI.Namespace,
	}
}

func GetClusterCRIRegistries(c *v1.Cluster, op cluster.Operator) ([]v1.RegistrySpec, error) {
	return GetClusterCRIRegistriesWithContext(context.Background(), c, op)
}

func NormalizeImageRepository(repository string) (normalized, host string, err error) {
	repository = strings.TrimSpace(strings.TrimSuffix(repository, "/"))
	if repository == "" {
		repository = v1.DefaultImageRepository
	}
	if strings.Contains(repository, "://") {
		return "", "", fmt.Errorf("image repository %q must not include a URL scheme", repository)
	}
	parsed, err := url.Parse("https://" + repository)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", fmt.Errorf("invalid image repository %q", repository)
	}
	if strings.Contains(parsed.Host, "\\") || strings.Contains(parsed.Path, "\\") {
		return "", "", fmt.Errorf("invalid image repository %q", repository)
	}
	return repository, parsed.Host, nil
}

func GetClusterCRIRegistriesWithContext(ctx context.Context, c *v1.Cluster, op cluster.Operator) ([]v1.RegistrySpec, error) {
	repository, host, err := NormalizeImageRepository(c.ImageRepository)
	if err != nil {
		return nil, err
	}
	c.ImageRepository = repository
	registries := []v1.RegistrySpec{{Scheme: "https", Host: host}}
	explicitHosts := make(map[string]v1.RegistrySpec)
	for _, ref := range c.ContainerRuntime.Registries {
		if ref.RegistryRef == nil || strings.TrimSpace(*ref.RegistryRef) == "" {
			return nil, fmt.Errorf("container runtime registryRef must not be empty")
		}
		registry, err := op.GetRegistry(ctx, *ref.RegistryRef)
		if err != nil {
			return nil, fmt.Errorf("get cri registry %s: %w", *ref.RegistryRef, err)
		}
		spec, err := normalizeRegistrySpec(registry.RegistrySpec)
		if err != nil {
			return nil, fmt.Errorf("cri registry %s: %w", *ref.RegistryRef, err)
		}
		if existing, ok := explicitHosts[spec.Host]; ok {
			if !reflect.DeepEqual(existing, spec) {
				return nil, fmt.Errorf("cri registry %s: registry host %s has conflicting configuration", *ref.RegistryRef, spec.Host)
			}
			continue
		}
		explicitHosts[spec.Host] = spec
	}
	explicitHostOrder := make([]string, 0, len(explicitHosts))
	for explicitHost := range explicitHosts {
		explicitHostOrder = append(explicitHostOrder, explicitHost)
	}
	sort.Strings(explicitHostOrder)
	for _, host := range explicitHostOrder {
		registries = replaceRegistryByHost(registries, explicitHosts[host])
	}
	sort.Slice(registries, func(i, j int) bool {
		return registries[i].Host < registries[j].Host
	})
	return registries, nil
}

func normalizeRegistrySpec(item v1.RegistrySpec) (v1.RegistrySpec, error) {
	item.Scheme = strings.ToLower(strings.TrimSpace(item.Scheme))
	item.Host = strings.TrimSuffix(strings.TrimSpace(item.Host), "/")
	if item.Scheme == "" {
		item.Scheme = "https"
	}
	if item.Host == "" {
		return v1.RegistrySpec{}, fmt.Errorf("registry host must not be empty")
	}
	if strings.Contains(item.Host, "://") || strings.Contains(item.Host, "/") {
		return v1.RegistrySpec{}, fmt.Errorf("registry host %q must not include a scheme or path", item.Host)
	}
	return item, nil
}

func replaceRegistryByHost(registries []v1.RegistrySpec, item v1.RegistrySpec) []v1.RegistrySpec {
	result := registries[:0]
	for i := range registries {
		if registries[i].Host != item.Host {
			result = append(result, registries[i])
		}
	}
	return append(result, item)
}
