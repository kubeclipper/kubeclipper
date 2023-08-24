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

package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cni"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	version         = "v1"
	packages        = "packages"
	kubeadmConfig   = "kubeadmConfig"
	controlPlane    = "controlPlane"
	clusterNode     = "clusterNode"
	cniInfo         = "cniInfo"
	health          = "health"
	container       = "container"
	kubectl         = "kubectl"
	kubectlTerminal = "kubectlTerminal"
	//KubeCertsCluVersion the version that kubeadm certs command appears
	KubeCertsCluVersion = "1.20"
)

type Runnable v1.Cluster

func (runnable *Runnable) GetStep(ctx context.Context, action v1.StepAction) ([]v1.Step, error) {
	err := runnable.Validate()
	if err != nil {
		return nil, err
	}
	switch action {
	case v1.ActionInstall:
		return runnable.GetInstallSteps(ctx)
	case v1.ActionUninstall:
		return runnable.GetUninstallSteps(ctx)
	case v1.ActionUpgrade:
		return runnable.GetUpgradeSteps(ctx)
	}

	return nil, nil
}

func (runnable *Runnable) Validate() error {
	if runnable == nil {
		return fmt.Errorf("kubeadm object is empty")
	}

	if len(runnable.Masters) == 0 {
		return fmt.Errorf("init step error, cluster contains at least one master node")
	}

	// check dualStack and ipv4
	switch runnable.CNI.Type {
	case "calico":
		if runnable.Networking.IPFamily == v1.IPFamilyDualStack &&
			len(runnable.Networking.Pods.CIDRBlocks) <= 1 {
			return fmt.Errorf("ipv4 and ipv6 cidr are both required when calico dual-stack is on")
		}
		if runnable.Networking.IPFamily != v1.IPFamilyDualStack &&
			len(runnable.Networking.Pods.CIDRBlocks) == 0 {
			return fmt.Errorf("calico ipv4 and ipv6 must have at least one")
		}
	}

	return nil
}

func (runnable *Runnable) GetInstallSteps(ctx context.Context) ([]v1.Step, error) {
	metadata := component.GetExtraMetadata(ctx)
	return runnable.makeInstallSteps(&metadata)
}

func (runnable *Runnable) GetUninstallSteps(ctx context.Context) ([]v1.Step, error) {
	metadata := component.GetExtraMetadata(ctx)
	return runnable.makeUninstallSteps(&metadata)
}

