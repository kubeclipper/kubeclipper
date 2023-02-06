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
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cni"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ component.StepRunnable = (*JoinCmd)(nil)
	_ component.StepRunnable = (*Drain)(nil)
)

const (
	joinNodeCmd = "joinNodeCmd"
	drain       = "drain"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, joinNodeCmd, version, component.TypeStep), &JoinCmd{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, drain, version, component.TypeStep), &Drain{}); err != nil {
		panic(err)
	}
}

type GenNode struct {
	Nodes          component.NodeList `json:"nodes"`
	Cluster        *v1.Cluster
	installSteps   []v1.Step
	uninstallSteps []v1.Step
}

type JoinCmd struct {
	ContainerRuntime string `json:"containerRuntime"`
}

type KubeadmJoinUtil struct {
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint"`
	Token                string `json:"token"`
	DiscoveryHash        string `json:"discoveryHash"`
	ContainerRuntime     string `json:"containerRuntime"`
}

type Drain struct {
	Hostname  string   `json:"hostname"`
	ExtraArgs []string `json:"extraArgs"`
}

func (stepper *GenNode) Validate() error {
	if stepper == nil {
		return fmt.Errorf("GenNode object is empty")
	}

	if stepper.Nodes == nil {
		return fmt.Errorf("GenNode Nodes object is empty")
	}

	return nil
}

func (stepper *GenNode) InitStepper(metadata *component.ExtraMetadata, cluster *v1.Cluster, role string) *GenNode {
	switch role {
	case NodeRoleMaster:
		stepper.Nodes = append(stepper.Nodes, metadata.Masters...)
	case NodeRoleWorker:
		stepper.Nodes = append(stepper.Nodes, metadata.Workers...)
	}
	stepper.Cluster = cluster
	return stepper
}

func (stepper *GenNode) MakeInstallSteps(metadata *component.ExtraMetadata, patchNodes []v1.StepNode, role string) error {
	err := stepper.Validate()
	if err != nil {
		return err
	}
	avaMasters, err := metadata.Masters.AvailableKubeMasters()
	if err != nil {
		return err
	}
	masters := utils.UnwrapNodeList(avaMasters)

	// add node to cluster
	if len(stepper.installSteps) == 0 {
		// We should use kubeadm to create join token on the first control plane node.
		steps, err := EnvSetupSteps(patchNodes)
		if err != nil {
			return err
		}
		stepper.installSteps = append(stepper.installSteps, steps...)

		if metadata.Offline {
			cf, err := cni.Load(stepper.Cluster.CNI.Type)
			if err != nil {
				return err
			}
			cniStepper := cf.Create().InitStep(metadata, &stepper.Cluster.CNI, &stepper.Cluster.Networking)
			steps, err = cniStepper.LoadImage(patchNodes)
			if err != nil {
				return err
			}
			stepper.installSteps = append(stepper.installSteps, steps...)

			for _, addon := range metadata.Addons {
				ad, ok := component.Load(fmt.Sprintf(component.RegisterFormat, addon.Name, addon.Version))
				if !ok {
					continue
				}
				compMeta := ad.NewInstance()
				if err := json.Unmarshal(addon.Config.Raw, compMeta); err != nil {
					return err
				}
				newComp, _ := compMeta.(component.Interface)
				if err := newComp.InitSteps(component.WithExtraMetadata(context.TODO(), *metadata)); err != nil {
					return err
				}
				stepList := newComp.GetInstallSteps()
				for _, st := range stepList {
					// TODO: Temporarily use hardcode to adjust, and subsequently optimize the step code for image download and load
					if strings.Contains(st.Name, "imageLoad") {
						st.Nodes = patchNodes
						stepper.installSteps = append(stepper.installSteps, st)
						break
					}
				}
			}
		}

		joinCmd := JoinCmd{}
		steps, err = joinCmd.InitStepper(stepper.Cluster.ContainerRuntime.Type).InstallSteps([]v1.StepNode{masters[0]})
		if err != nil {
			return err
		}
		stepper.installSteps = append(stepper.installSteps, steps...)

		// Notice: only support join worker node now.
		kubeadmConf := KubeadmConfig{}
		steps, err = kubeadmConf.InitStepper(stepper.Cluster, metadata).JoinSteps(false, patchNodes)
		if err != nil {
			return err
		}
		stepper.installSteps = append(stepper.installSteps, steps...)

		join := ClusterNode{}
		steps, err = join.InitStepper(stepper.Cluster, metadata).InstallSteps(role, patchNodes)
		if err != nil {
			return err
		}
		stepper.installSteps = append(stepper.installSteps, steps...)
	}

	return nil
}

