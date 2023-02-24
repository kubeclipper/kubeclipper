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

package version

import (
	"context"
	"encoding/json"
	"fmt"

	apimachineryversion "k8s.io/apimachinery/pkg/version"

	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"

	"github.com/spf13/cobra"
	"k8s.io/component-base/version"
	"sigs.k8s.io/yaml"
)

const (
	longDescription = `
  Print kcctl version information.`
	versionExample = `
  # Print version Information
  kcctl version -o yaml

  Please read 'kcctl version -h' get more version flags.`
)

type VersionOptions struct {
	options.IOStreams
	cliOpts *options.CliOptions
	client  *kc.Client
	output  string
}

func NewCmdVersion(streams options.IOStreams) *cobra.Command {
	v := &VersionOptions{
		IOStreams: streams,
		client:    nil,
		output:    "",
		cliOpts:   options.NewCliOptions(),
	}
	cmd := &cobra.Command{
		Use:     "version [flags]",
		Short:   "Print the version of kcctl",
		Long:    longDescription,
		Example: versionExample,
		Run: func(cmd *cobra.Command, args []string) {
			// return RunVersion(v.Out, cmd)
			utils.CheckErr(v.Complete(v.cliOpts))
			utils.CheckErr(v.RunVersion())
		},
	}
	v.cliOpts.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&v.output, "output", "o", "", "Output format; available options are 'yaml', 'json' and 'short'")

	_ = cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "short", "yaml"}, cobra.ShellCompDirectiveDefault
	})
	return cmd
}

func (v *VersionOptions) Complete(opts *options.CliOptions) error {
	// ignore error when get version
	if err := opts.Complete(); err != nil {
		return nil
	}
	c, err := kc.FromConfig(opts.ToRawConfig())
	if err != nil {
		return nil
	}
	v.client = c
	return nil
}

func (v *VersionOptions) RunVersion() error {
	clientVersion := version.Get()
	var (
		serverVersion *apimachineryversion.Info
		err           error
	)
	if v.client != nil {
		serverVersion, err = v.client.Version(context.TODO())
		if err != nil {
			_, _ = v.ErrOut.Write([]byte("get server version error\n"))
		}
	}
	switch v.output {
	case "short":
		_, _ = v.Out.Write([]byte("kcctl version:\n"))
		_, _ = fmt.Fprintf(v.Out, "%s\n", clientVersion.GitVersion)
		if serverVersion != nil {
			_, _ = v.Out.Write([]byte("kubeclipper-server version:\n"))
			_, _ = fmt.Fprintf(v.Out, "%s\n", serverVersion.GitVersion)
		}
	case "yaml":
		y, err := yaml.Marshal(clientVersion)
		if err != nil {
			return err
		}
		_, _ = v.Out.Write([]byte("kcctl version:\n"))
		_, _ = fmt.Fprintln(v.Out, string(y))
		if serverVersion != nil {
			s, err := yaml.Marshal(serverVersion)
			if err != nil {
				return err
			}
			_, _ = v.Out.Write([]byte("kubeclipper-server version:\n"))
			_, _ = fmt.Fprintln(v.Out, string(s))
		}
	case "json":
		y, err := json.MarshalIndent(clientVersion, "", "\t")
		if err != nil {
			return err
		}
		_, _ = v.Out.Write([]byte("kcctl version:\n"))
		_, _ = fmt.Fprintln(v.Out, string(y))
		if serverVersion != nil {
			s, err := json.MarshalIndent(serverVersion, "", "\t")
			if err != nil {
				return err
			}
			_, _ = v.Out.Write([]byte("kubeclipper-server version:\n"))
			_, _ = fmt.Fprintln(v.Out, string(s))
		}
	default:
		_, _ = fmt.Fprintf(v.Out, "kcctl version: %#v\n", clientVersion)
		if serverVersion != nil {
			_, _ = fmt.Fprintf(v.Out, "kubeclipper-server version: %#v\n", *serverVersion)
		}
	}
	return nil
}