func (runnable *Runnable) GetUpgradeSteps(ctx context.Context) ([]v1.Step, error) {
	metadata := component.GetExtraMetadata(ctx)
	id := strutil.GetUUID()

	return []v1.Step{
		{
			ID:         id,
			Name:       fmt.Sprintf("step-%s", id),
			Timeout:    metav1.Duration{Duration: 3 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      utils.UnwrapNodeList(metadata.Masters),
		},
	}, nil
}

func (runnable *Runnable) makeInstallSteps(metadata *component.ExtraMetadata) ([]v1.Step, error) {
	// 1. package download and install
	// 2. print kubeadm config(template step type)
	// 3. kubeadm init cluster
	// 4. join node to cluster
	// 5. install cni(template and shell step type)
	// 6. patch label and taint(shell step type)
	// 7. check cluster health
	// 8. apply kubectl pod
	c := v1.Cluster(*runnable)
	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)

	var installSteps []v1.Step
	steps, err := EnvSetupSteps(nodes)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	ext := Extension{}
	steps, err = ext.InitStepper(&c).InstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	pack := Package{}
	steps, err = pack.InitStepper(&c).InstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	kubeConf := KubeadmConfig{}
	// TODO: No vip is currently introduced as controlPlaneEndpoint
	steps, err = kubeConf.InitStepper(&c, metadata).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	controlPlane := ControlPlane{}
	steps, err = controlPlane.InitStepper(&c).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	if len(runnable.Masters) > 1 {
		kubeadmConf := KubeadmConfig{}
		steps, err = kubeadmConf.InitStepper(&c, metadata).JoinSteps(true, utils.UnwrapNodeList(metadata.Masters)[1:])
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}
	if len(runnable.Workers) > 0 {
		kubeadmConf := KubeadmConfig{}
		steps, err = kubeadmConf.InitStepper(&c, metadata).JoinSteps(false, utils.UnwrapNodeList(metadata.Workers))
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}

	if len(runnable.Masters) > 1 {
		cluNode := ClusterNode{}
		steps, err = cluNode.InitStepper(&c, metadata).InstallSteps(NodeRoleMaster, utils.UnwrapNodeList(metadata.Masters)[1:])
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}
	if len(runnable.Workers) > 0 {
		cluNode := ClusterNode{}
		steps, err = cluNode.InitStepper(&c, metadata).InstallSteps(NodeRoleWorker, utils.UnwrapNodeList(metadata.Workers))
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}

	// when disable cni, only install k8s control plane and kubelet
	// user can install other cni and other plugin manually
	if metadata.OnlyInstallKubernetesComp {
		steps, err = PatchTaintAndLabelStep(runnable.Masters, runnable.Workers, metadata)
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
		return installSteps, nil
	}

	//cn := CNIInfo{}
	cf, err := cni.Load(c.CNI.Type)
	if err != nil {
		return nil, err
	}
	cniStepper := cf.Create().InitStep(metadata, &c.CNI, &c.Networking)
	if metadata.Offline {
		steps, err = cniStepper.LoadImage(nodes)
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}
	steps, err = cniStepper.InstallSteps([]v1.StepNode{masters[0]}, runnable.KubernetesVersion)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	steps, err = PatchTaintAndLabelStep(runnable.Masters, runnable.Workers, metadata)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	heal := Health{}
	steps, err = heal.InitStepper(c.KubernetesVersion).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	kt := KubectlTerminal{}
	steps, err = kt.InitStepper(&c).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)
	return installSteps, nil
}

func (runnable *Runnable) makeUninstallSteps(metadata *component.ExtraMetadata) ([]v1.Step, error) {
	// TODO: need refactor

	c := v1.Cluster(*runnable)
	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)

	var uninstallSteps []v1.Step

	// clean cluster pv storage resource
	controlPlane := ControlPlane{}
	steps, err := controlPlane.InitStepper(&c).UninstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// exec kubeadm reset
	steps, err = KubeadmReset(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// NOTE: clean container must after kubeadm reset,see #122450
	container := Container{}
	steps, err = container.InitStepper(c.ContainerRuntime.Type).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove Kubernetes all
	steps, err = Clear(&c, metadata)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove configuration files and rpm packages already installed
	pack := Package{}
	steps, err = pack.InitStepper(&c).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)
	// remove extension binary and images
	ext := Extension{}
	steps, err = ext.InitStepper(&c).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// clean virtual network interfaces
	steps, err = CleanCNI(metadata, &c.CNI, &c.Networking, nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove kubeconfig
	ctl := Kubectl{}
	steps, err = ctl.InitStepper().UninstallSteps(masters)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	heal := Health{}
	steps, err = heal.InitStepper(c.KubernetesVersion).UninstallSteps(&runnable.Networking, nodes...)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove hostname
	steps, err = RemoveHostname(&c, nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	return uninstallSteps, nil
}

func (stepper *Package) InitStepper(c *v1.Cluster) *Package {
	stepper.Offline = c.Offline()
	stepper.Version = c.KubernetesVersion
	stepper.CriType = c.ContainerRuntime.Type
	stepper.LocalRegistry = c.LocalRegistry
	stepper.KubeletDir = c.Kubelet.RootDir
	return stepper
}

func (stepper *Package) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(&stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "installPackages",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, packages, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *Package) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "unInstallPackages",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, packages, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *KubeadmConfig) InitStepper(c *v1.Cluster, metadata *component.ExtraMetadata) *KubeadmConfig {
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", c.Networking.DNSDomain)
	cpEndpoint := fmt.Sprintf("%s:6443", apiServerDomain)
	if _, ok := c.Labels[common.LabelClusterProviderName]; ok {
		// TODO: No vip is currently introduced as controlPlaneEndpoint
		cpEndpoint = fmt.Sprintf("%s:6443", metadata.Masters[0].NodeIPv4)
	}

	stepper.ClusterConfigAPIVersion = ""
	stepper.ContainerRuntime = c.ContainerRuntime.Type
	stepper.Etcd = c.Etcd
	stepper.Networking = c.Networking
	stepper.KubeProxy = c.KubeProxy
	stepper.Kubelet = c.Kubelet
	stepper.ClusterName = metadata.ClusterName
	stepper.KubernetesVersion = c.KubernetesVersion
	stepper.ControlPlaneEndpoint = cpEndpoint
	stepper.CertSANs = c.GetAllCertSANs()
	stepper.LocalRegistry = c.LocalRegistry
	stepper.Offline = metadata.Offline
	stepper.FeatureGates = c.FeatureGates
	// TODO: No vip is currently introduced as controlPlaneEndpoint
	stepper.AdvertiseAddress = metadata.Masters[0].NodeIPv4

	return stepper
}

