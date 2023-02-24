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

// GetDeployConfig get online deploy config and dump to local if we need.
func GetDeployConfig(ctx context.Context, cli *kc.Client, dump bool) (*options.DeployConfig, error) {
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

	if !dump {
		if err = dc.Write(); err != nil {
			return nil, errors.WithMessage(err, "dump local deploy-config")
		}
	}
	return dc, nil
}

// UpdateDeployConfig update online deploy config and dump to local if we need.
func UpdateDeployConfig(ctx context.Context, cli *kc.Client, deployConfig *options.DeployConfig, dump bool) error {
	if dump {
		if err := deployConfig.Write(); err != nil {
			return errors.WithMessage(err, "dump local deploy config")
		}
	}
	marshal, err := yaml.Marshal(deployConfig)
	if err != nil {
		return err
	}
	timeout, cancelFunc := context.WithTimeout(ctx, time.Second*10)
	defer cancelFunc()

	// kcctl join、drain will update online config after cmd success,so update with retry to avoid data inconsistency
	return utils.RetryFunc(timeout, component.Options{DryRun: false}, time.Second, "updateDeployConfig", func(ctx context.Context, opts component.Options) error {
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
	})
}
