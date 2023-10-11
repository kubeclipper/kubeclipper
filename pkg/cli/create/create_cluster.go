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
	"strconv"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	"github.com/kubeclipper/kubeclipper/pkg/utils/autodetection"

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

const CalicoNetModeDescription = `
The following sections describe the available calico network modes.

1. BGP
Using the pod network in BGP mode, the pod network can be easily connected to the physical network with the best 
performance. It is suitable for bare metal environments and network environments that support the BGP protocol.

2. Overlay-IPIP-All
A pod network in overlay mode using IP-in-IP technology, suitable for environments where all underlying platforms support IPIP.

3. Overlay-IPIP-Cross-Subnet
Use the overlay mode pod network of IP-in-IP technology when communicating on different network segments, host routing 
when communicating on the same network segment, suitable for bare metal environments with complex network environments.

4. Overlay-Vxlan-All
The overlay mode pod network using vxlan technology is suitable for almost all platforms but the performance is reduced.

5. Overlay-Vxlan-Cross-Subnet
Use the overlay mode pod network of vxlan technology when communicating on different network segments, and host routing 
when communicating on the same network segment, suitable for bare metal environments with complex network environments.
`

const IPDetectDescription = `
When Calico is used for routing, each node must be configured with an IPv4 address and/or an IPv6 address that 
will beused to route between nodes. To eliminate node specific IP address configuration, the calico/node container
can be configuredto autodetect these IP addresses. In many systems, there might be multiple physical interfaces on
a host, or possibly multipleIP addresses configured on a physical interface. In these cases, there are multiple
addresses to choose from and so autodetectionof the correct address can be tricky.

The IP autodetection methods are provided to improve the selection of the correct address, by limiting the 
selection basedon suitable criteria for your deployment.

The following sections describe the available IP autodetection methods.

1. first-found
The first-found option enumerates all interface IP addresses and returns the first valid IP address 
(based on IP versionand type of address) on the first valid interface. 
Certain known “local” interfaces are omitted, such as the docker bridge.The order that both the interfaces 
and the IPaddresses are listed is system dependent.

This is the default detection method. 
However, since this method only makes a very simplified guess,it is recommended to either configure the node with
a specific IP address,or to use one of the other detection methods.

2. interface=INTERFACE-REGEX
The interface method uses the supplied interface regular expression to enumerate matching interfaces and to 
return thefirst IP address on the first matching interface. 
The order that both the interfaces and the IP addresses are listed is system dependent.

Example with valid IP address on interface eth0, eth1, eth2 etc.:
interface=eth.*

3. can-reach=DESTINATION
The can-reach method uses your local routing to determine which IP address will be used to reach the supplied
destination.Both IP addresses and domain names may be used.

Example using IP addresses:
IP_AUTODETECTION_METHOD=can-reach=8.8.8.8
IP6_AUTODETECTION_METHOD=can-reach=2001:4860:4860::8888

Example using domain names:
IP_AUTODETECTION_METHOD=can-reach=www.google.com
IP6_AUTODETECTION_METHOD=can-reach=www.google.com`

type CreateClusterOptions struct {
	BaseOptions
	Masters                   []string
	Workers                   []string
	UntaintMaster             bool
	Offline                   bool
	LocalRegistry             string
	InsecureRegistries        []string
	CRI                       string
	CRIVersion                string
	K8sVersion                string
	CNI                       string
	CNIVersion                string
	Name                      string
	createdByIP               bool
	CertSans                  []string
	CaCertFile                string
	CaKeyFile                 string
	DNSDomain                 string
	CalicoNetMode             string
	IPv4AutoDetection         string
	ServiceSubnet             string
	PodSubnet                 string
	OnlyInstallKubernetesComp bool
	FeatureGatesString        []string
	FeatureGates              map[string]bool
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
		UntaintMaster:     false,
		Offline:           true,
		CRI:               "containerd",
		CNI:               "calico",
		createdByIP:       false,
		DNSDomain:         "cluster.local",
		CalicoNetMode:     "Overlay-Vxlan-All",
		IPv4AutoDetection: autodetection.MethodFirst,
		ServiceSubnet:     "10.96.0.0/12",
		PodSubnet:         "172.25.0.0/16",
		FeatureGates:      map[string]bool{},
	}
}