func (stepper *KubeadmConfig) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	kubeadmBytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "renderKubeadmConfig",
			Timeout:    metav1.Duration{Duration: 1 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Commands: []v1.Command{
				{
					Type: v1.CommandTemplateRender,
					Template: &v1.TemplateCommand{
						Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, kubeadmConfig, version, component.TypeTemplate),
						Data:     kubeadmBytes,
					},
				},
			},
		},
	}, nil
}

func (stepper *KubeadmConfig) JoinSteps(isControlPlane bool, nodes []v1.StepNode) ([]v1.Step, error) {
	stepper.IsControlPlane = isControlPlane
	kubeadmBytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	step := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "renderWorkerJoinConfig",
		Timeout:    metav1.Duration{Duration: 1 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{

				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, kubeadmConfig, version, component.TypeStep),
				CustomCommand: kubeadmBytes,
			},
		},
	}
	if isControlPlane {
		step.Name = "renderMasterJoinConfig"
	}
	return []v1.Step{step}, nil

}

func (stepper *KubeadmConfig) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *ControlPlane) InitStepper(c *v1.Cluster) *ControlPlane {
	apiServerDomain := APIServerDomainPrefix +
		strutil.StringDefaultIfEmpty("cluster.local", c.Networking.DNSDomain)

	stepper.APIServerDomainName = apiServerDomain
	stepper.EtcdDataPath = c.Etcd.DataDir
	stepper.ContainerRuntime = c.ContainerRuntime.Type
	stepper.ExternalCaCert = c.ExternalCaCert
	stepper.ExternalCaKey = c.ExternalCaKey

	return stepper
}

func (stepper *ControlPlane) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "initControlPlane",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, controlPlane, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *ControlPlane) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	b, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "cleanStorage",
			Timeout:    metav1.Duration{Duration: 15 * time.Minute}, // maybe timeout, if there is a lot pv need to reclaim.
			ErrIgnore:  true,
			RetryTimes: 0,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, controlPlane, version, component.TypeStep),
					CustomCommand: b,
				},
			},
		},
	}, nil
}

func (stepper *ClusterNode) InitStepper(c *v1.Cluster, metadata *component.ExtraMetadata) *ClusterNode {
	apiServerDomain := APIServerDomainPrefix +
		strutil.StringDefaultIfEmpty("cluster.local", c.Networking.DNSDomain)

	stepper.NodeRole = ""
	stepper.WorkerNodeVIP = c.Networking.WorkerNodeVip
	stepper.Masters = metadata.GetMasterNodeIP()
	stepper.LocalRegistry = c.LocalRegistry
	stepper.APIServerDomainName = apiServerDomain
	// TODO: No vip is currently introduced as controlPlaneEndpoint
	stepper.JoinMasterIP = metadata.Masters[0].NodeIPv4
	stepper.EtcdDataPath = c.Etcd.DataDir

	return stepper
}

