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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

const (
	version         = "v1"
	packages        = "packages"
	kubeadmConfig   = "kubeadmConfig"
	controlPlane    = "controlPlane"
	clusterNode     = "clusterNode"
	cni             = "cni"
	health          = "health"
	container       = "container"
	kubectl         = "kubectl"
	kubectlTerminal = "kubectlTerminal"
)

type KubeadmRunnable v1.Kubeadm

func (runnable *KubeadmRunnable) GetStep(ctx context.Context, action v1.StepAction) ([]v1.Step, error) {
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

func (runnable *KubeadmRunnable) Validate() error {
	if runnable == nil {
		return fmt.Errorf("kubeadm object is empty")
	}

	if len(runnable.Masters) == 0 {
		return fmt.Errorf("init step error, cluster contains at least one master node")
	}

	// check dualStack and ipv4
	switch runnable.KubeComponents.CNI.Type {
	case "calico":
		if runnable.KubeComponents.CNI.Calico.DualStack &&
			(runnable.KubeComponents.CNI.PodIPv4CIDR == "" || runnable.KubeComponents.CNI.PodIPv6CIDR == "") {
			return fmt.Errorf("ipv4 and ipv6 cidr are both required when calico dual-stack is on")
		}
		if !runnable.KubeComponents.CNI.Calico.DualStack &&
			runnable.KubeComponents.CNI.PodIPv4CIDR == "" &&
			runnable.KubeComponents.CNI.PodIPv6CIDR == "" {
			return fmt.Errorf("calico ipv4 and ipv6 must have at least one")
		}
	}

	return nil
}

func (runnable *KubeadmRunnable) GetInstallSteps(ctx context.Context) ([]v1.Step, error) {
	metadata := component.GetExtraMetadata(ctx)
	return runnable.makeInstallSteps(&metadata)
}

func (runnable *KubeadmRunnable) GetUninstallSteps(ctx context.Context) ([]v1.Step, error) {
	metadata := component.GetExtraMetadata(ctx)
	return runnable.makeUninstallSteps(&metadata)
}

func (runnable *KubeadmRunnable) GetUpgradeSteps(ctx context.Context) ([]v1.Step, error) {
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

func (runnable *KubeadmRunnable) makeInstallSteps(metadata *component.ExtraMetadata) ([]v1.Step, error) {
	// 1. package download and install
	// 2. print kubeadm config(template step type)
	// 3. kubeadm init cluster
	// 4. join node to cluster
	// 5. install cni(template and shell step type)
	// 6. patch label and taint(shell step type)
	// 7. check cluster health
	// 8. apply kubectl pod

	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)
	kubeadm := (*v1.Kubeadm)(runnable)

	var installSteps []v1.Step
	steps, err := EnvSetupSteps(nodes)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	pack := Package{}
	steps, err = pack.InitStepper(kubeadm).InstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	kubeConf := KubeadmConfig{}
	steps, err = kubeConf.InitStepper(kubeadm, metadata).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	controlPlane := ControlPlane{}
	steps, err = controlPlane.InitStepper(kubeadm).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	if len(kubeadm.Masters) > 1 {
		cluNode := ClusterNode{}
		steps, err = cluNode.InitStepper(kubeadm, metadata).InstallSteps(NodeRoleMaster, utils.UnwrapNodeList(metadata.Masters)[1:])
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}
	if len(kubeadm.Workers) > 0 {
		cluNode := ClusterNode{}
		steps, err = cluNode.InitStepper(kubeadm, metadata).InstallSteps(NodeRoleWorker, utils.UnwrapNodeList(metadata.Workers))
		if err != nil {
			return nil, err
		}
		installSteps = append(installSteps, steps...)
	}

	c := CNI{}
	steps, err = c.InitStepper(kubeadm).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	steps, err = PatchTaintAndLabelStep(kubeadm.Masters, kubeadm.Workers, metadata)
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	heal := Health{}
	steps, err = heal.InitStepper(kubeadm).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)

	kt := KubectlTerminal{}
	steps, err = kt.InitStepper(kubeadm).InstallSteps([]v1.StepNode{masters[0]})
	if err != nil {
		return nil, err
	}
	installSteps = append(installSteps, steps...)
	return installSteps, nil
}

