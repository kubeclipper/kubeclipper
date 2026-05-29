/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package set

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
)

const (
	setClusterLongDesc = `
	Set cluster external access configuration.

	You can update or clear the external IP, domain, and port used to access the Kubernetes API server.
	Pass an empty string to clear a value, e.g. --external-ip="".
`
	setClusterExample = `
	# Set external IP for cluster
	kcctl set cluster my-cluster --external-ip=1.2.3.4

	# Set external domain and port
	kcctl set cluster my-cluster --external-domain=api.example.com --external-domain-port=8443

	# Clear external IP
	kcctl set cluster my-cluster --external-ip=""
`
)

type SetClusterOptions struct {
	PrintFlags *printer.PrintFlags
	CliOpts    *options.CliOptions
	options.IOStreams
	Client *kc.Client

	ClusterName        string
	ExternalIP         string
	ExternalDomain     string
	ExternalPort       string
	ExternalDomainPort string

	changedExternalIP         bool
	changedExternalDomain     bool
	changedExternalPort       bool
	changedExternalDomainPort bool
}

func NewSetClusterOptions(streams options.IOStreams) *SetClusterOptions {
	return &SetClusterOptions{
		PrintFlags: printer.NewPrintFlags(),
		CliOpts:    options.NewCliOptions(),
		IOStreams:  streams,
	}
}

func NewCmdSetCluster(streams options.IOStreams) *cobra.Command {
	o := NewSetClusterOptions(streams)
	cmd := &cobra.Command{
		Use:                   "cluster <cluster-name> [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Set cluster external access configuration",
		Long:                  setClusterLongDesc,
		Example:               setClusterExample,
		Args:                  cobra.ExactArgs(1),
		ValidArgsFunction:     o.completeClusterNames,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(cmd, args))
			utils.CheckErr(o.Validate(cmd))
			utils.CheckErr(o.Run())
		},
	}

	cmd.Flags().StringVar(&o.ExternalIP, "external-ip", o.ExternalIP, "k8s apiserver external IP, set empty string to clear")
	cmd.Flags().StringVar(&o.ExternalDomain, "external-domain", o.ExternalDomain, "k8s apiserver external domain, set empty string to clear")
	cmd.Flags().StringVar(&o.ExternalPort, "external-port", o.ExternalPort, "k8s apiserver external port (1-65535), set empty string to clear")
	cmd.Flags().StringVar(&o.ExternalDomainPort, "external-domain-port", o.ExternalDomainPort, "k8s apiserver external domain port (1-65535), set empty string to clear")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	return cmd
}

func (o *SetClusterOptions) Complete(cmd *cobra.Command, args []string) error {
	if err := o.CliOpts.Complete(); err != nil {
		return err
	}
	client, err := kc.FromConfig(o.CliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	o.Client = client
	o.ClusterName = args[0]

	o.changedExternalIP = cmd.Flags().Changed("external-ip")
	o.changedExternalDomain = cmd.Flags().Changed("external-domain")
	o.changedExternalPort = cmd.Flags().Changed("external-port")
	o.changedExternalDomainPort = cmd.Flags().Changed("external-domain-port")

	return nil
}

func (o *SetClusterOptions) Validate(cmd *cobra.Command) error {
	if !o.changedExternalIP && !o.changedExternalDomain &&
		!o.changedExternalPort && !o.changedExternalDomainPort {
		return utils.UsageErrorf(cmd,
			"at least one flag must be specified, use --external-ip, --external-domain, --external-port, or --external-domain-port")
	}

	if err := o.validateExternalIP(); err != nil {
		return err
	}
	if err := o.validateExternalDomain(); err != nil {
		return err
	}
	if err := o.validateExternalPort(); err != nil {
		return err
	}
	if err := o.validateExternalDomainPort(); err != nil {
		return err
	}

	return nil
}

func (o *SetClusterOptions) validateExternalIP() error {
	if o.changedExternalIP && o.ExternalIP != "" && !netutil.IsValidIP(o.ExternalIP) {
		return fmt.Errorf("invalid external IP: %s", o.ExternalIP)
	}
	return nil
}

func (o *SetClusterOptions) validateExternalDomain() error {
	if o.changedExternalDomain && o.ExternalDomain != "" {
		if err := netutil.IsValidDomain(o.ExternalDomain); err != nil {
			return fmt.Errorf("invalid external domain: %s: %s", o.ExternalDomain, err)
		}
	}
	return nil
}

func (o *SetClusterOptions) validateExternalPort() error {
	if o.changedExternalPort && o.ExternalPort != "" {
		if err := netutil.IsValidPortStr(o.ExternalPort); err != nil {
			return fmt.Errorf("invalid external port: %s", err)
		}
	}
	return nil
}

func (o *SetClusterOptions) validateExternalDomainPort() error {
	if o.changedExternalDomainPort && o.ExternalDomainPort != "" {
		if err := netutil.IsValidPortStr(o.ExternalDomainPort); err != nil {
			return fmt.Errorf("invalid external domain port: %s", err)
		}
	}
	return nil
}

func (o *SetClusterOptions) Run() error {
	clusterList, err := o.Client.DescribeCluster(context.TODO(), o.ClusterName)
	if err != nil {
		return err
	}
	if clusterList == nil || len(clusterList.Items) == 0 {
		return fmt.Errorf("cluster %q not found", o.ClusterName)
	}
	cluster := clusterList.Items[0]

	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}

	if o.changedExternalIP {
		if o.ExternalIP == "" {
			delete(cluster.Labels, common.LabelExternalIP)
		} else {
			cluster.Labels[common.LabelExternalIP] = o.ExternalIP
		}
	}

	if o.changedExternalDomain {
		if o.ExternalDomain == "" {
			delete(cluster.Labels, common.LabelExternalDomain)
		} else {
			cluster.Labels[common.LabelExternalDomain] = o.ExternalDomain
		}
	}

	if o.changedExternalPort {
		if o.ExternalPort == "" {
			delete(cluster.Labels, common.LabelExternalPort)
		} else {
			cluster.Labels[common.LabelExternalPort] = o.ExternalPort
		}
	}

	if o.changedExternalDomainPort {
		if o.ExternalDomainPort == "" {
			delete(cluster.Labels, common.LabelExternalDomainPort)
		} else {
			cluster.Labels[common.LabelExternalDomainPort] = o.ExternalDomainPort
		}
	}

	if err := o.Client.UpdateCluster(context.TODO(), &cluster); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "cluster %q updated\n", o.ClusterName)
	return nil
}

func (o *SetClusterOptions) completeClusterNames(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := o.CliOpts.Complete(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client, err := kc.FromConfig(o.CliOpts.ToRawConfig())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return o.listCluster(client, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func (*SetClusterOptions) listCluster(client *kc.Client, toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.Limit = 100
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	data, err := client.ListClusters(ctx, kc.Queries(*q))
	if err != nil {
		return nil
	}
	for i := range data.Items {
		if strings.HasPrefix(data.Items[i].Name, toComplete) {
			list = append(list, data.Items[i].Name)
		}
	}
	return list
}