func GetKubeConfig(ctx context.Context, name string, node component.Node, externalAddress string, deliveryCmd service.CmdDelivery) (string, error) {
	content, err := deliveryCmd.DeliverCmd(ctx, node.ID, []string{"cat", "/etc/kubernetes/admin.conf"}, 3*time.Minute)
	if err != nil {
		logger.Errorf(" cat kubeConfig error: %s", err.Error())
		return "", err
	}

	cfg, err := clientcmd.NewClientConfigFromBytes(content)
	if err != nil {
		return "", err
	}
	kubeConfig, err := cfg.RawConfig()
	if err != nil {
		return "", err
	}

	if externalAddress != "" {
		kubeConfig.Clusters[name].Server = fmt.Sprintf("https://%s:6443", externalAddress)
	} else {
		kubeConfig.Clusters[name].Server = fmt.Sprintf("https://%s:6443", node.NodeIPv4)
	}
	config, err := clientcmd.Write(kubeConfig)
	if err != nil {
		return "", err
	}

	return string(config), nil
}

func (stepper *ClusterNode) InstallSteps(role string, nodes []v1.StepNode) ([]v1.Step, error) {
	stepper.setRole(role)
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "joinNode",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, clusterNode, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *ClusterNode) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *Health) InitStepper(version string) *Health {
	stepper.KubernetesVersion = version
	return stepper
}

func (stepper *Health) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	checkBytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	registerSaCommands, err := stepper.getRegisterServiceAccountCommands()
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "checkHealth",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 0,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, health, version, component.TypeStep),
					CustomCommand: checkBytes,
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "registerServiceAccount",
			Timeout:    metav1.Duration{Duration: 2 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands:   registerSaCommands,
		}}, nil
}

func (stepper *Health) UninstallSteps(network *v1.Networking, nodes ...v1.StepNode) ([]v1.Step, error) {
	if network.ProxyMode == "ipvs" {
		// ipvs mode
		var bytes []byte
		bytes, err := json.Marshal(stepper)
		if err != nil {
			return nil, err
		}
		return []v1.Step{
			{
				ID:         strutil.GetUUID(),
				Name:       "clearIPVS",
				Timeout:    metav1.Duration{Duration: 10 * time.Second},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes:      nodes,
				Action:     v1.ActionUninstall,
				Commands: []v1.Command{
					{
						Type:          v1.CommandCustom,
						Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, health, version, component.TypeStep),
						CustomCommand: bytes,
					},
				},
			},
			{
				ID:         strutil.GetUUID(),
				Name:       "removeDummyInterface",
				Timeout:    metav1.Duration{Duration: 5 * time.Second},
				ErrIgnore:  true,
				Nodes:      nodes,
				RetryTimes: 1,
				Action:     v1.ActionUninstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"ip", "link", "delete", "kube-ipvs0"},
					},
				},
			},
		}, nil
	}

	// iptables mode
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "flushRules",
			Timeout:    metav1.Duration{Duration: 10 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "iptables -F && iptables -t nat -F && iptables -t mangle -F && iptables -X"},
				},
			},
		},
	}, nil
}

func (stepper *Certification) InitStepper() *Certification {
	return stepper
}

