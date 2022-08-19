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
	"github.com/spf13/cobra"

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
  # Using config file to create resource
  kcctl create -f 'FILE-PATH'`
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
		Short:                 "create kubeclipper resource",
		Long:                  longDescription,
		Example:               createExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.CliOpts))
			// utils.CheckErr(o.ValidateArgs(cmd, args))
			// utils.CheckErr(o.RunGet())
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
	c, err := opts.ToRawConfig().ToKcClient()
	if err != nil {
		return err
	}
	l.Client = c
	return nil
}
