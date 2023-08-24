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
	"os"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/scheme"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
  Create specified resource

  Using the create command to create cluster, user, or role resources.
  Or you can choose to create those directly from a file.`
	createExample = `
  # Create cluster offline. The default value of offline is true, so it can be omitted.
  kcctl create cluster --name demo --master 192.168.10.123

  # Create role has permission to view cluster
  kcctl create role --name cluster_viewer --rules=role-template-view-clusters

  # Create user with required parameters
  kcctl create user --name simple-user --role=platform-view --password 123456 --phone 10086 --email simple@example.com
  
  # Create cluster use cluster.yaml. 
  kcctl create -f cluster.yaml

cluster.yaml example:

kind: Cluster
apiVersion: core.kubeclipper.io/v1
metadata:
  annotations:
    kubeclipper.io/offline: "true"
    #kubeclipper.io/only-install-kubernetes-component: "true"
  name: test
kubernetesVersion: v1.27.4
localRegistry: ""
masters:
  - id: 088885e3-4098-413d-a7e7-39adf0ffa95f
#    labels:
#      test: "1234"
#    taints:
#      - key: node-role.kubernetes.io/control-plane
#        value: ""
#        effect: NoSchedule
workers: []
certSans: []
#featureGates:
#  xxx: true
cni:
  calico:
    IPManger: true
    IPv4AutoDetection: first-found
    IPv6AutoDetection: first-found
    mode: Overlay-Vxlan-All
    mtu: 1440
  criType: containerd
  localRegistry: ""
  namespace: calico-system
  offline: true
  type: calico
  version: v3.26.1
containerRuntime:
  type: containerd
  version: 1.6.4
etcd:
  dataDir: /var/lib/etcd
kubeProxy: {}
kubelet:
  ipAsName: false
  rootDir: /var/lib/kubelet
networking:
  dnsDomain: cluster.local
  ipFamily: IPv4
  pods:
    cidrBlocks:
      - "172.25.0.0/16"
  proxyMode: ipvs
  services:
    cidrBlocks:
      - "10.96.0.0/12"
  workerNodeVip: 169.254.169.100`
)

type BaseOptions struct {
	PrintFlags *printer.PrintFlags
	CliOpts    *options.CliOptions
	options.IOStreams
	Client *kc.Client
}

type CreateOptions struct {
	BaseOptions
	Filename string
}

func NewCreateOptions(streams options.IOStreams) *CreateOptions {
	return &CreateOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func NewCmdCreate(streams options.IOStreams) *cobra.Command {
	o := NewCreateOptions(streams)
	cmd := &cobra.Command{
		Use:                   "create (--filename | -f <FILE-NAME>)",
		DisableFlagsInUseLine: true,
		Short:                 "Create kubeclipper resource",
		Long:                  longDescription,
		Example:               createExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.Run(cmd))
		},
	}
	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", "use resource file to create")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	cmd.AddCommand(NewCmdCreateCluster(streams))
	cmd.AddCommand(NewCmdCreateRole(streams))
	cmd.AddCommand(NewCmdCreateUser(streams))
	return cmd
}

func (l *CreateOptions) Complete(opts *options.CliOptions) error {
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

func (l *CreateOptions) Run(cmd *cobra.Command) error {
	rData, err := os.ReadFile(l.Filename)
	if err != nil {
		return err
	}
	obj := unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	objRet, gvk, err := dec.Decode(rData, nil, &obj)
	if err != nil {
		return err
	}
	switch gvk.GroupKind().String() {
	case corev1.Kind(corev1.KindCluster).String():
		cluster := &corev1.Cluster{}
		if err := scheme.Scheme.Convert(objRet, cluster, nil); err != nil {
			return err
		}
		l.MergeDefaultCluster(cluster)
		if err := l.ValidateCluster(cmd, cluster); err != nil {
			return err
		}
		cluster.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind(corev1.KindCluster))
		resp, err := l.Client.CreateCluster(context.TODO(), cluster)
		if err != nil {
			return err
		}

		return l.PrintFlags.Print(resp, l.IOStreams.Out)
	}

	return nil
}

func (l *CreateOptions) MergeDefaultCluster(cluster *corev1.Cluster) {
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	_, ok := cluster.Annotations[common.AnnotationOffline]
	if !ok {
		cluster.Annotations[common.AnnotationOffline] = "true"
	}
	if cluster.Etcd.DataDir == "" {
		cluster.Etcd.DataDir = "/var/lib/etcd"
	}
	if cluster.Kubelet.RootDir == "" {
		cluster.Kubelet.RootDir = "/var/lib/kubelet"
	}
	if len(cluster.Networking.Pods.CIDRBlocks) == 0 {
		cluster.Networking.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	}
	if len(cluster.Networking.Services.CIDRBlocks) == 0 {
		cluster.Networking.Services.CIDRBlocks = []string{"10.96.0.0/12"}
	}
	if cluster.Networking.DNSDomain == "" {
		cluster.Networking.DNSDomain = "cluster.local"
	}
	if cluster.Networking.IPFamily == "" {
		cluster.Networking.IPFamily = corev1.IPFamilyIPv4
	}
	if cluster.Networking.WorkerNodeVip == "" {
		cluster.Networking.WorkerNodeVip = "169.254.169.100"
	}
}

func (l *CreateOptions) ValidateCluster(cmd *cobra.Command, cluster *corev1.Cluster) error {
	_, offline := cluster.Annotations[common.AnnotationOffline]
	_, onlyInstallK8sComp := cluster.Annotations[common.AnnotationOnlyInstallKubernetesComp]
	if !allowedCRI.Has(cluster.ContainerRuntime.Type) {
		return utils.UsageErrorf(cmd, "unsupported cri,support %v now", allowedCRI.List())
	}
	allowedCRIVersion := getComponentAvailableVersions(l.Client, offline, cluster.ContainerRuntime.Type)
	if !allowedCRIVersion.Has(cluster.ContainerRuntime.Version) {
		return utils.UsageErrorf(cmd, "unsupported cri version,support %v now", allowedCRIVersion.UnsortedList())
	}
	allowedK8sVersion := getComponentAvailableVersions(l.Client, offline, "k8s")
	if !allowedK8sVersion.Has(cluster.KubernetesVersion) {
		return utils.UsageErrorf(cmd, "unsupported kubernetes version,support %v now", allowedK8sVersion.UnsortedList())
	}
	if !allowedCNI.Has(cluster.CNI.Type) {
		logger.V(2).Infof("unsupported cni %s,support %s now", cluster.CNI.Type, allowedCNI)
		return nil
	}
	if !onlyInstallK8sComp {
		allCNIVersion := getComponentAvailableVersions(l.Client, offline, cluster.CNI.Type)
		if !allCNIVersion.Has(cluster.CNI.Version) {
			return utils.UsageErrorf(cmd, "unsupported cni version,support %v now", allCNIVersion.UnsortedList())
		}
	}
	return nil
}
