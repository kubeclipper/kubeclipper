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
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/pkg/errors"
	"github.com/txn2/txeh"
	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/ipvsutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, packages, version, component.TypeStep), &Package{}); err != nil {
		panic(err)
	}
	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat, kubeadmConfig, version, component.TypeTemplate), &KubeadmConfig{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, kubeadmConfig, version, component.TypeStep), &KubeadmConfig{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, controlPlane, version, component.TypeStep), &ControlPlane{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, clusterNode, version, component.TypeStep), &ClusterNode{}); err != nil {
		panic(err)
	}
	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat, kubectlTerminal, version, component.TypeTemplate), &KubectlTerminal{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, health, version, component.TypeStep), &Health{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, container, version, component.TypeStep), &Container{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, kubectl, version, component.TypeStep), &Kubectl{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, kubeadmConfigUpdaterName, kubeadmConfigUpdaterVersion, component.TypeStep), &KubeadmConfigUpdater{}); err != nil {
		panic(err)
	}
}

var (
	_ component.StepRunnable   = (*Package)(nil)
	_ component.TemplateRender = (*KubeadmConfig)(nil)
	_ component.StepRunnable   = (*KubeadmConfig)(nil)
	_ component.StepRunnable   = (*ControlPlane)(nil)
	_ component.StepRunnable   = (*ClusterNode)(nil)
	_ component.TemplateRender = (*KubectlTerminal)(nil)
	_ component.StepRunnable   = (*Health)(nil)
	_ component.StepRunnable   = (*Container)(nil)
	_ component.StepRunnable   = (*Kubectl)(nil)
)

type Package struct {
	Offline       bool   `json:"offline"`
	Version       string `json:"version"`
	CriType       string `json:"criType"`
	LocalRegistry string `json:"localRegistry"`
	KubeletDir    string `json:"kubeletDir"`
}

type KubeadmConfig struct {
	ClusterConfigAPIVersion string `json:"clusterConfigAPIVersion"`
	// If both Docker and containerd are detected, Docker takes precedence,so we must specify cri.
	// https://v1-20.docs.kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-runtime
	ContainerRuntime     string          `json:"containerRuntime"`
	Etcd                 v1.Etcd         `json:"etcd"`
	Networking           v1.Networking   `json:"networking"`
	KubeProxy            v1.KubeProxy    `json:"kubeProxy"`
	Kubelet              v1.Kubelet      `json:"kubelet"`
	ClusterName          string          `json:"clusterName"`
	KubernetesVersion    string          `json:"kubernetesVersion"`
	ControlPlaneEndpoint string          `json:"controlPlaneEndpoint"`
	CertSANs             []string        `json:"certSANs"`
	LocalRegistry        string          `json:"localRegistry"`
	Offline              bool            `json:"offline"`
	IsControlPlane       bool            `json:"isControlPlane,omitempty"`
	CACertHashes         string          `json:"caCertHashes,omitempty"`
	BootstrapToken       string          `json:"bootstrapToken,omitempty"`
	CertificateKey       string          `json:"certificateKey,omitempty"`
	AdvertiseAddress     string          `json:"advertiseAddress,omitempty"`
	FeatureGates         map[string]bool `json:"featureGates,omitempty"`
}

type ControlPlane struct {
	// KubeConfig file
	APIServerDomainName string
	EtcdDataPath        string
	ContainerRuntime    string
	ExternalCaCert      string
	ExternalCaKey       string
}

type ClusterNode struct {
	// use type enum instead
	NodeRole      string
	WorkerNodeVIP string
	// master ip
	Masters             map[string]string // for IPVS rules
	LocalRegistry       string
	APIServerDomainName string
	JoinMasterIP        string
	EtcdDataPath        string
}

type Health struct {
	KubernetesVersion string
}

type Certification struct{}

type SAN struct{}

type Container struct {
	CriType string
}

type Kubectl struct{}

func (stepper *Package) NewInstance() component.ObjectMeta {
	return &Package{}
}

