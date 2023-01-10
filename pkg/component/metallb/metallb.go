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

package metallb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"

	"github.com/kubeclipper/kubeclipper/pkg/component/common"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/component/validation"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

func init() {
	l := &MetalLB{}
	if err := component.Register(fmt.Sprintf(component.RegisterFormat, name, version), l); err != nil {
		panic(err)
	}

	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat, name, version, metallb), l); err != nil {
		panic(err)
	}

	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, name, version, metallb), l); err != nil {
		panic(err)
	}

	if err := initI18nForComponentMeta(); err != nil {
		panic(err)
	}
}

var (
	_ component.Interface      = (*MetalLB)(nil)
	_ component.TemplateRender = (*MetalLB)(nil)
	_ component.StepRunnable   = (*MetalLB)(nil)
)

const (
	metallb        = "metallb"
	name           = "metallb"
	version        = "v1"
	namespace      = "metallb-system"
	filename       = "metallb.yaml"
	ipAddressPool  = "ip-address-pool.yaml"
	advertisement  = "advertisement.yaml"
	manifestsDir   = "/tmp/.metallb"
	defaultMode    = "L2"
	defaultVersion = "v0.13.7"
)

var (
	errInvalidAddressPool = errors.New("invalid ip address pool")
)

type MetalLB struct {
	ImageRepoMirror                            string   `json:"imageRepoMirror"` // optional
	ManifestsDir                               string   `json:"manifestsDir"`    // optional
	Mode                                       string   `json:"mode"`            // required
	Addresses                                  []string `json:"addresses"`       // required
	Version                                    string   `json:"version"`         // optional
	installSteps, uninstallSteps, upgradeSteps []v1.Step
}

func (n *MetalLB) Ns() string {
	return namespace
}

func (n *MetalLB) Svc() string {
	return ""
}

func (n *MetalLB) RequestPath() string {
	return ""
}

// Supported TODO: add metallb healthy condition check
func (n *MetalLB) Supported() bool {
	return false
}

func (n *MetalLB) GetInstanceName() string {
	return name
}

func (n *MetalLB) RequireExtraCluster() []string {
	return nil
}

func (n *MetalLB) CompleteWithExtraCluster(extra map[string]component.ExtraMetadata) error {
	return nil
}

func (n *MetalLB) Validate() error {
	// load balancer mode
	if err := validation.MatchLoadBalancerMode(n.Mode); err != nil {
		return err
	}

	// valid ip address pool
	for _, address := range n.Addresses {
		if strings.Contains(address, "/") {
			if _, _, err := net.ParseCIDR(address); err != nil {
				return errInvalidAddressPool
			}
		} else if strings.Contains(address, "-") {
			for _, ip := range strings.Split(address, "-") {
				if !netutil.IsValidIP(ip) {
					return errInvalidAddressPool
				}
			}
		} else {
			return errInvalidAddressPool
		}
	}

	return nil
}

func (n *MetalLB) InitSteps(ctx context.Context) error {
	metadata := component.GetExtraMetadata(ctx)
	// when the component does not specify an ImageRepoMirror, the cluster LocalRegistry is inherited
	if n.ImageRepoMirror == "" {
		n.ImageRepoMirror = metadata.LocalRegistry
	}
	if metadata.Offline && n.ImageRepoMirror == "" {
		// TODO: arch is unnecessary, version can be configured
		imager := &common.Imager{
			PkgName: metallb,
			Version: defaultVersion,
			CriName: metadata.CRI,
			Offline: metadata.Offline,
		}
		steps, err := imager.InstallSteps(metadata.GetAllNodes())
		if err != nil {
			return err
		}
		n.installSteps = append(n.installSteps, steps...)
	}

	master := utils.UnwrapNodeList(metadata.Masters[:1])
	if len(metadata.Masters) > 1 && metadata.ClusterStatus == v1.ClusterRunning {
		avaMasters, err := metadata.Masters.AvailableKubeMasters()
		if err != nil {
			return err
		}
		master = utils.UnwrapNodeList(avaMasters[:1])
	}
	// strict ARP
	arpStep := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "enableKubeProxyStrictARP",
		Timeout:    metav1.Duration{Duration: 3 * time.Second},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      master,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"bash", "-c", `kubectl get configmap kube-proxy -n kube-system -o yaml | sed -e "s/strictARP: false/strictARP: true/" | kubectl apply -f - -n kube-system`},
			},
		},
	}
	// render step
	bytes, err := json.Marshal(n)
	if err != nil {
		return err
	}
	rs := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "renderMetalLBManifests",
		Timeout:    metav1.Duration{Duration: 3 * time.Second},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      master,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type: v1.CommandTemplateRender,
				Template: &v1.TemplateCommand{
					Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, name, version, metallb),
					Data:     bytes,
				},
			},
		},
	}

	n.installSteps = append(n.installSteps, []v1.Step{
		arpStep,
		rs,
		{
			ID:         strutil.GetUUID(),
			Name:       "deployMetalLB",
			Timeout:    metav1.Duration{Duration: 10 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, filename)},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "checkMetalLBHealth",
			Timeout:    metav1.Duration{Duration: 1 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, name, version, metallb),
					CustomCommand: bytes,
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "deployIPAddressPool",
			Timeout:    metav1.Duration{Duration: 15 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "sleep 10s"},
				},
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, ipAddressPool)},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "deployAdvertisement",
			Timeout:    metav1.Duration{Duration: 3 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, advertisement)},
				},
			},
		},
	}...)

	// uninstall
	if metadata.OperationType != v1.OperationDeleteCluster {
		n.uninstallSteps = []v1.Step{
			rs,
			{
				ID:         strutil.GetUUID(),
				Name:       "removeMetalLB",
				Timeout:    metav1.Duration{Duration: 5 * time.Minute},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes:      master,
				Action:     v1.ActionUninstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"kubectl", "delete", "-f", filepath.Join(n.ManifestsDir, filename)},
					},
				},
			},
			{
				ID:         strutil.GetUUID(),
				Name:       "disableKubeProxyStrictARP",
				Timeout:    metav1.Duration{Duration: 3 * time.Second},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes:      master,
				Action:     v1.ActionInstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"bash", "-c", `kubectl get configmap kube-proxy -n kube-system -o yaml | sed -e "s/strictARP: true/strictARP: false/" | kubectl apply -f - -n kube-system`},
					},
				},
			},
		}
	}

	return nil
}