// MakeUninstallSteps make uninstall-steps
func (stepper *GenNode) MakeUninstallSteps(metadata *component.ExtraMetadata, patchNodes []v1.StepNode) error {
	err := stepper.Validate()
	if err != nil {
		return err
	}
	avaMasters, err := metadata.Masters.AvailableKubeMasters()
	if err != nil {
		return err
	}
	masters := utils.UnwrapNodeList(avaMasters)
	if len(stepper.uninstallSteps) == 0 {
		args := []string{"--ignore-daemonsets", "--delete-local-data"}
		for _, node := range patchNodes {
			d := &Drain{}
			steps, err := d.InitStepper(node.Hostname, args).UninstallSteps([]v1.StepNode{masters[0]})
			if err != nil {
				return err
			}
			stepper.uninstallSteps = append(stepper.uninstallSteps, steps...)
		}

		steps, err := KubeadmReset(patchNodes)
		if err != nil {
			return err
		}
		stepper.uninstallSteps = append(stepper.uninstallSteps, steps...)
		// worker nodes don't need to remove etcd data dir
		stepper.uninstallSteps = append(stepper.uninstallSteps,
			doCommandRemoveStep("removeKubeletDataDir", patchNodes, KubeletDefaultDataDir),
			doCommandRemoveStep("removeDockershimDataDir", patchNodes, DockershimDefaultDataDir),
		)
		heal := Health{}
		steps, err = heal.InitStepper(metadata.KubeVersion).UninstallSteps(&stepper.Cluster.Networking, patchNodes...)
		if err != nil {
			return err
		}
		stepper.uninstallSteps = append(stepper.uninstallSteps, steps...)

		// clean CNI config
		steps, err = CleanCNI(metadata, &stepper.Cluster.CNI, &stepper.Cluster.Networking, patchNodes)
		if err != nil {
			return err
		}

		stepper.uninstallSteps = append(stepper.uninstallSteps, steps...)
		// clean Kubernetes config
		stepper.uninstallSteps = append(stepper.uninstallSteps,
			doCommandRemoveStep("removeKubernetesConfig", patchNodes, K8SDefaultConfigDir))
		// clear worker /etc/hosts vip domain
		// sed -i '/apiserver.cluster.local/d' /etc/hosts
		apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local",
			stepper.Cluster.Networking.DNSDomain)
		stepper.uninstallSteps = append(stepper.uninstallSteps, v1.Step{
			ID:         strutil.GetUUID(),
			Name:       "clearVIPDomain",
			Timeout:    metav1.Duration{Duration: 5 * time.Second},
			ErrIgnore:  true,
			RetryTimes: 1,
			Nodes:      patchNodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"bash", "-c", fmt.Sprintf("sed -i '/%s/d' /etc/hosts", apiServerDomain)},
				},
			},
		})

	}

	return nil
}

func (stepper *GenNode) GetSteps(action v1.StepAction) []v1.Step {
	switch action {
	case v1.ActionInstall:
		return stepper.installSteps
	case v1.ActionUninstall:
		return stepper.uninstallSteps
	}
	return nil
}

func (stepper *JoinCmd) InitStepper(criType string) *JoinCmd {
	stepper.ContainerRuntime = criType
	return stepper
}

func (stepper *JoinCmd) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}

	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "getJoinCommand",
			Timeout:    metav1.Duration{Duration: 10 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, joinNodeCmd, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *JoinCmd) UninstallSteps() ([]v1.Step, error) {
	return nil, fmt.Errorf("JoinCmd dose not support uninstall steps")
}

func (stepper *Drain) InitStepper(hostname string, extraArgs []string) *Drain {
	stepper.Hostname = hostname
	stepper.ExtraArgs = extraArgs
	return stepper
}

func (stepper *Drain) InstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	return nil, fmt.Errorf("Drain dose not support install steps")
}

