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
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
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
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(cmd, args))
			utils.CheckErr(o.Validate())
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

func (o *SetClusterOptions) Validate() error {
	if !o.changedExternalIP && !o.changedExternalDomain && !o.changedExternalPort && !o.changedExternalDomainPort {
		return errors.New("at least one flag must be specified, use --external-ip, --external-domain, --external-port, or --external-domain-port")
	}

	if o.changedExternalIP && o.ExternalIP != "" {
		if !netutil.IsValidIP(o.ExternalIP) {
			return fmt.Errorf("invalid external IP: %s", o.ExternalIP)
		}
	}

	if o.changedExternalDomain && o.ExternalDomain != "" {
		if err := netutil.IsValidDomain(o.ExternalDomain); err != nil {
			return fmt.Errorf("invalid external domain: %s: %s", o.ExternalDomain, err)
		}
	}

	if o.changedExternalPort && o.ExternalPort != "" {
		if err := validatePort(o.ExternalPort); err != nil {
			return fmt.Errorf("invalid external port: %s", err)
		}
	}

	if o.changedExternalDomainPort && o.ExternalDomainPort != "" {
		if err := validatePort(o.ExternalDomainPort); err != nil {
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

func validatePort(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%s is not a valid port number", portStr)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}