func (n *MetalLB) GetName() string {
	return name
}

func (n *MetalLB) GetVersion() string {
	return version
}

func (n *MetalLB) GetComponentMeta(lang component.Lang) component.Meta {
	loc := component.GetLocalizer(lang)
	mode := component.JSON(defaultMode)

	propMap := map[string]component.JSONSchemaProps{
		"mode": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "metallb.mode"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      mode,
			Description:  "load balancer mode",
			Priority:     2,
			Dependencies: []string{"enabled"},
			EnumNames:    []string{"L2", "BGP"},
			Enum:         []component.JSON{"L2", "BGP"},
		},
		"addresses": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "metallb.addresses"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeArray,
			Default:      nil,
			Description:  "A list of IP address ranges over which MetalLB has authority. You can list multiple ranges in a single pool, they will all share the same settings. Each range can be either a CIDR prefix, or an explicit start-end range of IPs.",
			Priority:     3,
			Dependencies: []string{"enabled"},
		},
		"imageRepoMirror": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "metallb.imageRepoMirror"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      nil,
			Description:  "metallb image repository mirror, the component official repository is used by default",
			Priority:     4,
			Dependencies: []string{"enabled"},
		},
	}

	return component.Meta{
		Title:      loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "metallb.metaTitle"}),
		Name:       name,
		Version:    version,
		Unique:     true,
		Template:   true,
		Deprecated: false,
		Dependence: []string{component.InternalCategoryKubernetes},
		Category:   component.InternalCategoryLB,
		Priority:   1,
		Schema: &component.JSONSchemaProps{
			Properties: propMap,
			Required:   []string{"mode", "addresses"},
			Type:       component.JSONSchemaTypeObject,
			Default:    nil,
		},
	}
}

func (n *MetalLB) NewInstance() component.ObjectMeta {
	return &MetalLB{
		ManifestsDir: manifestsDir,
		Version:      defaultVersion,
	}
}

func (n *MetalLB) GetDependence() []string {
	return []string{component.InternalCategoryKubernetes}
}

func (n *MetalLB) GetInstallSteps() []v1.Step {
	return n.installSteps
}

func (n *MetalLB) GetUninstallSteps() []v1.Step {
	return n.uninstallSteps
}

func (n *MetalLB) GetUpgradeSteps() []v1.Step {
	return n.upgradeSteps
}

func (n *MetalLB) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	if err := utils.RetryFunc(ctx, opts, 5*time.Second, "checkMetalLBPodStatus", n.checkMetalLBPodStatus); err != nil {
		return nil, err
	}
	return nil, nil
}

func (n *MetalLB) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (n *MetalLB) checkMetalLBPodStatus(ctx context.Context, opts component.Options) error {
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", fmt.Sprintf("kubectl get po -n %s | grep %s", namespace, "controller"))
	if err != nil {
		return err
	}

	if ec.StdOut() == "" {
		return fmt.Errorf("there are no running metallb pods: %s", strings.Join(ec.Args[1:], ","))
	}
	return nil
}

func (n *MetalLB) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, manifestTemplate, n)
	return err
}

func (n *MetalLB) renderIPAddressPool(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, ipAddressPoolTemplate, n)
	return err
}

func (n *MetalLB) renderAdvertisement(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, advertisementTemplate, n)
	return err
}

func (n *MetalLB) Render(ctx context.Context, opts component.Options) error {
	if err := os.MkdirAll(n.ManifestsDir, 0755); err != nil {
		return err
	}
	ipAddressPoolFile := filepath.Join(n.ManifestsDir, ipAddressPool)
	if err := fileutil.WriteFileWithContext(ctx, ipAddressPoolFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderIPAddressPool, opts.DryRun); err != nil {
		return err
	}
	advertisementFile := filepath.Join(n.ManifestsDir, advertisement)
	if err := fileutil.WriteFileWithContext(ctx, advertisementFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderAdvertisement, opts.DryRun); err != nil {
		return err
	}
	manifestsFile := filepath.Join(n.ManifestsDir, filename)
	return fileutil.WriteFileWithContext(ctx, manifestsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderTo, opts.DryRun)
}

// GetImageRepoMirror return ImageRepoMirror
func (n *MetalLB) GetImageRepoMirror() string {
	return n.ImageRepoMirror
}