func (stepper *Package) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, K8s, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	// local registry not filled and is in offline mode, download images.tar.gz file from tarballs
	if stepper.Offline && stepper.LocalRegistry == "" {
		imageSrc, err := instance.DownloadImages()
		if err != nil {
			return nil, err
		}
		if err = utils.LoadImage(ctx, opts.DryRun, imageSrc, stepper.CriType); err != nil {
			return nil, err
		}
		logger.Info("image tarball decompress successfully")
	}
	// create kubelet config dir
	err = os.MkdirAll(Kubelet10KubeadmDir, 0755)
	if err != nil {
		return nil, err
	}
	// install configs.tar.gz
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	// enable kubelet
	if err = stepper.enableKubeletService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	logger.Debug("k8s packages offline install successfully")
	return nil, nil
}

func (stepper *Package) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	if err := stepper.disableKubeletService(ctx, opts.DryRun); err != nil {
		return nil, err
	}
	// remove related binary configuration files
	instance, err := downloader.NewInstance(ctx, K8s, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err = instance.RemoveAll(); err != nil {
		logger.Error("remove k8s configs and images compressed files failed", zap.Error(err))
	}

	if err = os.Remove(filepath.Join(KubeletDefaultDataDir, "config.yaml")); err != nil {
		logger.Errorf("remove config.yaml failed,err:%v", err.Error())
	}

	err = unmountKubeletDirectory(stepper.KubeletDir)
	if err != nil {
		logger.Warn("unmount kubelet dir failed", zap.Error(err))
		return nil, err
	}
	if _, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "rm", "-rf", stepper.KubeletDir); err != nil {
		logger.Warn("clean kubelet dir failed", zap.Error(err))
	}

	return nil, nil
}

func (stepper *Package) enableKubeletService(ctx context.Context, dryRun bool) error {
	// chmod 711
	files := []string{"kubelet-pre-start.sh", "kubelet", "kubeadm", "kubectl", "conntrack"}
	for _, f := range files {
		if err := os.Chmod(filepath.Join(KubeBinaryDir, f), 0711); err != nil {
			return err
		}
	}

	// enable systemd containerd service
	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "daemon-reload")
	if err != nil {
		return err
	}
	_, err = cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "enable", "kubelet", "--now")
	if err != nil {
		return err
	}
	logger.Debug("enable kubelet systemd service successfully")
	return nil
}

func (stepper *Package) disableKubeletService(ctx context.Context, dryRun bool) error {
	// The following command execution error is ignored
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "stop", "kubelet"); err != nil {
		logger.Warn("stop systemd kubelet service failed", zap.Error(err))
	}
	if _, err := cmdutil.RunCmdWithContext(ctx, dryRun, "systemctl", "disable", "kubelet"); err != nil {
		logger.Warn("disable systemd kubelet service failed", zap.Error(err))
	}
	return nil
}

func (stepper KubeadmConfig) NewInstance() component.ObjectMeta {
	return &KubeadmConfig{}
}

