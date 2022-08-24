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

	"github.com/spf13/cobra"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

func NewKiProxyCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:           "kubeclipper-proxy",
		Short:         "kubeclipper-proxy: kubeclipper proxy server",
		Long:          "TODO: Add long description for kubeclipper-proxy",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	stopCh := genericapiserver.SetupSignalHandler()

	cmds.ResetFlags()
	cmds.AddCommand(newCmdVersion(out))
	cmds.AddCommand(newServeCommand(stopCh))

	return cmds
}
