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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
)

const (
	registryLongDescription = `
  Create registry using command line`
	createRegistryExample = `
  # Create registry with required parameters
  kcctl create registry --name myregistry --host 1.1.1.1:5000

  # Create registry with all parameters
  kcctl create registry --name myregistry --host 1.1.1.1:5000 --scheme http --skip-tls-verify true --ca xxxxx --auth-username myuser --auth-password myassword --description 'a full info registry''

  Please read 'kcctl create registry -h' get more create registry flags.`
)

var SupportRegistrySchemes = []string{"http", "https"}

type CreateRegistryOptions struct {
	BaseOptions
	Description   string
	Name          string // registry name
	Host          string // registry host
	Scheme        string // http or https
	SkipTLSVerify bool   // is skip tls verify
	CA            string // ca
	// Registry Auth Info
	Username string
	Password string
}

func NewCreateRegistryOptions(streams options.IOStreams) *CreateRegistryOptions {
	return &CreateRegistryOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
		Scheme:        "http",
		SkipTLSVerify: true,
	}
}

func NewCmdCreateRegistry(streams options.IOStreams) *cobra.Command {
	o := NewCreateRegistryOptions(streams)
	cmd := &cobra.Command{
		Use:                   "registry (--host) (--scheme) (--ca) (--skip-tls-versify) (--auth-username) (--auth-password) [flag]",
		DisableFlagsInUseLine: true,
		Short:                 "create kubeclipper registry resource",
		Long:                  registryLongDescription,
		Example:               createRegistryExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunCreate())
		},
	}
	cmd.Flags().StringVar(&o.Name, "name", "", "registry name")
	cmd.Flags().StringVar(&o.Scheme, "scheme", o.Scheme, "registry scheme")
	cmd.Flags().StringVar(&o.Host, "host", "", "registry host")
	cmd.Flags().BoolVar(&o.SkipTLSVerify, "skip-tls-verify", o.SkipTLSVerify, "user email address")
	cmd.Flags().StringVar(&o.Description, "description", "", "registry description")
	cmd.Flags().StringVar(&o.CA, "ca", "", "registry ca")
	cmd.Flags().StringVar(&o.Username, "auth-username", "", "registry auth username")
	cmd.Flags().StringVar(&o.Password, "auth-password", "", "registry auth password")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("host")
	return cmd
}

func (l *CreateRegistryOptions) Complete(opts *options.CliOptions) error {
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

func (l *CreateRegistryOptions) ValidateArgs(cmd *cobra.Command) error {
	if l.Name == "" {
		return utils.UsageErrorf(cmd, "registry name must be specified")
	}
	if l.Scheme == "" {
		return utils.UsageErrorf(cmd, "registry scheme must be one of %v", SupportRegistrySchemes)
	}
	if l.Host == "" {
		return utils.UsageErrorf(cmd, "registry host cannot be empty")
	}

	userEmpty := l.Username == ""
	passwordEmpty := l.Password == ""
	if userEmpty != passwordEmpty {
		return utils.UsageErrorf(cmd, "registry auth-username、auth-password must be all specified or all empty")
	}
	return nil
}

func (l *CreateRegistryOptions) RunCreate() error {
	c := l.newRegistry()
	resp, err := l.Client.CreateRegistry(context.TODO(), c)
	if err != nil {
		return err
	}
	return l.PrintFlags.Print(resp, l.IOStreams.Out)
}

func (l *CreateRegistryOptions) newRegistry() *v1.Registry {
	reg := &v1.Registry{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.KindRegistry,
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: l.Name,
			Annotations: map[string]string{
				common.AnnotationDescription: l.Description,
			},
		},
		RegistrySpec: v1.RegistrySpec{
			Scheme:     l.Scheme,
			Host:       l.Host,
			SkipVerify: l.SkipTLSVerify,
			CA:         l.CA,
		},
	}
	if l.Username != "" && l.Password != "" {
		reg.RegistrySpec.RegistryAuth = &v1.RegistryAuth{
			Username: l.Username,
			Password: l.Password,
		}
	}
	return reg
}