func (stepper KubeadmConfig) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	v := component.GetExtraData(ctx)
	if v == nil {
		return nil, fmt.Errorf("no join command received")
	}

	logger.Debug("get join command", zap.ByteString("cmd", v))
	cmdStr := strings.ReplaceAll(string(v), "\\n", "")
	cmds := strings.Split(cmdStr, ",")
	if len(cmds) != 2 {
		return nil, fmt.Errorf("join command invalid")
	}

	apiVersion, err := stepper.matchClusterConfigAPIVersion()
	if err != nil {
		return nil, err
	}
	stepper.ClusterConfigAPIVersion = apiVersion
	if stepper.Kubelet.RootDir == "" {
		stepper.Kubelet.RootDir = KubeletDefaultDataDir
	}
	agentIP, err := stepper.getAgentNodeIP()
	if err != nil {
		logger.Errorf("get node ip failed: %s", err.Error())
		return nil, err
	}
	// NodeIP is used for kubelet communication and api-server host ip
	stepper.Kubelet.NodeIP = agentIP
	// kubeadm join apiserver.cluster.local:6443 --token s9afr8.tsibqbmgddqku6xu --discovery-token-ca-cert-hash sha256:e34ec831f206237b38a1b29d46b5df599018c178959b874f6b14fe2438194f9d --control-plane --certificate-key 46518267766fc19772ecc334c13190f8131f1bf48a213538879f6427f74fe8e2
	// kubeadm join apiserver.cluster.local:6443 --token s9afr8.tsibqbmgddqku6xu --discovery-token-ca-cert-hash sha256:e34ec831f206237b38a1b29d46b5df599018c178959b874f6b14fe2438194f9d
	if stepper.IsControlPlane {
		masterJoinCmd := strings.Split(cmds[0], " ")
		if len(masterJoinCmd) < 10 {
			return nil, fmt.Errorf("master join command invalid")
		}
		stepper.BootstrapToken = masterJoinCmd[4]
		stepper.CACertHashes = masterJoinCmd[6]
		stepper.CertificateKey = masterJoinCmd[9]
		stepper.AdvertiseAddress = agentIP
	} else {
		workerJoinCmd := strings.Split(cmds[1], " ")
		if len(workerJoinCmd) < 6 {
			return nil, fmt.Errorf("worker join command invalid")
		}
		stepper.BootstrapToken = workerJoinCmd[4]
		stepper.CACertHashes = workerJoinCmd[6]
	}
	// check resolv-conf
	ok, err := isServiceActive("systemd-resolved")
	if err != nil {
		logger.Warnf("cannot determine if systemd-resolved is active: %v", err)
	}
	if ok {
		stepper.Kubelet.ResolvConf = KubeletSystemdResolverConfig
	} else {
		stepper.Kubelet.ResolvConf = KubeletDefaultResolvConf
	}

	if err := os.MkdirAll(ManifestDir, 0755); err != nil {
		return nil, err
	}
	manifestFile := filepath.Join(ManifestDir, "kubeadm-join.yaml")
	return v, fileutil.WriteFileWithContext(ctx, manifestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		stepper.renderJoin, opts.DryRun)
}

func (stepper KubeadmConfig) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (stepper KubeadmConfig) Render(ctx context.Context, opts component.Options) error {
	apiVersion, err := stepper.matchClusterConfigAPIVersion()
	if err != nil {
		return err
	}
	stepper.ClusterConfigAPIVersion = apiVersion
	stepper.Networking.Services.CIDRBlocks = []string{strings.Join(stepper.Networking.Services.CIDRBlocks, ",")}
	stepper.Networking.Pods.CIDRBlocks = []string{strings.Join(stepper.Networking.Pods.CIDRBlocks, ",")}

	if stepper.Kubelet.RootDir == "" {
		stepper.Kubelet.RootDir = KubeletDefaultDataDir
	}
	// local registry not filled and is in online mode, the default repo mirror proxy will be used
	if !stepper.Offline && stepper.LocalRegistry == "" {
		stepper.LocalRegistry = component.GetRepoMirror(ctx)
		logger.Info("render kubernetes config, the default repo mirror proxy will be used", zap.String("local_registry", stepper.LocalRegistry))
	}
	agentIP, err := stepper.getAgentNodeIP()
	if err != nil {
		logger.Errorf("get node ip failed: %s", err.Error())
		return err
	}
	stepper.Kubelet.NodeIP = agentIP

	if err := os.MkdirAll(ManifestDir, 0755); err != nil {
		return err
	}
	manifestFile := filepath.Join(ManifestDir, "kubeadm.yaml")
	return fileutil.WriteFileWithContext(ctx, manifestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		stepper.renderTo, opts.DryRun)
}

func (stepper *KubeadmConfig) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, kubeadmTemplate, stepper)
	return err
}

func (stepper *KubeadmConfig) renderJoin(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, KubeadmJoinTemplate, stepper)
	return err
}

// TODO: use kubeadm migrate
func (stepper *KubeadmConfig) matchClusterConfigAPIVersion() (string, error) {
	version := stepper.KubernetesVersion

	version = strings.ReplaceAll(version, "v", "")
	version = strings.ReplaceAll(version, ".", "")

	version = strings.Join(strings.Split(version, "")[0:3], "")
	ver, err := strconv.Atoi(version)
	if err != nil {
		return "", err
	}

	if ver < 118 {
		return "v1beta1", nil
	}

	if ver >= 118 && ver <= 121 {
		return "v1beta2", nil
	}
	// +1.22.x version
	return "v1beta3", nil
}