func (stepper *Certification) InstallSteps(clu *v1.Cluster, nodes []v1.StepNode) ([]v1.Step, error) {
	var cmd []string
	if clu.KubernetesVersion[1:] < KubeCertsCluVersion {
		cmd = []string{"kubeadm", "alpha", "certs", "renew", "all"}
	} else {
		cmd = []string{"kubeadm", "certs", "renew", "all"}
	}

	step := []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "updateCerts",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: cmd,
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "restartPods",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			// TODO: can use configMap get path
			BeforeRunCommands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"mkdir", "-pv", "/tmp/.k8s/config"},
				},
			},
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "mv /etc/kubernetes/manifests/etcd.yaml /etc/kubernetes/manifests/kube-apiserver.yaml /etc/kubernetes/manifests/kube-controller-manager.yaml /etc/kubernetes/manifests/kube-scheduler.yaml /tmp/.k8s/config && sleep 20"},
				},
			},
			AfterRunCommands: []v1.Command{
				{
					Type: v1.CommandShell,
					ShellCommand: []string{
						"mv",
						"/tmp/.k8s/config/etcd.yaml",
						"/tmp/.k8s/config/kube-apiserver.yaml",
						"/tmp/.k8s/config/kube-controller-manager.yaml",
						"/tmp/.k8s/config/kube-scheduler.yaml",
						"/etc/kubernetes/manifests",
					},
				},
			},
		},
	}

	return step, nil
}

func (stepper *Container) InitStepper(criType string) *Container {
	stepper.CriType = criType
	return stepper
}

func (stepper *Container) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *Container) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	b, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "cleanContainer",
			Timeout:    metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore:  true,
			RetryTimes: 0,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, container, version, component.TypeStep),
					CustomCommand: b,
				},
			},
		},
	}, nil
}

func (stepper *Kubectl) InitStepper() *Kubectl {
	return stepper
}

func (stepper *Kubectl) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *Kubectl) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "removeKubeConfig",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, kubectl, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func EnvSetupSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	var steps []v1.Step
	steps = append(steps, v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "nodeEnvSetup",
		Timeout:    metav1.Duration{Duration: 10 * time.Second},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      nodes,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", nodeScript},
			},
		},
	})

	return steps, nil
}

func PatchTaintAndLabelStep(master, workers v1.WorkerNodeList, metadata *component.ExtraMetadata) ([]v1.Step, error) {
	var shellCommand []v1.Command

	for _, v := range master {
		hostname := metadata.GetMasterHostname(v.ID)
		if len(v.Taints) == 0 {
			shellCommand = append(shellCommand, v1.Command{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("kubectl taint node %s node-role.kubernetes.io/master- || true", hostname)},
			}, v1.Command{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("kubectl taint node %s node-role.kubernetes.io/control-plane- || true", hostname)},
			})
		} else {
			for _, t := range v.Taints {
				shellCommand = append(shellCommand, v1.Command{
					Type:         v1.CommandShell,
					ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("kubectl taint node %s %s=%s:%s || true", hostname, t.Key, t.Value, t.Effect)},
				})
			}
		}
		if len(v.Labels) != 0 {
			for key, value := range v.Labels {
				shellCommand = append(shellCommand, v1.Command{
					Type:         v1.CommandShell,
					ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("kubectl label node %s %s=%s", hostname, key, value)},
				})
			}
		}
	}

	for _, v := range workers {
		hostname := metadata.GetWorkerHostname(v.ID)
		if len(v.Labels) != 0 {
			for key, value := range v.Labels {
				shellCommand = append(shellCommand, v1.Command{
					Type:         v1.CommandShell,
					ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("kubectl label node %s %s=%s", hostname, key, value)},
				})
			}
		}
	}

	if len(shellCommand) > 0 {
		avaMasters := metadata.Masters
		var err error
		if len(metadata.Masters) > 1 && metadata.ClusterStatus != "" {
			avaMasters, err = metadata.Masters.AvailableKubeMasters()
			if err != nil {
				return nil, err
			}
		}
		return []v1.Step{
			{
				ID:         strutil.GetUUID(),
				Name:       "updateNodeMetadata",
				Timeout:    metav1.Duration{Duration: 1 * time.Minute},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes: []v1.StepNode{
					{
						ID:       avaMasters[0].ID,
						IPv4:     avaMasters[0].IPv4,
						NodeIPv4: avaMasters[0].NodeIPv4,
						Hostname: metadata.GetMasterHostname(avaMasters[0].ID),
					},
				},
				Commands: shellCommand,
			},
		}, nil
	}

	return nil, nil
}

