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
	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
)

const (
	setLongDescription = `
	Set specific fields on kubeclipper resources.
`
	setExample = `
	# Set cluster external access configuration
	kcctl set cluster my-cluster --external-ip=1.2.3.4
`
)

func NewCmdSet(streams options.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "set (SUBCOMMAND) [options]",
		DisableFlagsInUseLine: true,
		Short:                 "Set specific fields on resources",
		Long:                  setLongDescription,
		Example:               setExample,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(NewCmdSetCluster(streams))
	return cmd
}
