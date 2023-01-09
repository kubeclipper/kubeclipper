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

package cluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/test/framework/cluster"

	"github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cni"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

const (
	kubeVersion = "v1.23.6"
	criType     = corev1.CRIContainerd
	criVersion  = "1.6.4"
	cniType     = "calico"
	cniVersion  = "v3.22.4"
	//arch        = runtime.GOARCH
	arch = "amd64"
)

type Config struct {
	KubeConfig  string `json:"kubeConfig"`
	ClusterName string `json:"clusterName"`
}

type e2eCluster struct {
	// TODO: Static File Server TLS is not supported at this time
	StaticURL string
	SSH       *sshutils.SSH
	NodeIP    string
	// Only for special handling of nodes where kc-agent already exists
	RemoveNodeID string
}

func initCloudProvider(ec *e2eCluster, clusterName string) (*corev1.CloudProvider, error) {
	kubeConf, err := getKubeConfig(ec)
	if err != nil {
		return nil, err
	}
	c := &Config{
		KubeConfig:  base64.StdEncoding.EncodeToString([]byte(kubeConf)),
		ClusterName: clusterName,
	}
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	cp := &corev1.CloudProvider{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CloudProvider",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "e2e-kubeadm-provider",
			Annotations: map[string]string{common.AnnotationDescription: "e2e kubeadm provider"},
		},
		Type:   "kubeadm",
		Region: "default",
		Config: runtime.RawExtension{Raw: data},
		SSH: corev1.SSH{
			User:               ec.SSH.User,
			Password:           ec.SSH.Password,
			PrivateKey:         ec.SSH.PrivateKey,
			PrivateKeyPassword: ec.SSH.PkPassword,
			Port:               ec.SSH.Port,
		},
	}

	return cp, nil
}

func getKubeConfig(ec *e2eCluster) (string, error) {
	cmd := "cat $HOME/.kube/config"
	ret, err := sshutils.SSHCmdWithSudo(ec.SSH, ec.NodeIP, cmd)
	if err != nil {
		return "", errors.WithMessagef(err, "run cmd [%s] on node [%s]", cmd, ec.NodeIP)
	}
	if err = ret.Error(); err != nil {
		return "", errors.WithMessage(err, ret.String())
	}
	return ret.Stdout, nil
}

func initE2ECluster(ctx context.Context, f *framework.Framework) (*e2eCluster, error) {
	configList, err := f.KcClient().DescribeConfigMap(ctx, "deploy-config")
	if err != nil {
		return nil, err
	}

	data := configList.Items[0].Data[constatns.DeployConfigConfigMapKey]
	dc := new(DeployConfig)
	err = yaml.Unmarshal([]byte(data), dc)
	if err != nil {
		return nil, err
	}
	ec := new(e2eCluster)
	ec.SSH = dc.SSHConfig
	ec.SSH.PkFile = ""
	ec.StaticURL = fmt.Sprintf("http://%s:%v", dc.ServerIPs[0], dc.StaticServerPort)

	nodes, err := f.KcClient().ListNodes(context.TODO(), kc.Queries{
		Pagination:    query.NoPagination(),
		LabelSelector: fmt.Sprintf("!%s", common.LabelNodeRole),
	})
	if err != nil {
		return nil, err
	}
	if nodes.TotalCount == 0 {
		return nil, fmt.Errorf("not enough nodes to test kubeadm-cluster-import")
	}

	ec.NodeIP = nodes.Items[0].Status.Ipv4DefaultIP

	return ec, nil
}

func cmd(ssh *sshutils.SSH, nodeIP string, cmdList ...string) error {
	for _, c := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(ssh, nodeIP, c)
		if err != nil {
			return errors.WithMessagef(err, "run cmd [%s] on node [%s]", c, nodeIP)
		}
		if err = ret.Error(); err != nil {
			return errors.WithMessage(err, ret.String())
		}
	}

	return nil
}