func (stepper *Drain) UninstallSteps(nodes []v1.StepNode) ([]v1.Step, error) {
	bytes, err := json.Marshal(stepper)
	if err != nil {
		return nil, err
	}
	return []v1.Step{
		{
			ID:         strutil.GetUUID(),
			Name:       "drainNode",
			Timeout:    metav1.Duration{Duration: 30 * time.Minute},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      nodes,
			Action:     v1.ActionUninstall,
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, drain, version, component.TypeStep),
					CustomCommand: bytes,
				},
			},
		},
	}, nil
}

func (stepper *KubeadmJoinUtil) InitStepper(line, cri string) *KubeadmJoinUtil {
	line = strings.TrimLeft(line, " ")
	line = strings.TrimRight(line, " ")
	for _, str := range []string{"  ", "   ", "    ", "     "} {
		line = strings.ReplaceAll(line, str, " ")
	}
	strs := strings.Split(line, " ")
	if len(strs) < 6 {
		stepper = &KubeadmJoinUtil{}
	}
	stepper.ContainerRuntime = cri
	stepper.ControlPlaneEndpoint = strs[2]
	stepper.Token = strs[4]
	stepper.DiscoveryHash = strs[6]
	return stepper
}

func (stepper JoinCmd) NewInstance() component.ObjectMeta {
	return &JoinCmd{}
}

func (stepper JoinCmd) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	if opts.DryRun {
		return nil, nil
	}

	// kubeadm token create --print-join-command
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubeadm", "token", "create", "--print-join-command")
	if err != nil {
		logger.Error("run kubeadm token create error", zap.Error(err))
		return nil, err
	}
	cmd := KubeadmJoinUtil{}
	cmd.InitStepper(ec.StdOut(), stepper.ContainerRuntime)
	// bytes, err = json.Marshal(cmd)
	// format: ${master node join command};${worker node join command}
	// Work around to split out the worker node join command.
	return []byte("," + strings.Join(cmd.GetCmd(), " ")), nil
}

func (stepper JoinCmd) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, fmt.Errorf("JoinNodeCmd dose not support Uninstall")
}

func (stepper *Drain) NewInstance() component.ObjectMeta {
	return &Drain{}
}

func (stepper *Drain) Install(ctx context.Context, opts component.Options) (bytes []byte, err error) {
	return
}

func (stepper *Drain) Uninstall(ctx context.Context, opts component.Options) (bytes []byte, err error) {
	var ec *cmdutil.ExecCmd
	var logErrMsg string
	errMsg := fmt.Sprintf("nodes \"%s\" not found", stepper.Hostname)
	defer func() {
		if err != nil {
			if strings.Contains(ec.StdErr(), errMsg) {
				err = nil
				return
			}
			err = fmt.Errorf(ec.StdErr())
			logger.Error(logErrMsg, zap.Error(err))
		}
	}()

	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "get", "node", stepper.Hostname)
	if err != nil {
		logErrMsg = "kubectl get nodes error"
		return
	}

	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "taint", "nodes", stepper.Hostname, "NoExec=true:NoExecute")
	if err != nil {
		logErrMsg = "kubectl taint node error"
		return
	}

	cmds := strings.Split(fmt.Sprintf("kubectl drain %s", stepper.Hostname), " ")
	cmds = append(cmds, stepper.ExtraArgs...)
	// kubectl drain ${node_name} --ignore-daemonsets --delete-local-data (v1.20.13)
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, cmds[0], cmds[1:]...)
	if err != nil {
		logErrMsg = "kubectl drain node error"
		return
	}

	// kubectl delete node ${node_name}
	_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubectl", "delete", "node", stepper.Hostname)
	if err != nil {
		// logger.Error("kubectl delete node error", zap.Error(err))
		logErrMsg = "kubectl delete node error"
		return
	}

	return
}

func (stepper *KubeadmJoinUtil) GetCmd() []string {
	cmd := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash %s",
		stepper.ControlPlaneEndpoint, stepper.Token, stepper.DiscoveryHash)
	if stepper.ContainerRuntime == "containerd" {
		cmd += " --cri-socket /run/containerd/containerd.sock"
	}
	return strings.Split(cmd, " ")
}
