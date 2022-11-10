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

package delete

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/query"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
  Delete kubeclipper resources.

  Currently,only clusters, users and roles resources can be supported.`
	deleteExample = `
  # Delete kubeclipper cluster
  kcctl delete cluster 'CLUSTER-NAME'

  # Delete kubeclipper user
  kcctl delete user 'USER-NAME'

  # Delete kubeclipper role
  kcctl delete role 'ROLE-NAME'

  Please read 'kcctl delete -h' get more delete flags.`
)

type BaseOptions struct {
	PrintFlags *printer.PrintFlags
	CliOpts    *options.CliOptions
	options.IOStreams
	Client *kc.Client
}

type DeleteOptions struct {
	BaseOptions
	resource string
	name     string
	force    bool
}

var (
	allowedResource = sets.NewString(options.ResourceUser, options.ResourceRole, options.ResourceCluster)
)

func NewCmdDelete(streams options.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)
	cmd := &cobra.Command{
		Use:                   "delete (<cluster> | <user> | <role>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "Delete kubeclipper resource",
		Long:                  longDescription,
		Example:               deleteExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.ValidateArgs(cmd, args))
			if !o.precheck() {
				return
			}
			utils.CheckErr(o.RunDelete())
		},
		ValidArgsFunction: ValidArgsFunction(o),
	}
	o.CliOpts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVarP(&o.force, "force", "F", o.force, "Force delete resource. Now is only support cluster.")
	return cmd
}

func NewDeleteOptions(streams options.IOStreams) *DeleteOptions {
	return &DeleteOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func (l *DeleteOptions) Complete(opts *options.CliOptions) error {
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

func (l *DeleteOptions) ValidateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return utils.UsageErrorf(cmd, "You must specify the type of resource to delete,support %v now", allowedResource.List())
	}
	l.resource = args[0]
	if !allowedResource.Has(l.resource) {
		return utils.UsageErrorf(cmd, "unsupported resource type,support %v now", allowedResource.List())
	}
	if len(args) < 2 {
		return utils.UsageErrorf(cmd, "You must specify the name of %s to delete", l.resource)
	}
	args = args[1:]
	if len(args) > 0 {
		l.name = args[0]
	}
	return nil
}

func (l *DeleteOptions) RunDelete() error {
	var err error
	switch l.resource {
	case options.ResourceUser:
		err = l.Client.DeleteUser(context.TODO(), l.name)
		if err != nil {
			return err
		}
	case options.ResourceRole:
		err = l.Client.DeleteRole(context.TODO(), l.name)
		if err != nil {
			return err
		}
	case options.ResourceCluster:
		queryString := url.Values{}
		if l.force {
			queryString.Set(query.ParameterForce, "true")
		}
		err = l.Client.DeleteClusterWithQuery(context.TODO(), l.name, queryString)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported resource")
	}
	return nil
}

func (l *DeleteOptions) precheck() bool {
	if l.force {
		_, _ = l.IOStreams.Out.Write([]byte("Force delete cluster which is in used maybe cause data inconsistency." +
			" Are you sure this cluster are not in used or your really want to force delete?  Please input (yes/no)"))
		return utils.AskForConfirmation()
	}
	return true
}

func ValidArgsFunction(o *DeleteOptions) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		utils.CheckErr(o.Complete(o.CliOpts))
		if len(args) == 0 {
			return allowedResource.List(), cobra.ShellCompDirectiveNoFileComp
		}

		switch args[0] {
		case options.ResourceUser:
			return o.listUser(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceRole:
			return o.listRole(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		case options.ResourceCluster:
			return o.listCluster(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func (l *DeleteOptions) listCluster(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	data, err := l.Client.ListClusters(context.TODO(), kc.Queries(*q))
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

func (l *DeleteOptions) listUser(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	users, err := l.Client.ListUsers(context.TODO(), kc.Queries(*q))
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

func (l *DeleteOptions) listRole(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = "!kubeclipper.io/role-template,!kubeclipper.io/hidden"
	data, err := l.Client.ListRoles(context.TODO(), kc.Queries(*q))
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
