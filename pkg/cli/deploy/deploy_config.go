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

package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// GetDeployConfig returns the authoritative deployment configuration stored by kc-server.
// When syncLocal is true, it also refreshes the default local cache.
func GetDeployConfig(ctx context.Context, cli *kc.Client, syncLocal bool) (*options.DeployConfig, error) {
	dc := new(options.DeployConfig)
	configMap, err := cli.DescribeConfigMap(ctx, constatns.DeployConfigConfigMapName)
	if err != nil {
		return nil, err
	}
	if len(configMap.Items) == 0 {
		return nil, fmt.Errorf("configmap %s not found in server", constatns.DeployConfigConfigMapName)
	}

	data := configMap.Items[0].Data[constatns.DeployConfigConfigMapKey]
	if err = yaml.Unmarshal([]byte(data), dc); err != nil {
		return nil, err
	}
	if dc.Agents == nil {
		dc.Agents = make(options.Agents)
	}

	if syncLocal {
		if err = writeLocalDeployConfig(dc); err != nil {
			return nil, errors.WithMessage(err, "dump local deploy-config")
		}
	}
	return dc, nil
}

// UpdateDeployConfig updates the authoritative deployment configuration first.
// The local cache is refreshed only after the server accepts the update.
func UpdateDeployConfig(ctx context.Context, cli *kc.Client, deployConfig *options.DeployConfig, syncLocal bool) error {
	marshal, err := yaml.Marshal(deployConfig)
	if err != nil {
		return err
	}
	timeout, cancelFunc := context.WithTimeout(ctx, time.Second*10)
	defer cancelFunc()

	// kcctl join、drain will update online config after cmd success,so update with retry to avoid data inconsistency
	if err := utils.RetryFunc(
		timeout, component.Options{DryRun: false}, time.Second, "updateDeployConfig",
		func(ctx context.Context, _ component.Options) error {
			configMap, err := cli.DescribeConfigMap(ctx, constatns.DeployConfigConfigMapName)
			if err != nil {
				return errors.WithMessage(err, "get configmap")
			}
			if len(configMap.Items) == 0 {
				return fmt.Errorf("configmap %s not found in server", constatns.DeployConfigConfigMapName)
			}
			cm := configMap.Items[0]
			cm.Data[constatns.DeployConfigConfigMapKey] = string(marshal)
			if _, err = cli.UpdateConfigMap(ctx, &cm); err != nil {
				return errors.WithMessage(err, "update online deploy config")
			}
			return nil
		},
	); err != nil {
		return err
	}
	if !syncLocal {
		return nil
	}
	if err := writeLocalDeployConfig(deployConfig); err != nil {
		return errors.WithMessage(err, "dump local deploy config")
	}
	return nil
}

func writeLocalDeployConfig(deployConfig *options.DeployConfig) error {
	localCopy := *deployConfig
	localCopy.Config = ""
	return localCopy.Write()
}