func (stepper *KubeadmConfig) getAgentNodeIP() (string, error) {
	agentConfig, err := config.TryLoadFromDisk()
	if err != nil {
		return "", errors.WithMessage(err, "load agent config")
	}
	ip, err := netutil.GetDefaultIP(true, agentConfig.NodeIPDetect)
	if err != nil {
		return "", err
	}
	if ip.String() == "" {
		return "", fmt.Errorf("agent node ip is empty, adjust the node-ip-detect configuration")
	}
	return ip.String(), nil
}

func (stepper *ControlPlane) NewInstance() component.ObjectMeta {
	return &ControlPlane{}
}

func (stepper *ControlPlane) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	// 1. add kubeadm config render
	// 2. systemctl enable kubelet --now
	// 3. kubeadm init --config xxx --upload-certs
	// 4. remove kubeconfig to $HOME/.kube
	// 5. return kubeadm join command

	clearCmd := fmt.Sprintf("kubeadm reset -f && rm -rf %s", strutil.StringDefaultIfEmpty(EtcdDefaultDataDir, stepper.EtcdDataPath))
	_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", clearCmd)
	if err != nil {
		logger.Warnf("clean init node env error: %s", err.Error())
	}

	err = stepper.externalCa(ctx, opts)
	if err != nil {
		return nil, err
	}

	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		return nil, err
	}
	agentConfig, err := config.TryLoadFromDisk()
	if err != nil {
		return nil, errors.WithMessage(err, "load agent config")
	}
	ipnet, err := netutil.GetDefaultIP(true, agentConfig.NodeIPDetect)
	if err != nil {
		return nil, err
	}
	// add apiserver domain name to /etc/hosts
	hosts.AddHost(ipnet.String(), stepper.APIServerDomainName)
	if err := hosts.Save(); err != nil {
		return nil, err
	}

	// before run 'kubeadm init', clean the processes 'kube-controller, kube-apiserver, kube-proxy, kube-scheduler, containerd-shim, etcd'
	pids, err := getProcessID(ctx, opts.DryRun)
	if err != nil {
		logger.Error("get process id error", zap.Error(err))
		return nil, err
	}

	if len(pids) > 0 {
		err = killProcess(ctx, opts.DryRun, pids)
		if err != nil {
			logger.Error("kill process error", zap.Error(err))
			return nil, err
		}
	}

	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "kubeadm", "init", "--config", "/tmp/.k8s/kubeadm.yaml", "--upload-certs")
	if err != nil {
		logger.Error("run kubeadm init error", zap.Error(err))
		return nil, err
	}

	joinControlPlaneCMD := "dry run join control plane"
	joinWorkerCMD := "dry run join worker"
	if !opts.DryRun {
		joinControlPlaneCMD = getJoinCmdFromStdOut(ec.StdOut(), "You can now join any number of the control-plane node running the following command on each as root:")
		joinWorkerCMD = getJoinCmdFromStdOut(ec.StdOut(), "Then you can join any number of worker nodes by running the following on each as root:")
		if err := generateKubeConfig(ctx); err != nil {
			return nil, err
		}
	}
	return []byte(fmt.Sprintf("%s,%s", joinControlPlaneCMD, joinWorkerCMD)), nil
}

