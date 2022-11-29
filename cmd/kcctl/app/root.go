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

package app

import (
	"io"

	"github.com/kubeclipper/kubeclipper/pkg/cli/cluster"

	"github.com/kubeclipper/kubeclipper/pkg/cli/upgrade"

	"github.com/kubeclipper/kubeclipper/pkg/cli/completion"
	"github.com/kubeclipper/kubeclipper/pkg/cli/proxy"

	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/resource"

	"github.com/kubeclipper/kubeclipper/pkg/cli/registry"

	"github.com/kubeclipper/kubeclipper/pkg/cli/drain"

	"github.com/kubeclipper/kubeclipper/pkg/cli/join"

	"github.com/kubeclipper/kubeclipper/pkg/cli/create"

	"github.com/kubeclipper/kubeclipper/pkg/cli/clean"

	"github.com/kubeclipper/kubeclipper/pkg/cli/delete"

	"github.com/kubeclipper/kubeclipper/pkg/cli/deploy"

	"github.com/kubeclipper/kubeclipper/pkg/cli/get"

	"github.com/kubeclipper/kubeclipper/pkg/cli/login"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/version"
)

const (
	longDescription = `
kcctl is Kubeclipper command line tool.

kcctl controls the Kubeclipper platform.`
)

func NewKubeClipperCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "kcctl",
		Short: "kcctl is kubeclipper command line tool",
		Long:  longDescription,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmds.ResetFlags()
	cmds.CompletionOptions.DisableDefaultCmd = true
	logger.AddFlags(cmds.PersistentFlags())
	cmds.PersistentFlags().BoolVarP(&options.AssumeYes, "assumeyes", "y", false, "Assume yes; assume that the answer to any question which would be asked is yes.")

	ioStreams := options.IOStreams{
		In:     in,
		Out:    out,
		ErrOut: err,
	}
	cmds.AddCommand(deploy.NewCmdDeploy(ioStreams))
	cmds.AddCommand(clean.NewCmdClean(ioStreams))
	cmds.AddCommand(login.NewCmdLogin(ioStreams))
	cmds.AddCommand(get.NewCmdGet(ioStreams))
	cmds.AddCommand(create.NewCmdCreate(ioStreams))
	cmds.AddCommand(delete.NewCmdDelete(ioStreams))
	cmds.AddCommand(version.NewCmdVersion(ioStreams))
	cmds.AddCommand(join.NewCmdJoin(ioStreams))
	cmds.AddCommand(drain.NewCmdDrain(ioStreams))
	cmds.AddCommand(registry.NewCmdRegistry(ioStreams))
	cmds.AddCommand(resource.NewCmdResource(ioStreams))
	cmds.AddCommand(completion.NewCmdCompletion(ioStreams.Out))
	cmds.AddCommand(proxy.NewCmdProxy(ioStreams))
	cmds.AddCommand(upgrade.NewCmdUpgrade(ioStreams))
	cmds.AddCommand(cluster.NewCmdCluster(ioStreams))

	return cmds
}
