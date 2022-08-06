package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, name, version, insecureRegistry), &AddInsecureRegistryToCRI{}); err != nil {
		panic(err)
	}
}

var (
	_ component.StepRunnable = (*AddInsecureRegistryToCRI)(nil)
)

const (
	name             = "insecure-registry"
	version          = "v1"
	insecureRegistry = "add-insecure-registry"
)

type AddInsecureRegistryToCRI struct {
	CriType  string
	Registry string
}

func (n *AddInsecureRegistryToCRI) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	if err := utils.AddOrRemoveInsecureRegistryToCRI(ctx, n.CriType, n.Registry, true, opts.DryRun); err != nil {
		logger.Error("add insecure registry to CRI failed", zap.Error(err))
		return nil, err
	}
	return nil, nil
}

func (n *AddInsecureRegistryToCRI) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (n *AddInsecureRegistryToCRI) NewInstance() component.ObjectMeta {
	return &AddInsecureRegistryToCRI{}
}

// GetAddInsecureRegistry get the common step
func GetAddInsecureRegistry(nodes component.NodeList, criType, registry string) (v1.Step, error) {
	addInsecureRegistry := &AddInsecureRegistryToCRI{
		CriType:  criType,
		Registry: registry,
	}
	aData, err := json.Marshal(addInsecureRegistry)
	if err != nil {
		return v1.Step{}, err
	}
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "addInsecureRegistryToCRI",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      utils.UnwrapNodeList(nodes),
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, name, version, insecureRegistry),
				CustomCommand: aData,
			},
		},
	}, nil
}