func (stepper *ControlPlane) externalCa(ctx context.Context, opts component.Options) error {
	if stepper.ExternalCaCert != "" && stepper.ExternalCaKey != "" {
		certBytes, err := base64.StdEncoding.DecodeString(stepper.ExternalCaCert)
		if err != nil {
			return fmt.Errorf("external ca-cert file decode failed: %v", err)
		}
		keyBytes, err := base64.StdEncoding.DecodeString(stepper.ExternalCaKey)
		if err != nil {
			return fmt.Errorf("external ca-key file decode failed: %v", err)
		}

		caDir := fmt.Sprintf("%s/pki", K8SDefaultConfigDir)
		cmdList := []string{
			fmt.Sprintf("mkdir -p %s", caDir),
			fmt.Sprintf(`echo "%s" > %s/ca.crt`, string(certBytes), caDir),
			fmt.Sprintf(`echo "%s" > %s/ca.key`, string(keyBytes), caDir),
		}
		for _, cmd := range cmdList {
			_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getProcessID(ctx context.Context, dryRun bool) ([]string, error) {
	var (
		err error
		ec  *cmdutil.ExecCmd
	)
	pids := make([]string, 0)
	processes := []string{"kube-proxy", "kube-apiserver", "kube-controller", "kube-scheduler", "containerd-shim", "etcd"}

	for _, process := range processes {
		if strings.Contains(process, "etcd") {
			ec, err = cmdutil.RunCmdWithContext(ctx, dryRun, "/bin/bash", "-c", "ps -ef | grep "+process+" | grep -v grep | grep -v \"/usr\" | awk '{print $2}'")
			if err != nil {
				logger.Error("run ps -ef error", zap.Error(err))
				return pids, err
			}
		} else {
			ec, err = cmdutil.RunCmdWithContext(ctx, dryRun, "/bin/bash", "-c", "ps -ef | grep "+process+" | grep -v grep | awk '{print $2}'")
			if err != nil {
				logger.Error("run ps -ef error", zap.Error(err))
				return pids, err
			}
		}
		if ec.StdOut() != "" {
			p := strings.Split(ec.StdOut(), "\n")
			for i, v := range p {
				if v == "" {
					p = append(p[:i], p[i+1:]...)
				}
			}
			pids = append(pids, p...)
		}
	}

	return pids, nil
}

func killProcess(ctx context.Context, dryRun bool, pids []string) error {
	for _, pid := range pids {
		_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "kill", "-9", pid)
		if err != nil {
			logger.Error("kill the process error", zap.Error(err))
			return err
		}
	}
	return nil
}

func (stepper *ControlPlane) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	// just need deal with dynamic pv
	// 1.delete all pod
	// 2.delete all pv
	// 3.wait pv reclaim

	// get the namespace containing the pvc
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
		`kubectl get pvc -A  -o=custom-columns=NAMESPACE:.metadata.namespace`)
	if err != nil {
		logger.Warn("run 'kubectl get pvc' error", zap.Error(err))
	}
	// remove first line 'NAMESPACE'
	nsList := strings.Split(strings.Replace(ec.StdOut(), " ", "", -1), "\n")[1:]
	// removal of duplicates
	namespaces := make([]string, 0)
	tmpMap := make(map[string]interface{})
	for _, namespace := range nsList {
		// ignore kube-system namespace
		if namespace == "kube-system" || namespace == "" {
			continue
		}
		if _, ok := tmpMap[namespace]; !ok {
			namespaces = append(namespaces, namespace)
			tmpMap[namespace] = nil
		}
	}

	// delete pods and controllers
	_ = stepper.deletePodsAndCtls(ctx, opts, namespaces...)
	// delete pvcs
	_ = stepper.deletePVC(ctx, opts, namespaces...)
	_ = stepper.waitPVReclaim(ctx, opts)

	// clean the processes 'kube-controller, kube-apiserver, kube-proxy, kube-scheduler, containerd-shim, etcd'
	pids, err := getProcessID(ctx, opts.DryRun)
	if err != nil {
		logger.Error("get process id error", zap.Error(err))
		return nil, err
	}

	if len(pids) > 0 {
		err = killProcess(ctx, opts.DryRun, pids)
		if err != nil {
			logger.Error("kill process error", zap.Error(err))
			return nil, err
		}
	}
	return nil, nil
}

