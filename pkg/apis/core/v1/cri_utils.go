/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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
	"fmt"
	"reflect"

	componentutils "github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/controller/clustercontroller"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func (h *handler) getClusterCRIRegistries(ctx context.Context, c *v1.Cluster) ([]v1.RegistrySpec, error) {
	return componentutils.GetClusterCRIRegistriesWithContext(ctx, c, h.clusterOperator)
}

func registriesEqual(a, b []v1.RegistrySpec) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (h *handler) getCRIRegistriesStep(ctx context.Context, cluster *v1.Cluster, registries []v1.RegistrySpec) (*v1.Step, error) {
	if !registriesEqual(cluster.Status.Registries, registries) {
		q := query.New()
		q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, cluster.Name)

		nodeList, err := h.clusterOperator.ListNodes(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("list")
		}
		return criRegistryUpdateStep(cluster, registries, nodeList.Items)
	}
	return nil, nil
}

func criRegistryUpdateStep(cluster *v1.Cluster, registries []v1.RegistrySpec, nodes []v1.Node) (*v1.Step, error) {
	wrapNodes := make([]*v1.Node, 0, len(nodes))
	for i := range nodes {
		wrapNodes = append(wrapNodes, &nodes[i])
	}
	return clustercontroller.CRIRegistryUpdateStep(cluster, registries, wrapNodes)
}