func KubeadmReset(nodes []v1.StepNode) ([]v1.Step, error) {
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "kubeadmReset",
			Timeout:    metav1.Duration{Duration: 3 * 60 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubeadm", "reset", "-f"},
				},
			},
		},
	}, nil
}

func Clear(c *v1.Cluster, metadata *component.ExtraMetadata) ([]v1.Step, error) {
	var steps []v1.Step
	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)
	workers := utils.UnwrapNodeList(metadata.Workers)
	// remove etcd data dir
	steps = append(steps,
		doCommandRemoveStep("clearDatabase", masters,
			c.Etcd.DataDir))
	kubeletDataDir := KubeletDefaultDataDir
	if c.Kubelet.RootDir != "" {
		kubeletDataDir = c.Kubelet.RootDir
	}
	steps = append(steps,
		doCommandRemoveStep("removeKubeletDataDir", nodes, kubeletDataDir),
		doCommandRemoveStep("removeDockershimDataDir", nodes, DockershimDefaultDataDir),
	)

	// clean CNI config
	steps = append(steps,
		doCommandRemoveStep("cleanCNIConfig", nodes, CniDefaultConfigDir),
		doCommandRemoveStep("removeCNIData", nodes, CniDefaultConfigDir),
		doCommandRemoveStep("removeCNIRunData", nodes, CniDefaultConfigDir))

	// clean Kubernetes config
	steps = append(steps,
		doCommandRemoveStep("removeKubernetesConfig", masters, K8SDefaultConfigDir))

	// clear worker /etc/hosts vip domain
	// sed -i '/apiserver.cluster.local/d' /etc/hosts
	if len(c.Workers) > 0 {
		apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local",
			c.Networking.DNSDomain)
		steps = append(steps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "clearVIPDomain",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      workers,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", fmt.Sprintf("sed -i '/%s/d' /etc/hosts", apiServerDomain)},
				},
			},
		})
	}

	return steps, nil
}

// CleanCNI clean cni image and cni network veth
func CleanCNI(metadata *component.ExtraMetadata, c *v1.CNI, networking *v1.Networking, nodes []v1.StepNode) ([]v1.Step, error) {
	cf, err := cni.Load(c.Type)
	if err != nil {
		logger.Debugf("clean cni error: %v", err)
		return nil, nil
	}

	return cf.Create().InitStep(metadata, c, networking).UninstallSteps(nodes)
}

func RemoveHostname(c *v1.Cluster, nodes []v1.StepNode) ([]v1.Step, error) {
	var steps []v1.Step
	apiServerDomain := APIServerDomainPrefix +
		strutil.StringDefaultIfEmpty("cluster.local", c.Networking.DNSDomain)

	steps = append(steps, v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "removeHostname",
		Timeout:    metav1.Duration{Duration: 5 * time.Second},
		ErrIgnore:  true,
		RetryTimes: 1,
		Nodes:      nodes,
		Action:     v1.ActionUninstall,
		Commands: []v1.Command{
			{
				Type: v1.CommandShell,
				ShellCommand: []string{"bash", "-c", fmt.Sprintf("sed -i -e '/%s/d' /etc/hosts",
					apiServerDomain)},
			},
		},
	})

	return steps, nil
}

func (stepper *KubectlTerminal) InitStepper(c *v1.Cluster) *KubectlTerminal {
	stepper.ImageRegistryAddr = c.LocalRegistry
	return stepper
}

