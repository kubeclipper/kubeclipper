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

package get

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
  Display one or many resources

  Prints a table of the most important information about the specified resources.
  You can filter the list using a label selector and the --selector flag.

  Notice: You must run 'kcctl login' at first, you can get help to run 'kcctl login -h'`
	getExample = `
  # List all users in ps output format.
  kcctl get user

  # List user in json output format
  kcctl get user -o json

  # List user with label-selector
  kcctl get user --selector foo=bar

  # List user with field-selector
  kcctl get user --field-selector .metadata.name=foo

  # Describe user admin
  kcctl get user admin -o yaml

  # List other resource
  kcctl get [role,cluster,node]

  Please read 'kcctl get -h' get more get flags`
)

type GetOptions struct {
	PrintFlags *printer.PrintFlags
	cliOpts    *options.CliOptions
	options.IOStreams
	LabelSelector string
	FieldSelector string
	Watch         bool
	client        *kc.Client
	resource      string
	name          string
}

var (
	allowedResource = sets.NewString(options.ResourceUser, options.ResourceRole, options.ResourceNode, options.ResourceCluster, options.ResourceConfigMap)
)

func NewGetOptions(streams options.IOStreams) *GetOptions {
	return &GetOptions{
		PrintFlags: printer.NewPrintFlags(),
		IOStreams:  streams,
		cliOpts:    options.NewCliOptions(),
	}
}

func NewCmdGet(streams options.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)
	cmd := &cobra.Command{
		Use:                   "get [(-o|--output=)table|json|yaml] (TYPE [NAME | -l label] | TYPE/NAME ...) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Display one or multiple resources",
		Long:                  longDescription,
		Example:               getExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.cliOpts))
			utils.CheckErr(o.ValidateArgs(cmd, args))
			utils.CheckErr(o.RunGet())
		},
		ValidArgsFunction: ValidArgsFunction(o),
	}
	o.cliOpts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", o.Watch, "After listing/getting the requested object, watch for changes.")
	cmd.Flags().StringVarP(&o.LabelSelector, "selector", "l", o.LabelSelector, "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().StringVar(&o.FieldSelector, "field-selector", o.FieldSelector, "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	o.PrintFlags.AddFlags(cmd)
	return cmd
}

func (l *GetOptions) ValidateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return utils.UsageErrorf(cmd, "You must specify the type of resource to get,support %v now", allowedResource.List())
	}
	l.resource = args[0]
	args = args[1:]
	if len(args) > 0 {
		l.name = args[0]
	}
	if !allowedResource.Has(l.resource) {
		return utils.UsageErrorf(cmd, "unsupported resource type,support %v now", allowedResource.List())
	}
	return nil
}

func (l *GetOptions) Complete(opts *options.CliOptions) error {
	if err := opts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(opts.ToRawConfig())
	if err != nil {
		return err
	}
	l.client = c
	return nil
}

func (l *GetOptions) RunGet() error {
	if l.name != "" {
		return l.describe()
	}
	return l.list()
}

func (l *GetOptions) list() error {
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	var (
		result printer.ResourcePrinter
		err    error
	)
	switch l.resource {
	case options.ResourceNode:
		result, err = l.client.ListNodes(context.TODO(), kc.Queries(*q))
	case options.ResourceUser:
		result, err = l.client.ListUsers(context.TODO(), kc.Queries(*q))
	case options.ResourceRole:
		result, err = l.client.ListRoles(context.TODO(), kc.Queries(*q))
	case options.ResourceCluster:
		result, err = l.client.ListClusters(context.TODO(), kc.Queries(*q))
	case options.ResourceConfigMap:
		result, err = l.client.ListConfigMaps(context.TODO(), kc.Queries(*q))
	default:
		return fmt.Errorf("unsupported resource")
	}

	if err != nil {
		return err
	}
	return l.PrintFlags.Print(result, l.IOStreams.Out)
}

func (l *GetOptions) describe() error {
	var (
		result printer.ResourcePrinter
		err    error
	)
	switch l.resource {
	case options.ResourceNode:
		result, err = l.client.DescribeNode(context.TODO(), l.name)
	case options.ResourceUser:
		result, err = l.client.DescribeUser(context.TODO(), l.name)
	case options.ResourceRole:
		result, err = l.client.DescribeRole(context.TODO(), l.name)
	case options.ResourceCluster:
		result, err = l.client.DescribeCluster(context.TODO(), l.name)
	case options.ResourceConfigMap:
		result, err = l.client.DescribeConfigMap(context.TODO(), l.name)
	default:
		return fmt.Errorf("unsupported resource")
	}
	if err != nil {
		return err
	}
	return l.PrintFlags.Print(result, l.IOStreams.Out)
}

func ValidArgsFunction(o *GetOptions) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		utils.CheckErr(o.Complete(o.cliOpts))

		if len(args) == 0 {
			return allowedResource.List(), cobra.ShellCompDirectiveNoFileComp
		}
		switch resource := args[0]; resource {
		case options.ResourceUser:
			return o.listUser(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceRole:
			return o.listRole(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceNode:
			return o.listNode(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceCluster:
			return o.listCluster(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceConfigMap:
			return o.listConfigMaps(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func (l *GetOptions) listUser(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	users, err := l.client.ListUsers(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range users.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}

func (l *GetOptions) listRole(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	data, err := l.client.ListRoles(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}

func (l *GetOptions) listNode(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	data, err := l.client.ListNodes(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}

func (l *GetOptions) listCluster(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	data, err := l.client.ListClusters(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}

func (l *GetOptions) listConfigMaps(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = l.LabelSelector
	q.FieldSelector = l.FieldSelector
	data, err := l.client.ListConfigMaps(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}
