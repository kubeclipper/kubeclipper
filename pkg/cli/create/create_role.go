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
	"encoding/json"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

/*
create kubeclipper role resource

Usage:
  kcctl create role

Examples:
TODO..

Flags:
  -c, --config string   Path to the config file to use for CLI requests.
  -h, --help            help for role
      --name string     role name
  -o, --output string   Output format either: json,yaml,table (default "table")
      --rules strings   role template rules
*/

const (
	roleLongDescription = `
  Create role using command line`
	createRoleExample = `
  # Create role has permission to view cluster
  kcctl create role --name cluster_viewer --rules=role-template-view-clusters

  # Create role has permission to view cluster and user
  kcctl create role --name viewer --rules=role-template-view-clusters  --rules=role-template-view-users
  
  You can use cmd kcctl get role --selector=kubeclipper.io/role-template=true to query rules.
  Please read 'kcctl create role -h' get more create role flags.`
)

type CreateRoleOptions struct {
	BaseOptions
	Name             string
	Rules            []string
	aggregationRoles string
}

func NewCreateRoleOptions(streams options.IOStreams) *CreateRoleOptions {
	return &CreateRoleOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func NewCmdCreateRole(streams options.IOStreams) *cobra.Command {
	o := NewCreateRoleOptions(streams)
	cmd := &cobra.Command{
		Use:                   "role (--rules <rules>)",
		DisableFlagsInUseLine: true,
		Short:                 "create kubeclipper role resource",
		Long:                  roleLongDescription,
		Example:               createRoleExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunCreate())
		},
	}
	cmd.Flags().StringVar(&o.Name, "name", "", "role name")
	cmd.Flags().StringSliceVar(&o.Rules, "rules", o.Rules, "role template rules (separated by comma)")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("rules", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listRule(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.MarkFlagRequired("name"))
	utils.CheckErr(cmd.MarkFlagRequired("rules"))
	return cmd
}

func (l *CreateRoleOptions) Complete(opts *options.CliOptions) error {
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

func (l *CreateRoleOptions) ValidateArgs(cmd *cobra.Command) error {
	if l.Name == "" {
		return utils.UsageErrorf(cmd, "role name must be specified")
	}
	if len(l.Rules) == 0 {
		return utils.UsageErrorf(cmd, "role must have aggregationRoles")
	}
	return nil
}

func (l *CreateRoleOptions) RunCreate() error {
	// TODO: check role template
	ruleByte, err := json.Marshal(l.Rules)
	if err != nil {
		return err
	}
	l.aggregationRoles = string(ruleByte)
	c := l.newRole()
	resp, err := l.Client.CreateRole(context.TODO(), c)
	if err != nil {
		return err
	}
	return l.PrintFlags.Print(resp, l.IOStreams.Out)
}

func (l *CreateRoleOptions) newRole() *v1.GlobalRole {
	return &v1.GlobalRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindGlobalRole,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: l.Name,
			Annotations: map[string]string{
				"kubeclipper.io/aggregation-roles": l.aggregationRoles,
			},
		},
		Rules: nil,
	}
}

func (l *CreateRoleOptions) listRule(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))
	rules := l.rules(toComplete)
	list := sliceutil.RemoveString(rules, func(item string) bool {
		return sliceutil.HasString(l.Rules, item)
	})
	return list
}

func (l *CreateRoleOptions) rules(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = "kubeclipper.io/role-template=true"
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