func (stepper *ControlPlane) deletePodsAndCtls(ctx context.Context, opts component.Options, namespaces ...string) error {
	for _, ns := range namespaces {
		// delete controller
		_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
			fmt.Sprintf(`kubectl delete --force=true --all=true deploy,sts,ds,rc,rs,cronjob,job,pod -n %s`, ns))
		if err == nil {
			logger.Infof("resources under the %s namespace are being deleted", ns)
		}
	}

	for _, ns := range namespaces {
		if err := utils.RetryFunc(ctx, opts, 5*time.Second, "get-resources", func(ctx context.Context, opts component.Options) error {
			ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
				fmt.Sprintf(`kubectl get deploy,sts,ds,rc,rs,cronjob,job,pod -n %s`, ns))
			if err != nil {
				logger.Warnf("kubectl get error: %s", err.Error())
			}
			if ec != nil && ec.StdOut() == "" {
				return nil
			}
			return nil
		}); err != nil {
			return err
		}
	}
	logger.Info("clear pvc pod and controller finish!")
	return nil
}

func (stepper *ControlPlane) deletePVC(ctx context.Context, opts component.Options, namespaces ...string) error {
	for _, ns := range namespaces {
		_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
			fmt.Sprintf(`kubectl delete --force=true --all=true pvc -n %s`, ns))
		if err == nil {
			logger.Infof("resources under the %s namespace are being deleted", ns)
		}
	}
	return nil
}

func (stepper *ControlPlane) waitPVReclaim(ctx context.Context, opts component.Options) error {
	// wait for provisioner reclaim all delete mode's pv
	if err := utils.RetryFunc(ctx, opts, 5*time.Second, "get-resources", func(ctx context.Context, opts component.Options) error {
		ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
			`kubectl get pv | awk {'print "pv",$1,"wait",$4'} | grep MODES -v | grep Delete`)
		if err != nil {
			logger.Warnf("kubectl get error: %s", err.Error())
		}
		if ec != nil {
			if ec.StdOut() == "" {
				return nil
			}
			return fmt.Errorf("wait for provisioner reclaim all delete mode's pv")
		}
		return nil
	}); err != nil {
		return err
	}
	logger.Info("all pv reclaimed!")
	return nil
}

func (stepper *ClusterNode) NewInstance() component.ObjectMeta {
	return &ClusterNode{}
}

func (stepper *ClusterNode) setRole(role string) {
	stepper.NodeRole = role
}

