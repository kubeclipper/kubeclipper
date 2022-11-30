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

package create

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

const (
	clusterLongDescription = `
  Create cluster using command line`
	createClusterExample = `
  # Create cluster offline. The default value of offline is true, so it can be omitted.
  kcctl create cluster --name demo --master 192.168.10.123

  # Create cluster online
  kcctl create cluster --name demo --master 192.168.10.123 --offline false --local-registry 192.168.10.123:5000

  # Create cluster with taint manage
  kcctl create cluster --name demo --master 192.168.10.123 --untaint-master

  # Create cluster with worker.
  kcctl create cluster --name demo --master 192.168.10.123 --worker 192.168.10.124

  Please read 'kcctl create cluster -h' get more create cluster flags.`
)

type CreateClusterOptions struct {
	BaseOptions
	Masters       []string
	Workers       []string
	UntaintMaster bool
	Offline       bool
	LocalRegistry string
	CRI           string
	CRIVersion    string
	K8sVersion    string
	CNI           string
	CNIVersion    string
	Name          string
	createdByIP   bool
	CertSans      []string
	CaCertFile    string
	CaKeyFile     string
}

var (
	allowedCRI = sets.NewString("containerd", "docker")
	allowedCNI = sets.NewString("calico")
)

func NewCreateClusterOptions(streams options.IOStreams) *CreateClusterOptions {
	return &CreateClusterOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
		UntaintMaster: false,
		Offline:       true,
		CRI:           "containerd",
		CNI:           "calico",
		createdByIP:   false,
	}
}