func installAIOCluster(ec *e2eCluster) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	cmdList, err := initEnv()
	if err != nil {
		return err
	}
	err = cmd(ec.SSH, ec.NodeIP, cmdList...)
	if err != nil {
		return fmt.Errorf("init node env failed: %v", err)
	}
	ginkgo.By("init cluster env [ok]")

	cmdList, err = installCRI(ec.StaticURL)
	if err != nil {
		return err
	}
	err = cmd(ec.SSH, ec.NodeIP, cmdList...)
	if err != nil {
		return fmt.Errorf("install cri failed: %v", err)
	}
	ginkgo.By("install cri [ok]")

	cmdList, err = initControlPlane(ec.StaticURL, ec.NodeIP)
	if err != nil {
		return err
	}
	err = cmd(ec.SSH, ec.NodeIP, cmdList...)
	if err != nil {
		return fmt.Errorf("init controlPlane failed: %v", err)
	}
	ginkgo.By("init cluster [ok]")

	err = installCNI(ec)
	if err != nil {
		return fmt.Errorf("init cni failed: %v", err)
	}
	ginkgo.By("install cni [ok]")

	return retryFunc(ctx, time.Second*3, ec, checkReady)
}

func uninstallAIOCluster(ec *e2eCluster) error {
	cmdList, err := uninstallCluster()
	if err != nil {
		return err
	}
	err = cmd(ec.SSH, ec.NodeIP, cmdList...)
	if err != nil {
		return fmt.Errorf("uninstall cluster failed: %v", err)
	}

	return nil
}

type DeployConfig struct {
	// TODO: Static File Server TLS is not supported at this time
	ServerIPs        []string      `json:"serverIPs" yaml:"serverIPs,omitempty"`
	StaticServerPort int           `json:"staticServerPort" yaml:"staticServerPort,omitempty"`
	SSHConfig        *sshutils.SSH `json:"ssh" yaml:"ssh,omitempty"`
}

func initEnv() ([]string, error) {
	return []string{k8s.GetNodeScript()}, nil
}

func installCRI(staticURL string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/%s/%s", staticURL, criType, criVersion, arch)
	at := tmplutil.New()
	tmpl, err := at.Render(cri.ConfigTomlTemplate, cri.ContainerdRunnable{PauseVersion: cri.MatchPauseVersion(kubeVersion)})
	if err != nil {
		return nil, err
	}

	cmdList := make([]string, 0)
	cmdList = append(cmdList, fmt.Sprintf("mkdir -p /tmp/.%s", criType))
	cmdList = append(cmdList, fmt.Sprintf("curl %s/configs.tar.gz -o /tmp/.%s/configs.tar.gz", url, criType))
	cmdList = append(cmdList, fmt.Sprintf("tar -zxvf /tmp/.%s/configs.tar.gz -C /", criType))
	cmdList = append(cmdList, fmt.Sprintf("mkdir -p %s", cri.ContainerdDefaultConfigDir))
	cmdList = append(cmdList, fmt.Sprintf(`echo '%s'  >  %s/config.toml`, tmpl, cri.ContainerdDefaultConfigDir))
	cmdList = append(cmdList, "systemctl daemon-reload", "systemctl enable containerd", "systemctl restart containerd")

	return cmdList, err
}