func (runnable *KubeadmRunnable) makeUninstallSteps(metadata *component.ExtraMetadata) ([]v1.Step, error) {
	// TODO: need refactor

	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)
	kubeadm := (*v1.Kubeadm)(runnable)

	var uninstallSteps []v1.Step

	// clean cluster pv storage resource
	controlPlane := ControlPlane{}
	steps, err := controlPlane.InitStepper(kubeadm).UninstallSteps([]v1.StepNode{masters[0]})
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
	steps, err = container.InitStepper(kubeadm).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove Kubernetes all
	steps, err = Clear(kubeadm, metadata)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove configuration files and rpm packages already installed
	pack := Package{}
	steps, err = pack.InitStepper(kubeadm).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// clean virtual network interfaces
	c := CNI{}
	steps, err = c.InitStepper(kubeadm).UninstallSteps(nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove kubeconfig
	ctl := Kubectl{}
	steps, err = ctl.InitStepper(kubeadm).UninstallSteps(masters)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	heal := Health{}
	steps, err = heal.InitStepper(kubeadm).UninstallSteps(&kubeadm.KubeComponents.KubeProxy, nodes...)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	// remove hostname
	steps, err = RemoveHostname(kubeadm, nodes)
	if err != nil {
		return nil, err
	}
	uninstallSteps = append(uninstallSteps, steps...)

	return uninstallSteps, nil
}

func (stepper *Package) InitStepper(kubeadm *v1.Kubeadm) *Package {
	stepper.Arch = ""
	stepper.Offline = kubeadm.Offline
	stepper.Version = kubeadm.KubernetesVersion
	stepper.CriType = kubeadm.ContainerRuntime.Type.String()
	stepper.LocalRegistry = kubeadm.LocalRegistry
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
			Name:       "installPackages",
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

func (stepper *KubeadmConfig) InitStepper(kubeadm *v1.Kubeadm, metadata *component.ExtraMetadata) *KubeadmConfig {
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", kubeadm.Networking.DNSDomain)
	cpEndpoint := fmt.Sprintf("%s:6443", apiServerDomain)

	stepper.ClusterConfigAPIVersion = ""
	stepper.ContainerRuntime = kubeadm.ContainerRuntime.Type.String()
	stepper.Etcd = kubeadm.KubeComponents.Etcd
	stepper.Network = kubeadm.Networking
	stepper.KubeProxy = kubeadm.KubeComponents.KubeProxy
	stepper.Kubelet = kubeadm.KubeComponents.Kubelet
	stepper.ClusterName = metadata.ClusterName
	stepper.KubernetesVersion = kubeadm.KubernetesVersion
	stepper.ControlPlaneEndpoint = cpEndpoint
	stepper.CertSANs = kubeadm.CertSANs
	stepper.LocalRegistry = kubeadm.LocalRegistry
	stepper.WorkerNodeVip = kubeadm.WorkerNodeVip

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

func (stepper *KubeadmConfig) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *ControlPlane) InitStepper(kubeadm *v1.Kubeadm) *ControlPlane {
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", kubeadm.Networking.DNSDomain)

	stepper.APIServerDomainName = apiServerDomain
	stepper.EtcdDataPath = kubeadm.KubeComponents.Etcd.DataDir
	stepper.ContainerRuntime = kubeadm.ContainerRuntime.Type.String()

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

func (stepper *ClusterNode) InitStepper(kubeadm *v1.Kubeadm, metadata *component.ExtraMetadata) *ClusterNode {
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", kubeadm.Networking.DNSDomain)

	stepper.NodeRole = ""
	stepper.WorkerNodeVIP = kubeadm.WorkerNodeVip
	stepper.Masters = metadata.GetMasterNodeIP()
	stepper.LocalRegistry = kubeadm.LocalRegistry
	stepper.APIServerDomainName = apiServerDomain
	stepper.JoinMasterIP = metadata.Masters[0].IPv4
	stepper.EtcdDataPath = kubeadm.KubeComponents.Etcd.DataDir

	return stepper
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

func (stepper *CNI) InitStepper(kubeadm *v1.Kubeadm) *CNI {
	c := CNI(kubeadm.KubeComponents.CNI)
	*stepper = c
	return stepper
}

func (stepper *CNI) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "installCNI",
			Timeout:    metav1.Duration{Duration: 1 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Commands: []v1.Command{
				{
					Type: v1.CommandTemplateRender,
					Template: &v1.TemplateCommand{
						Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, cni, version, component.TypeTemplate),
						Data:     bytes,
					},
				},
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(ManifestDir, "cni.yaml")},
				},
			},
		},
	}, nil
}

func (stepper *CNI) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, nil
}

func (stepper *Health) InitStepper(kubeadm *v1.Kubeadm) *Health {
	return stepper
}

func (stepper *Health) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	checkBytes, err := json.Marshal(stepper)
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
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "create", "sa", "kc-server", "-n", "kube-system"},
				},
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "create", "clusterrolebinding", "kc-server", "--clusterrole=cluster-admin", "--serviceaccount=kube-system:kc-server"},
				},
			},
		}}, nil
}