func NewCmdCreateCluster(streams options.IOStreams) *cobra.Command {
	o := NewCreateClusterOptions(streams)
	cmd := &cobra.Command{
		Use:                   "cluster (--name) <name> (-m|--master) <id or ip> [(--offline <online> | <offline>)] [(--cri <docker> | <containerd>)] [(--cni <calico> | <others> )] [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "create kubeclipper cluster resource",
		Long:                  clusterLongDescription,
		Example:               createClusterExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.PreRun())
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.RunCreate())
		},
	}
	cmd.Flags().StringVar(&o.Name, "name", "", "k8s cluster name")
	cmd.Flags().StringSliceVarP(&o.Masters, "master", "m", o.Masters, "k8s master node id or ip")
	cmd.Flags().StringSliceVar(&o.Workers, "worker", o.Workers, "k8s worker node id or ip")
	cmd.Flags().BoolVar(&o.UntaintMaster, "untaint-master", o.UntaintMaster, "untaint master node after cluster create")
	cmd.Flags().BoolVar(&o.Offline, "offline", o.Offline, "create cluster online or offline")
	cmd.Flags().StringVar(&o.LocalRegistry, "local-registry", o.LocalRegistry, "use local registry address to pull image")
	cmd.Flags().StringVar(&o.CRI, "cri", o.CRI, "k8s cri type, docker or containerd")
	cmd.Flags().StringVar(&o.CRIVersion, "cri-version", o.CRIVersion, "k8s cri version")
	cmd.Flags().StringVar(&o.K8sVersion, "k8s-version", o.K8sVersion, "k8s version")
	cmd.Flags().StringVar(&o.CNI, "cni", o.CNI, "k8s cni type, calico or others")
	cmd.Flags().StringVar(&o.CNIVersion, "cni-version", o.CNIVersion, "k8s cni version")
	cmd.Flags().StringSliceVar(&o.CertSans, "cert-sans", o.CertSans, "k8s cluster certificate signing ipList or domainList")
	cmd.Flags().StringVar(&o.CaCertFile, "ca-cert", o.CaCertFile, "k8s external root-ca cert file")
	cmd.Flags().StringVar(&o.CaKeyFile, "ca-key", o.CaKeyFile, "k8s external root-ca key file")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("master", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listNode(toComplete, append(o.Masters, o.Workers...)), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("worker", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listNode(toComplete, append(o.Masters, o.Workers...)), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("local-registry", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listRegistry(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("cri", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return allowedCRI.List(), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("cni", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return allowedCNI.List(), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("cri-version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listCRI(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("cni-version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listCNI(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("k8s-version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listK8s(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("name"))
	utils.CheckErr(cmd.MarkFlagRequired("master"))
	return cmd
}

func (l *CreateClusterOptions) Complete(opts *options.CliOptions) error {
	if err := opts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(opts.ToRawConfig())
	if err != nil {
		return err
	}
	l.Client = c
	return nil
}

func (l *CreateClusterOptions) PreRun() error {
	if l.CRIVersion == "" {
		cri := l.listCRI("")
		if len(cri) == 0 {
			return errors.New("no valid cri-version")
		}
		l.CRIVersion = cri[0]
		logger.Infof("use default %s version %s", l.CRI, l.CRIVersion)
	}
	if l.CNIVersion == "" {
		cni := l.listCNI("")
		if len(cni) == 0 {
			return errors.New("no valid cni-version")
		}
		l.CNIVersion = cni[0]
		logger.Infof("use default %s version %s", l.CNI, l.CNIVersion)
	}
	if l.K8sVersion == "" {
		k8s := l.listK8s("")
		if len(k8s) == 0 {
			return errors.New("no valid k8s-version")
		}
		l.K8sVersion = k8s[0]
		logger.Infof("use default k8s version %s", l.K8sVersion)
	}
	return nil
}

func (l *CreateClusterOptions) ValidateArgs(cmd *cobra.Command) error {
	if !allowedCRI.Has(l.CRI) {
		return utils.UsageErrorf(cmd, "unsupported cri,support %v now", allowedCRI.List())
	}
	if !allowedCNI.Has(l.CNI) {
		return utils.UsageErrorf(cmd, "unsupported cni,support %v now", allowedCNI.List())
	}
	if len(l.Masters)%2 == 0 {
		return utils.UsageErrorf(cmd, "master node must be odd")
	}
	if l.Name == "" {
		return utils.UsageErrorf(cmd, "cluster name must be specified")
	}
	k8sVersions := l.listK8s("")
	if !sliceutil.HasString(k8sVersions, l.K8sVersion) {
		return utils.UsageErrorf(cmd, "unsupported k8s version,support %v now", k8sVersions)
	}
	criVersions := l.listCRI("")
	if !sliceutil.HasString(criVersions, l.CRIVersion) {
		return utils.UsageErrorf(cmd, "unsupported cri version,support %v now", criVersions)
	}
	cniVersions := l.listCNI("")
	if !sliceutil.HasString(cniVersions, l.CNIVersion) {
		return utils.UsageErrorf(cmd, "unsupported cni version,support %v now", cniVersions)
	}

	nodes := make([]string, 0)
	nodes = append(nodes, l.Masters...)
	nodes = append(nodes, l.Workers...)
	pre := net.ParseIP(nodes[0])
	for i := 1; i < len(nodes); i++ {
		cur := net.ParseIP(nodes[i])
		if pre == nil && cur != nil || pre != nil && cur == nil {
			return errors.New("only enter id or ip")
		}
		pre = cur
	}
	if pre != nil {
		l.createdByIP = true
	}

	if l.CaCertFile != "" || l.CaKeyFile != "" {
		caCert, err := os.ReadFile(l.CaCertFile)
		if err != nil {
			return err
		}
		caKey, err := os.ReadFile(l.CaKeyFile)
		if err != nil {
			return err
		}

		l.CaCertFile = base64.StdEncoding.EncodeToString(caCert)
		l.CaKeyFile = base64.StdEncoding.EncodeToString(caKey)
	}
	return nil
}

func (l *CreateClusterOptions) RunCreate() error {
	if err := l.transformNodeIP(); err != nil {
		return err
	}
	c := l.newCluster()
	// TODO: check node exist
	resp, err := l.Client.CreateCluster(context.TODO(), c)
	if err != nil {
		return err
	}
	return l.PrintFlags.Print(resp, l.IOStreams.Out)
}

func (l *CreateClusterOptions) transformNodeIP() error {
	if !l.createdByIP {
		return nil
	}
	q := query.New()
	nodes, err := l.Client.ListNodes(context.TODO(), kc.Queries(*q))
	if err != nil {
		return err
	}
	trans := make(map[string]string)
	for _, item := range nodes.Items {
		trans[item.Status.Ipv4DefaultIP] = item.Name
	}
	var (
		i  string
		ok bool
	)
	master := make([]string, 0)
	worker := make([]string, 0)
	for _, m := range l.Masters {
		if i, ok = trans[m]; !ok {
			return fmt.Errorf("node %s does not exist", m)
		}
		master = append(master, i)
	}
	for _, w := range l.Workers {
		if i, ok = trans[w]; !ok {
			return fmt.Errorf("node %s does not exist", w)
		}
		worker = append(worker, i)
	}
	l.Masters = master
	l.Workers = worker
	return nil
}

func (l *CreateClusterOptions) newCluster() *v1.Cluster {
	var annotations = map[string]string{}
	if l.Offline {
		annotations[common.AnnotationOffline] = ""
	}

	c := &v1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "core.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        l.Name,
			Annotations: annotations,
		},

		Masters:           nil,
		Workers:           nil,
		KubernetesVersion: l.K8sVersion,
		CertSANs:          l.CertSans,
		ExternalCaCert:    l.CaCertFile,
		ExternalCaKey:     l.CaKeyFile,
		LocalRegistry:     l.LocalRegistry,
		ContainerRuntime:  v1.ContainerRuntime{},
		Networking: v1.Networking{
			IPFamily:      v1.IPFamilyIPv4,
			Services:      v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterServiceSubnet}},
			Pods:          v1.NetworkRanges{CIDRBlocks: []string{constatns.ClusterPodSubnet}},
			DNSDomain:     "cluster.local",
			ProxyMode:     "ipvs",
			WorkerNodeVip: "169.254.169.100",
		},

		KubeProxy: v1.KubeProxy{},
		Etcd:      v1.Etcd{},
		CNI: v1.CNI{
			LocalRegistry: l.LocalRegistry,
			Type:          l.CNI,
			Version:       l.CNIVersion,
			Calico: &v1.Calico{
				IPv4AutoDetection: "first-found",
				IPv6AutoDetection: "first-found",
				Mode:              "Overlay-Vxlan-All",
				IPManger:          true,
				MTU:               1440,
			},
		},

		Status: v1.ClusterStatus{},
	}

	masters := make([]v1.WorkerNode, 0)
	for _, n := range l.Masters {
		if l.UntaintMaster {
			masters = append(masters, v1.WorkerNode{
				ID:     n,
				Labels: nil,
				Taints: nil,
			})
		} else {
			masters = append(masters, v1.WorkerNode{
				ID:     n,
				Labels: nil,
				Taints: []v1.Taint{
					{
						Key:    "node-role.kubernetes.io/master",
						Value:  "",
						Effect: v1.TaintEffectNoSchedule,
					},
				},
			})
		}
	}
	workers := make([]v1.WorkerNode, 0)
	for _, n := range l.Workers {
		workers = append(workers, v1.WorkerNode{
			ID:     n,
			Labels: nil,
			Taints: nil,
		})
	}
	c.Masters = masters
	c.Workers = workers
	var insecureRegistry []string
	if l.LocalRegistry != "" {
		insecureRegistry = []string{l.LocalRegistry}
	}
	switch l.CRI {
	case "docker":
		c.ContainerRuntime = v1.ContainerRuntime{
			Type:             v1.CRIDocker,
			Version:          l.CRIVersion,
			InsecureRegistry: insecureRegistry,
		}
	case "containerd":
		fallthrough
	default:
		c.ContainerRuntime = v1.ContainerRuntime{
			Type:             v1.CRIContainerd,
			Version:          l.CRIVersion,
			InsecureRegistry: insecureRegistry,
		}
	}
	return c
}

func (l *CreateClusterOptions) listCRI(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	if !allowedCRI.Has(l.CRI) {
		logger.V(2).Infof("unsupported cri %s,support %s now", l.CRI, allowedCRI)
		return nil
	}
	return l.componentVersions(l.CRI, toComplete)
}

func (l *CreateClusterOptions) listCNI(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	if !allowedCNI.Has(l.CNI) {
		logger.V(2).Infof("unsupported cni %s,support %s now", l.CNI, allowedCNI)
		return nil
	}
	return l.componentVersions(l.CNI, toComplete)
}

func (l *CreateClusterOptions) listK8s(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	return l.componentVersions("k8s", toComplete)
}

func (l *CreateClusterOptions) componentVersions(component, toComplete string) []string {
	metas, err := l.Client.GetComponentMeta(context.TODO(), nil)
	if err != nil {
		logger.Errorf("get component meta failed: %s. please check .kc/config", err)
		return nil
	}
	set := sets.NewString()
	for _, resource := range metas.Addons {
		if resource.Name == component && strings.HasPrefix(resource.Version, toComplete) {
			set.Insert(resource.Version)
		}
	}
	return set.List()
}

func (l *CreateClusterOptions) listNode(toComplete string, exclude []string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	nodes := l.nodes(toComplete)
	completions := sliceutil.RemoveString(nodes, func(item string) bool {
		return sliceutil.HasString(exclude, item)
	})
	return completions
}

func (l *CreateClusterOptions) listRegistry(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	return l.registry(toComplete)
}

func (l *CreateClusterOptions) registry(toComplete string) []string {
	list := make([]string, 0)
	template, err := l.Client.GetPlatformSetting(context.TODO())
	if err != nil {
		return nil
	}
	for _, v := range template.InsecureRegistry {
		if strings.HasPrefix(v.Host, toComplete) {
			list = append(list, v.Host)
		}
	}
	return list
}

func (l *CreateClusterOptions) nodes(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = fmt.Sprintf("!%s", common.LabelNodeRole)
	data, err := l.Client.ListNodes(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Status.Ipv4DefaultIP, toComplete) {
			list = append(list, v.Status.Ipv4DefaultIP)
		}
	}
	return list
}