func (stepper *KubectlTerminal) InstallSteps(stepMaster0 []v1.StepNode) ([]v1.Step, error) {
	installSteps := make([]v1.Step, 0)
	terminal, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "applyKubectlPod",
		Timeout:    metav1.Duration{Duration: 10 * time.Second},
		ErrIgnore:  true,
		RetryTimes: 1,
		Nodes:      stepMaster0,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type: v1.CommandTemplateRender,
				Template: &v1.TemplateCommand{
					Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, kubectlTerminal, version, component.TypeTemplate),
					Data:     terminal,
				},
			},
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(ManifestDir, "kc-kubectl.yaml")},
			},
		},
	})
	return installSteps, nil
}

func (stepper *KubectlTerminal) UninstallSteps() ([]v1.Step, error) {
	return nil, fmt.Errorf("KubectlTerminal dose not support uninstall")
}

func (stepper *SAN) InitStepper() *SAN {
	return stepper
}

func (stepper *SAN) InstallSteps(nodes []v1.StepNode, sans []string) ([]v1.Step, error) {
	updater := KubeadmConfigUpdater{
		SANs:       sans,
		ConfigFile: "/tmp/.k8s/kubeadm-new.yaml",
	}
	stepData, err := json.Marshal(&updater)
	if err != nil {
		return nil, fmt.Errorf("step marshal:%w", err)
	}
	checkHealth := &Health{}
	checkBytes, err := json.Marshal(checkHealth)
	if err != nil {
		return nil, err
	}

	step := []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "getKubeadmConfig",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: time.Second * 30},
			ErrIgnore:  false,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "kubectl -n kube-system get configmap kubeadm-config -o jsonpath='{.data.ClusterConfiguration}' > /tmp/.k8s/kubeadm-new.yaml"},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "updateKubeadmConfig",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: time.Second * 30},
			ErrIgnore:  false,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, kubeadmConfigUpdaterName, kubeadmConfigUpdaterVersion, component.TypeStep),
					CustomCommand: stepData,
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "backupCerts",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: time.Second * 10},
			ErrIgnore:  false,
			RetryTimes: 1,
			BeforeRunCommands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"mkdir", "-pv", "/tmp/.k8s/pki"},
				},
			},
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "mv -f /etc/kubernetes/pki/apiserver.{crt,key} /tmp/.k8s/pki/"},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "generateCerts",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: time.Second * 30},
			ErrIgnore:  false,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubeadm", "init", "phase", "certs", "apiserver", "--config", "/tmp/.k8s/kubeadm-new.yaml"},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "restartPods",
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			BeforeRunCommands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"mkdir", "-pv", "/tmp/.k8s/config"},
				},
			},
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", "mv -f /etc/kubernetes/manifests/kube-apiserver.yaml /tmp/.k8s/config && sleep 20"},
				},
			},
			AfterRunCommands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"mv", "/tmp/.k8s/config/kube-apiserver.yaml", "/etc/kubernetes/manifests"},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "checkHealth",
			Timeout:    metav1.Duration{Duration: 3 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, health, version, component.TypeStep),
					CustomCommand: checkBytes,
				},
			},
		},
	}

	return step, nil
}

type KubeadmConfigUpdater struct {
	SANs       []string
	ConfigFile string
}

const (
	kubeadmConfigUpdaterName    = "KubeadmConfigUpdater"
	kubeadmConfigUpdaterVersion = "v1"
)

func (d *KubeadmConfigUpdater) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	// update kubeadm config
	buf, err := os.ReadFile(d.ConfigFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config file: %s failed: %w", d.ConfigFile, err)
	}

	var config ClusterConfiguration
	if err = yaml.Unmarshal(buf, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config file: %s failed: %w", d.ConfigFile, err)
	}
	config.APIServer.CertSANs = sets.NewString(d.SANs...).List()
	marshal, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	if !opts.DryRun {
		if err = os.WriteFile(d.ConfigFile, marshal, 0600); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (d *KubeadmConfigUpdater) Uninstall(_ context.Context, _ component.Options) ([]byte, error) {
	// nothing
	return nil, nil
}

func (d *KubeadmConfigUpdater) NewInstance() component.ObjectMeta {
	return new(KubeadmConfigUpdater)
}