func (stepper *Health) UninstallSteps(proxy *v1.KubeProxy, nodes ...v1.StepNode) ([]v1.Step, error) {
	if proxy.IPvs {
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
	rawCmds := []string{
		"iptables -F",
		"iptables -t nat -F",
		"iptables -t mangle -F",
		"iptables -X",
		// append command line here
	}
	cmds := make([]v1.Command, len(rawCmds))
	for i, cmd := range rawCmds {
		cmds[i] = v1.Command{
			Type:         v1.CommandShell,
			ShellCommand: strings.Split(cmd, " "),
		}
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "flushRules",
			Timeout:    metav1.Duration{Duration: 10 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Action:     v1.ActionUninstall,
			Commands:   cmds,
		},
	}, nil
}

func (stepper *Container) InitStepper(kubeadm *v1.Kubeadm) *Container {
	stepper.CriType = kubeadm.ContainerRuntime.Type.String()
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

func (stepper *Kubectl) InitStepper(kubeadm *v1.Kubeadm) *Kubectl {
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
				Type: v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", `
systemctl stop firewalld || true
systemctl disable firewalld || true
setenforce 0
sed -i s/^SELINUX=.*$/SELINUX=disabled/ /etc/selinux/config
modprobe br_netfilter && modprobe nf_conntrack
cat > /etc/sysctl.d/k8s.conf << EOF
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward=1
EOF
cat > /etc/sysctl.conf << EOF
net.ipv6.conf.all.forwarding=1
fs.file-max = 100000
vm.max_map_count=262144
EOF
cat > /etc/security/limits.conf << EOF
#IncreaseMaximumNumberOfFileDescriptors
* soft nproc 65535
* hard nproc 65535
* soft nofile 65535
* hard nofile 65535
#IncreaseMaximumNumberOfFileDescriptors
EOF
sysctl --system
sysctl -p
swapoff -a
sed -i /swap/d /etc/fstab`},
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
		return []v1.Step{
			{
				ID:         strutil.GetUUID(),
				Name:       "updateNodeMetadata",
				Timeout:    metav1.Duration{Duration: 1 * time.Minute},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes: []v1.StepNode{
					{
						ID: metadata.Masters[0].ID, IPv4: metadata.Masters[0].IPv4, Hostname: metadata.GetMasterHostname(metadata.Masters[0].ID),
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

func Clear(kubeadm *v1.Kubeadm, metadata *component.ExtraMetadata) ([]v1.Step, error) {
	var steps []v1.Step
	nodes := utils.UnwrapNodeList(metadata.GetAllNodes())
	masters := utils.UnwrapNodeList(metadata.Masters)
	workers := utils.UnwrapNodeList(metadata.Workers)
	// remove etcd data dir
	steps = append(steps,
		doCommandRemoveStep("clearDatabase", masters,
			kubeadm.KubeComponents.Etcd.DataDir))
	kubeletDataDir := KubeletDefaultDataDir
	if kubeadm.KubeComponents.Kubelet.RootDir != "" {
		kubeletDataDir = kubeadm.KubeComponents.Kubelet.RootDir
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
	if len(kubeadm.Workers) > 0 {
		apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", kubeadm.Networking.DNSDomain)
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

func CleanCNI(c *v1.CNI, nodes []v1.StepNode) ([]v1.Step, error) {
	switch c.Type {
	case "calico":
		return ClearCalico(c.Calico, nodes), nil
	}

	return nil, fmt.Errorf("no support cni type: %s", c.Type)
}

func ClearCalico(calico v1.Calico, nodes []v1.StepNode) []v1.Step {
	var steps []v1.Step

	switch calico.Mode {
	case CalicoNetworkIPIPAll, CalicoNetworkIPIPSubnet:
		steps = append(steps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "removeTunl",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"modprobe", "-r", "ipip"},
				},
			},
		})
	case CalicoNetworkVXLANAll, CalicoNetworkVXLANSubnet:
		steps = append(steps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "removeVtep",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			RetryTimes: 1,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"ip", "link", "delete", "vxlan.calico"},
				},
			},
		})
	}

	return steps
}

func RemoveHostname(kubeadm *v1.Kubeadm, nodes []v1.StepNode) ([]v1.Step, error) {
	var steps []v1.Step
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", kubeadm.Networking.DNSDomain)

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

func (c *KubectlTerminal) InitStepper(kubeadm *v1.Kubeadm) *KubectlTerminal {
	c.ImageRegistryAddr = kubeadm.LocalRegistry
	return c
}

func (c *KubectlTerminal) InstallSteps(stepMaster0 []v1.StepNode) ([]v1.Step, error) {
	installSteps := make([]v1.Step, 0)
	terminal, err := json.Marshal(c)
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

func (c *KubectlTerminal) UninstallSteps() ([]v1.Step, error) {
	return nil, nil
}