func NewCmdCreateCluster(streams options.IOStreams) *cobra.Command {
	o := NewCreateClusterOptions(streams)
	cmd := &cobra.Command{
		Use:                   "cluster (--name) <name> (-m|--master) <id or ip> [(--offline <false> | <true>)] [(--cri <docker> | <containerd>)] [(--cni <calico> | <others> )] [flags]",
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
	cmd.Flags().BoolVar(&o.Offline, "offline", o.Offline, "create cluster online(false) or offline(true)")
	cmd.Flags().StringVar(&o.LocalRegistry, "local-registry", o.LocalRegistry, "use local registry address to pull image")
	cmd.Flags().StringSliceVar(&o.InsecureRegistries, "insecure-registry", o.InsecureRegistries, "use remote registry address to pull image")
	cmd.Flags().StringVar(&o.CRI, "cri", o.CRI, "k8s cri type, docker or containerd")
	cmd.Flags().StringVar(&o.CRIVersion, "cri-version", o.CRIVersion, "k8s cri version")
	cmd.Flags().StringVar(&o.K8sVersion, "k8s-version", o.K8sVersion, "k8s version")
	cmd.Flags().StringVar(&o.CNI, "cni", o.CNI, "k8s cni type, calico or others")
	cmd.Flags().StringVar(&o.CNIVersion, "cni-version", o.CNIVersion, "k8s cni version")
	cmd.Flags().StringSliceVar(&o.CertSans, "cert-sans", o.CertSans, "k8s cluster certificate signing ipList or domainList")
	cmd.Flags().StringVar(&o.CaCertFile, "ca-cert", o.CaCertFile, "k8s external root-ca cert file")
	cmd.Flags().StringVar(&o.CaKeyFile, "ca-key", o.CaKeyFile, "k8s external root-ca key file")
	cmd.Flags().StringVar(&o.DNSDomain, "cluster-dns-domain", o.DNSDomain, "k8s cluster domain")
	cmd.Flags().StringVar(&o.IPv4AutoDetection, "calico.ipv4-auto-detection", o.IPv4AutoDetection, fmt.Sprintf("node ipv4 auto detection. \n%s", IPDetectDescription))
	cmd.Flags().StringVar(&o.CalicoNetMode, "calico.net-mode", o.CalicoNetMode, "calico network mode, support [BGP|Overlay-IPIP-All|Overlay-IPIP-Cross-Subnet|Overlay-Vxlan-All|Overlay-Vxlan-Cross-Subnet] now. \n"+CalicoNetModeDescription)
	cmd.Flags().StringVar(&o.ServiceSubnet, "service-subnet", o.ServiceSubnet, "serviceSubnet is the subnet used by Kubernetes Services. Defaults to '10.96.0.0/12'")
	cmd.Flags().StringVar(&o.PodSubnet, "pod-subnet", o.PodSubnet, "podSubnet is the subnet used by Pods. Defaults to '172.25.0.0/16'")
	cmd.Flags().BoolVar(&o.OnlyInstallKubernetesComp, "only-install-kubernetes-component", o.OnlyInstallKubernetesComp, "only install kubernetes component, not install cni")
	cmd.Flags().StringSliceVar(&o.FeatureGatesString, "feature-gates", o.FeatureGatesString, "k8s feature gates, format as: --feature-gates=xxx=true|false")
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
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("insecure-registry", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.registry(toComplete), cobra.ShellCompDirectiveNoFileComp
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
	if l.CalicoNetMode != "" && !sliceutil.HasString([]string{
		"BGP", "Overlay-IPIP-All", "Overlay-IPIP-Cross-Subnet", "Overlay-Vxlan-All", "Overlay-Vxlan-Cross-Subnet",
	}, l.CalicoNetMode) {
		return utils.UsageErrorf(cmd, "unsupported calico net mode, support [BGP|Overlay-IPIP-All|Overlay-IPIP-Cross-Subnet|Overlay-Vxlan-All|Overlay-Vxlan-Cross-Subnet] now")
	}
	if l.IPv4AutoDetection != "" && !autodetection.CheckCalicoMethod(l.IPv4AutoDetection) {
		return utils.UsageErrorf(cmd, "unsupported ip detect method, support [first-found,interface=xxx,can-reach=xxx] now")
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

	for _, fg := range l.FeatureGatesString {
		if strings.Contains(fg, "=") {
			arr := strings.Split(fg, "=")
			if len(arr) != 2 {
				continue
			}
			k := strings.TrimSpace(arr[0])
			v := strings.TrimSpace(arr[1])
			boolValue, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("invalid value %v for feature-gate key: %s, use true|false instead", v, k)
			}
			l.FeatureGates[k] = boolValue
		}
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
	if l.OnlyInstallKubernetesComp {
		annotations[common.AnnotationOnlyInstallKubernetesComp] = "true"
	}
	if l.ServiceSubnet == "" {
		l.ServiceSubnet = constatns.ClusterServiceSubnet
	}
	if l.PodSubnet == "" {
		l.PodSubnet = constatns.ClusterPodSubnet
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
			Services:      v1.NetworkRanges{CIDRBlocks: []string{l.ServiceSubnet}},
			Pods:          v1.NetworkRanges{CIDRBlocks: []string{l.PodSubnet}},
			DNSDomain:     l.DNSDomain,
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
				IPv4AutoDetection: l.IPv4AutoDetection,
				IPv6AutoDetection: "first-found",
				Mode:              l.CalicoNetMode,
				IPManger:          true,
				MTU:               1440,
			},
		},
		FeatureGates: l.FeatureGates,

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
			InsecureRegistry: append(insecureRegistry, l.InsecureRegistries...),
		}
	case "containerd":
		fallthrough
	default:
		c.ContainerRuntime = v1.ContainerRuntime{
			Type:             v1.CRIContainerd,
			Version:          l.CRIVersion,
			InsecureRegistry: append(insecureRegistry, l.InsecureRegistries...),
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
	var metas *kc.ComponentMeta
	var err error
	if l.Offline {
		metas, err = l.Client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"false"}})
	} else {
		metas, err = l.Client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"true"}})
	}
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
