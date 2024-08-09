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

package create

import (
	"context"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

func getComponentAvailableVersions(client *kc.Client, offline bool, component string) sets.Set[string] {
	var metas *kc.ComponentMeta
	var err error
	if offline {
		metas, err = client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"false"}})
	} else {
		metas, err = client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"true"}})
	}
	if err != nil {
		logger.Errorf("get component meta failed: %s. please check .kc/config", err)
		return nil
	}
	set := sets.New[string]()
	for _, resource := range metas.Addons {
		if resource.Name == component {
			set.Insert(resource.Version)
		}
	}
	return set
}