func (stepper *ClusterNode) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	// before join node, clean the processes 'kube-controller, kube-apiserver, kube-proxy, kube-scheduler, containerd-shim, etcd'
	pids, err := getProcessID(ctx, opts.DryRun)
	if err != nil {
		logger.Error("get process id error", zap.Error(err))
		return nil, err
	}

	if len(pids) > 0 {
		err = killProcess(ctx, opts.DryRun, pids)
		if err != nil {
			logger.Error("kill process error", zap.Error(err))
			return nil, err
		}
	}
	joincmd := []string{"kubeadm", "join", "--config", filepath.Join(ManifestDir, "kubeadm-join.yaml")}
	_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "modprobe br_netfilter && modprobe nf_conntrack")
	if err != nil {
		logger.Warnf("modprobe command error: %s", err.Error())
	}

	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		return nil, err
	}
	if stepper.NodeRole == NodeRoleMaster {
		clearCmd := fmt.Sprintf("kubeadm reset -f && rm -rf %s && rm -rf %s",
			strutil.StringDefaultIfEmpty(EtcdDefaultDataDir, stepper.EtcdDataPath), K8SDefaultConfigDir)
		_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", clearCmd)
		if err != nil {
			logger.Warnf("clean init node env error: %s", err.Error())
		}
		// add apiserver domain name to /etc/hosts
		hosts.AddHost(stepper.JoinMasterIP, stepper.APIServerDomainName)
		if err := hosts.Save(); err != nil {
			return nil, err
		}
		_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, joincmd[0], joincmd[1:]...)
		if err != nil {
			return nil, err
		}
		agentConfig, err := config.TryLoadFromDisk()
		if err != nil {
			return nil, errors.WithMessage(err, "load agent config")
		}
		ipnet, err := netutil.GetDefaultIP(true, agentConfig.NodeIPDetect)
		if err != nil {
			return nil, err
		}
		// add apiserver domain name to /etc/hosts
		hosts.AddHost(ipnet.String(), stepper.APIServerDomainName)
		if err := hosts.Save(); err != nil {
			return nil, err
		}

		// cp admin.conf
		_, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun,
			"bash", "-c", "mkdir -p $HOME/.kube && cp -if /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config")
		if err != nil {
			return nil, err
		}
	}
	if stepper.NodeRole == NodeRoleWorker {
		clearCmd := fmt.Sprintf("kubeadm reset -f && rm -rf %s", K8SDefaultConfigDir)
		_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", clearCmd)
		if err != nil {
			logger.Warnf("clean init node env error: %s", err.Error())
		}
		hosts.AddHost(stepper.WorkerNodeVIP, stepper.APIServerDomainName)
		if len(stepper.Masters) == 1 {
			hosts.AddHost(stepper.JoinMasterIP, stepper.APIServerDomainName)
		}

		if err := hosts.Save(); err != nil {
			return nil, err
		}

		if len(stepper.Masters) > 1 && stepper.WorkerNodeVIP != "" {
			var rsList []ipvsutil.RealServer
			for _, ip := range stepper.Masters {
				dest := ipvsutil.RealServer{
					Address: ip,
					Port:    6443,
				}
				rsList = append(rsList, dest)
			}

			// TODO: refactor ipvs utils
			vs := ipvsutil.VirtualServer{
				Address:     stepper.WorkerNodeVIP,
				Port:        6443,
				RealServers: rsList,
			}
			err = ipvsutil.Clear(opts.DryRun)
			if err != nil {
				logger.Warnf("ipvs clear service error info: %v", err)
			}
			err = ipvsutil.CreateIPVS(&vs, opts.DryRun)
			if err != nil {
				return nil, err
			}
		}
		if _, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, joincmd[0], joincmd[1:]...); err != nil {
			return nil, err
		}

		if len(stepper.Masters) > 1 && stepper.WorkerNodeVIP != "" && !opts.DryRun {
			err = stepper.generatesIPSOCareStaticPod(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (stepper *ClusterNode) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, fmt.Errorf("ClusterNode dose not support uninstall")
}

func (stepper *ClusterNode) generatesIPSOCareStaticPod(ctx context.Context) error {
	if err := os.MkdirAll("/etc/kubernetes/manifests", 0755); err != nil {
		return err
	}
	manifestFile := filepath.Join("/etc/kubernetes/manifests", "kube-lvscare.yaml")
	return fileutil.WriteFileWithContext(ctx, manifestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		stepper.renderIPVSCarePod, false)
}

func (stepper *ClusterNode) renderIPVSCarePod(w io.Writer) error {
	_, err := tmplutil.New().RenderTo(w, lvscareV111, stepper)
	return err
}

func (stepper *Health) NewInstance() component.ObjectMeta {
	return &Health{}
}

func (stepper *Health) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	if err := utils.RetryFunc(ctx, opts, 10*time.Second, "allNodeReady", stepper.allNodeReady); err != nil {
		return nil, err
	}
	if err := utils.RetryFunc(ctx, opts, 10*time.Second, "checkPodStatus", stepper.checkPodStatus); err != nil {
		return nil, err
	}

	return nil, nil
}

func (stepper *Health) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	// clear ipvs
	if err := ipvsutil.Clear(opts.DryRun); err == nil {
		logger.Info("clear ipvs successfully")
	}
	return nil, nil
}

// the status of all nodes in the cluster is ready
func (stepper *Health) allNodeReady(ctx context.Context, opts component.Options) error {
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", `kubectl get node | grep NotReady`)
	if err != nil {
		logger.Warn("run kubectl get node error", zap.Error(err))
	}
	if ec.StdOut() == "" {
		logger.Info("all nodes are ready")
		return nil
	}
	logger.Warn("some nodes are not ready", zap.String("node", ec.StdOut()))
	return fmt.Errorf("some nodes are not ready")
}