func initControlPlane(staticURL, nodeIP string) ([]string, error) {
	kubeName := "k8s"
	url := fmt.Sprintf("%s/%s/%s/%s", staticURL, kubeName, kubeVersion, arch)

	cmdList := make([]string, 0)
	cmdList = append(cmdList, fmt.Sprintf("mkdir -p /tmp/.%s", kubeName))
	cmdList = append(cmdList, fmt.Sprintf("curl %s/configs.tar.gz -o /tmp/.%s/configs.tar.gz", url, kubeName))
	cmdList = append(cmdList, fmt.Sprintf("curl %s/images.tar.gz -o /tmp/.%s/images.tar.gz", url, kubeName))
	cmdList = append(cmdList, fmt.Sprintf("tar -zxvf /tmp/.%s/configs.tar.gz -C /", kubeName))
	cmdList = append(cmdList, fmt.Sprintf("gzip -df /tmp/.%s/images.tar.gz", kubeName))
	cmdList = append(cmdList, fmt.Sprintf("chmod +x %s/kubelet-pre-start.sh %s/conntrack", k8s.KubeBinaryDir, k8s.KubeBinaryDir))
	cmdList = append(cmdList, fmt.Sprintf("ctr --namespace k8s.io image import /tmp/.%s/images.tar", kubeName))
	cmdList = append(cmdList, fmt.Sprintf("kubeadm init --kubernetes-version=%s --pod-network-cidr=%s --service-cidr=%s --apiserver-advertise-address=%s", strings.Replace(kubeVersion, "v", "", -1), constatns.ClusterPodSubnet, constatns.ClusterServiceSubnet, nodeIP))
	cmdList = append(cmdList, "rm -rf $HOME/.kube", "mkdir -p $HOME/.kube", "sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config", "sudo chown $(id -u):$(id -g) $HOME/.kube/config")

	return cmdList, nil
}

func installCNI(ec *e2eCluster) error {
	dir := fmt.Sprintf("/tmp/.%s", cniType)
	url := fmt.Sprintf("%s/%s/%s/%s", ec.StaticURL, cniType, cniVersion, arch)
	cmdList := make([]string, 0)
	cmdList = append(cmdList, fmt.Sprintf("mkdir -p %s", dir))
	cmdList = append(cmdList, fmt.Sprintf("curl %s/images.tar.gz -o %s/images.tar.gz", url, dir))
	cmdList = append(cmdList, fmt.Sprintf("gzip -df %s/images.tar.gz", dir))
	cmdList = append(cmdList, fmt.Sprintf("ctr --namespace k8s.io image import %s/images.tar", dir))

	err := cmd(ec.SSH, ec.NodeIP, cmdList...)
	if err != nil {
		return err
	}

	at := tmplutil.New()
	tmpl, err := at.Render(cni.CalicoV3224,
		cni.CalicoRunnable{
			BaseCni: cni.BaseCni{
				PodIPv4CIDR: constatns.ClusterPodSubnet,
				CNI: corev1.CNI{
					Type:    cniType,
					Version: cniVersion,
					Calico: &corev1.Calico{
						IPv4AutoDetection: "first-found",
						IPv6AutoDetection: "first-found",
						Mode:              "Overlay-Vxlan-All",
						IPManger:          true,
						MTU:               1440,
					},
				},
			}})
	if err != nil {
		return err
	}

	err = os.WriteFile("calico.yaml", []byte(tmpl), os.ModePerm)
	if err != nil {
		return err
	}
	defer os.Remove("calico.yaml")

	err = ec.SSH.Copy(ec.NodeIP, "calico.yaml", fmt.Sprintf("%s/calico.yaml", dir))
	if err != nil {
		return err
	}

	c := fmt.Sprintf(`kubectl apply -f %s/calico.yaml`, dir)
	return cmd(ec.SSH, ec.NodeIP, c)
}

func checkReady(ec *e2eCluster) error {
	ret, err := sshutils.SSHCmdWithSudo(ec.SSH, ec.NodeIP, "kubectl get node | grep NotReady")
	if err != nil {
		return fmt.Errorf("check node status failed: %v", err)
	}
	if ret.StdoutToString("") == "" {
		return nil
	}
	return fmt.Errorf("node status not-ready")
}

func uninstallCluster() ([]string, error) {
	cmdList := make([]string, 0)
	cmdList = append(cmdList, "kubeadm reset -f", "systemctl disable kubelet --now")
	cmdList = append(cmdList, "rm -rf /var/lib/etcd", "rm -rf /var/lib/kubelet", "rm -rf /etc/cni", "rm -rf /etc/kubernetes", "rm -rf $HOME/.kube")
	cmdList = append(cmdList, "systemctl disable containerd --now", "rm -rf /run/containerd", "rm -rf /etc/containerd", "rm -rf /var/lib/containerd")
	cmdList = append(cmdList, "rm -rf /usr/local/bin/kubelet", "rm -rf /usr/local/bin/containerd", "rm -rf /usr/local/bin/conntrack", "rm -rf /usr/local/bin/kubelet-pre-start.sh")
	cmdList = append(cmdList, "rm -rf /etc/systemd/system/containerd.service", "rm -rf /etc/systemd/system/kubelet.service")
	return cmdList, nil
}

