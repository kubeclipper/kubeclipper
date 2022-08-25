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
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

const (
	userLongDescription = `
  Create user using command line`
	createUserExample = `
  # Create user with required parameters
  kcctl create user --name simple-user --role=platform-view --password 123456 --phone 10086 --email simple@example.com

  # Create user with all parameters
  kcctl create user --name full-user --role=platform-view --password 123456 --phone 10010 --email full@example.com --description 'a full info user' --display-name 'full'

  Please read 'kcctl create user -h' get more create user flags.`
)

type CreateUserOptions struct {
	BaseOptions
	Name        string
	Role        string
	Email       string
	Description string
	DisplayName string
	Password    string
	Phone       string
}

func NewCreateUserOptions(streams options.IOStreams) *CreateUserOptions {
	return &CreateUserOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func NewCmdCreateUser(streams options.IOStreams) *cobra.Command {
	o := NewCreateUserOptions(streams)
	cmd := &cobra.Command{
		Use:                   "user (--name) (--role) (--password) (--phone) (--email) [flag]",
		DisableFlagsInUseLine: true,
		Short:                 "create kubeclipper user resource",
		Long:                  userLongDescription,
		Example:               createUserExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunCreate())
		},
	}
	cmd.Flags().StringVar(&o.Name, "name", "", "user name")
	cmd.Flags().StringVar(&o.Role, "role", "", "user role")
	cmd.Flags().StringVar(&o.Email, "email", "", "user email address")
	cmd.Flags().StringVar(&o.Description, "description", "", "user description")
	cmd.Flags().StringVar(&o.DisplayName, "display-name", "", "user display name")
	cmd.Flags().StringVar(&o.Password, "password", "", "user password")
	cmd.Flags().StringVar(&o.Phone, "phone", "", "user phone number")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("role", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listRoles(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.MarkFlagRequired("password")
	_ = cmd.MarkFlagRequired("phone")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func (l *CreateUserOptions) Complete(opts *options.CliOptions) error {
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

func (l *CreateUserOptions) ValidateArgs(cmd *cobra.Command) error {
	if l.Name == "" {
		return utils.UsageErrorf(cmd, "user name must be specified")
	}
	if l.Role == "" {
		return utils.UsageErrorf(cmd, "user role must be specified")
	}
	if l.Password == "" {
		return utils.UsageErrorf(cmd, "user password cannot be empty")
	}
	if l.Phone == "" {
		return utils.UsageErrorf(cmd, "user phone cannot be empty")
	}
	if l.Email == "" {
		return utils.UsageErrorf(cmd, "user email cannot be empty")
	}
	return nil
}

func (l *CreateUserOptions) RunCreate() error {
	// TODO: check user template
	c := l.newUser()
	resp, err := l.Client.CreateUser(context.TODO(), c)
	if err != nil {
		return err
	}
	return l.PrintFlags.Print(resp, l.IOStreams.Out)
}

func (l *CreateUserOptions) newUser() *v1.User {
	return &v1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindUser,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: l.Name,
			Annotations: map[string]string{
				common.RoleAnnotation: l.Role,
			},
		},
		Spec: v1.UserSpec{
			Email:             l.Email,
			Lang:              "",
			Phone:             l.Phone,
			Description:       l.Description,
			DisplayName:       l.DisplayName,
			Groups:            nil,
			EncryptedPassword: l.Password,
		},
	}
}

func (l *CreateUserOptions) listRoles(toComplete string) []string {
	utils.CheckErr(l.Complete(l.CliOpts))

	return l.roles(toComplete)
}

func (l *CreateUserOptions) roles(toComplete string) []string {
	list := make([]string, 0)
	q := query.New()
	q.LabelSelector = "!kubeclipper.io/role-template,!kubeclipper.io/hidden"
	data, err := l.Client.ListRoles(context.TODO(), kc.Queries(*q))
	if err != nil {
		logger.Warnf("Failed to list roles: %v", err)
		return nil
	}
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			list = append(list, v.Name)
		}
	}
	return list
}
