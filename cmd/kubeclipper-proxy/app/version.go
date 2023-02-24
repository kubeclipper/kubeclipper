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
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/component-base/version"
	"sigs.k8s.io/yaml"
)

// newCmdVersion provides the version information of kubeclipper-server.
func newCmdVersion(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of kubeclipper-server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersion(out, cmd)
		},
		Args: cobra.NoArgs,
	}
	cmd.Flags().StringP("output", "o", "", "Output format; available options are 'yaml', 'json' and 'short'")
	return cmd
}

// RunVersion provides the version information of kubeclipper-server in format depending on arguments
// specified in cobra.Command.
func RunVersion(out io.Writer, cmd *cobra.Command) error {
	// logger.Debugf("[version] retrieving version info")
	v := version.Get()

	const flag = "output"
	of, err := cmd.Flags().GetString(flag)
	if err != nil {
		return errors.Wrapf(err, "error accessing flag %s for command %s", flag, cmd.Name())
	}

	switch of {
	case "":
		fmt.Fprintf(out, "ki agent version: %#v\n", v)
	case "short":
		fmt.Fprintf(out, "%s\n", v.GitVersion)
	case "yaml":
		y, err := yaml.Marshal(&v)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	case "json":
		y, err := json.MarshalIndent(&v, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(y))
	default:
		return errors.Errorf("invalid output format: %s", of)
	}

	return nil
}