func retryFunc(ctx context.Context, intervalTime time.Duration, ec *e2eCluster, fn func(ec *e2eCluster) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(intervalTime):
			err := fn(ec)
			if err == nil {
				return nil
			}
		}
	}
}

func enableKcAgent(f *framework.Framework, ec *e2eCluster, enable bool) error {
	cmd := `systemctl stop kc-agent && rm -rf /etc/kubeclipper-agent-bak && cp -r /etc/kubeclipper-agent /etc/kubeclipper-agent-bak && rm -rf /usr/local/bin/kubeclipper-agent.bak  && cp /usr/local/bin/kubeclipper-agent /usr/local/bin/kubeclipper-agent.bak && rm -rf /usr/lib/systemd/system/kc-agent.service.bak && \cp /usr/lib/systemd/system/kc-agent.service /usr/lib/systemd/system/kc-agent.service.bak`
	if enable {
		cmd = `rm -rf /etc/kubeclipper-agent && cp -r /etc/kubeclipper-agent-bak /etc/kubeclipper-agent && rm -rf /usr/local/bin/kubeclipper-agent && cp /usr/local/bin/kubeclipper-agent.bak /usr/local/bin/kubeclipper-agent && rm -rf /usr/lib/systemd/system/kc-agent.service && cp /usr/lib/systemd/system/kc-agent.service.bak /usr/lib/systemd/system/kc-agent.service && systemctl daemon-reload && systemctl restart kc-agent`
	}
	_, err := sshutils.SSHCmdWithSudo(ec.SSH, ec.NodeIP, cmd)
	if err != nil {
		return errors.WithMessagef(err, "run cmd [%s] on node [%s]", cmd, ec.NodeIP)
	}
	return nil
}

func recoveryNode(ec *e2eCluster) {
	ginkgo.By("The cloud cluster e2e test test is over. Tail work being processed")
	errs := ""
	ginkgo.By(fmt.Sprintf("starting %s kc-agent service", ec.NodeIP))
	err := enableKcAgent(nil, ec, true)
	if err != nil {
		errs = fmt.Sprintf("enable kc-agent err: %v.", err)
	}
	ginkgo.By(fmt.Sprintf("uninstall cluster for %s node", ec.NodeIP))
	err = uninstallAIOCluster(ec)
	if err != nil {
		errs = fmt.Sprintf("uninstall cluster err: %v.", err)
	}
	if errs != "" {
		framework.ExpectNoError(fmt.Errorf("%s", errs))
	}
	ginkgo.By("enable kc-agent and uninstall cluster successfully")
}

func removeCloudProvider(f *framework.Framework, providerName, clusterName string) {
	errs := ""
	ginkgo.By("delete cloud provider")
	err := f.KcClient().DeleteCloudProvider(context.TODO(), providerName)
	if err != nil && !f.KcClient().NotFound(err) {
		errs = fmt.Sprintf("delete cloud provider err: %v.", err)
	}
	ginkgo.By("wait cloud provider remove")
	err = cluster.WaitForCloudProviderNotFound(f.KcClient(), providerName, 5*time.Minute)
	if err != nil && !f.KcClient().NotFound(err) {
		errs = fmt.Sprintf("describe cloud provider err: %v.", err)
	}

	queryString := url.Values{}
	queryString.Set(query.ParameterForce, "true")
	err = f.KcClient().DeleteClusterWithQuery(context.TODO(), clusterName, queryString)
	if err != nil && !f.KcClient().NotFound(err) {
		errs = fmt.Sprintf("delete cluster %s err: %v.", clusterName, err)
	}

	if errs != "" {
		framework.ExpectNoError(fmt.Errorf("%s", errs))
	}
}