// k8s own component(kube-systemd) pod status is running
func (stepper *Health) checkPodStatus(ctx context.Context, opts component.Options) error {
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", `kubectl get po -n kube-system | grep -v Running`)
	if err != nil {
		logger.Warn("run 'kubectl get po -n kube-system | grep -v Running' error", zap.Error(err))
	}

	if len(strings.Split(ec.StdOut(), "\n")) > 2 {
		return fmt.Errorf("there are no running pods: %s", strings.Join(ec.Args[1:], ","))
	}
	return err
}

func (stepper *Health) getRegisterServiceAccountCommands() ([]v1.Command, error) {
	commands := []v1.Command{
		{
			Type:         v1.CommandShell,
			ShellCommand: []string{"kubectl", "create", "sa", "kc-server", "-n", "kube-system"},
		},
		{
			Type:         v1.CommandShell,
			ShellCommand: []string{"kubectl", "create", "clusterrolebinding", "kc-server", "--clusterrole=cluster-admin", "--serviceaccount=kube-system:kc-server"},
		},
	}
	ver, err := strutil.StealKubernetesMajorVersionNumber(stepper.KubernetesVersion)
	if err != nil {
		return nil, err
	}
	// Kubernetes does not automatically create a secret for `ServiceAccount` after v1.24. You need to do so yourself.
	if ver >= 124 {
		secretJSON := `{"apiVersion":"v1","kind":"Secret","metadata":{"annotations":{"kubernetes.io/service-account.name":"kc-server"},"name":"kc-server-secret","namespace":"kube-system"},"type":"kubernetes.io/service-account-token"}`
		filePath := filepath.Join(ManifestDir, "kc-server-secret.json")
		createSecretCmd := fmt.Sprintf("echo '%s' > %s | kubectl create -f %s -n kube-system", secretJSON, filePath, filePath)
		// 1. create service-account-token secret
		// 2. patch service-account secrets
		patchCommand := []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"bash", "-c", createSecretCmd},
			},
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"bash", "-c", `kubectl patch sa kc-server -n kube-system -p '{"secrets":[{"name":"kc-server-secret"}]}'`},
			},
		}
		commands = append(commands[:1], append(patchCommand, commands[1])...)
	}
	return commands, nil
}

func (stepper *Certification) NewInstance() component.ObjectMeta {
	return &Certification{}
}

func (stepper *Certification) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (stepper *Certification) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (stepper *Container) NewInstance() component.ObjectMeta {
	return &Container{}
}

func (stepper *Container) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, fmt.Errorf("Container dose not support install")
}

func (stepper *Container) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	var err error
	switch stepper.CriType {
	case "containerd":
		err = deleteContainer("k8s.io")
		if err != nil {
			logger.Warnf("delete containerd container error: %s", err.Error())
		}
	case "docker":
	// TODO
	default:
		logger.Errorf("current cri type is '%s', '%s' is not supported clean", stepper.CriType, stepper.CriType)
	}
	return nil, err
}

func (stepper *Kubectl) NewInstance() component.ObjectMeta {
	return &Kubectl{}
}

func (stepper *Kubectl) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, fmt.Errorf("Kubectl dose not support uninstall")
}

func (stepper *Kubectl) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	// remove kubeconfig
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(filepath.Join(home, KubeConfigDir)); err != nil {
		return nil, err
	}
	logger.Debug("remove kubeconfig successfully")
	return nil, nil
}

type KubectlTerminal struct {
	ImageRegistryAddr string
}

func (stepper *KubectlTerminal) NewInstance() component.ObjectMeta {
	return &KubectlTerminal{}
}

func (stepper *KubectlTerminal) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, KubectlPodTemplate, stepper)
	return err
}

func (stepper *KubectlTerminal) Render(ctx context.Context, opts component.Options) error {
	if err := os.MkdirAll(ManifestDir, 0755); err != nil {
		return err
	}
	manifestFile := filepath.Join(ManifestDir, "kc-kubectl.yaml")
	return fileutil.WriteFileWithContext(ctx, manifestFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		stepper.renderTo, opts.DryRun)
}

func (stepper *SAN) NewInstance() component.ObjectMeta {
	return &SAN{}
}

func (stepper *SAN) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}

func (stepper *SAN) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, nil
}
