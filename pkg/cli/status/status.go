/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	defaultTimeout       = 10 * time.Second
	componentColumnWidth = 14
	statusColumnWidth    = 12
	labelWidth           = 8
)

type ExitError struct {
	code int
	err  error
}

func (e *ExitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *ExitError) Unwrap() error { return e.err }
func (e *ExitError) ExitCode() int { return e.code }

type Options struct {
	options.IOStreams
	cliOptions *options.CliOptions
	client     *kc.Client
	output     string
	timeout    time.Duration
}

func NewCmdStatus(streams options.IOStreams) *cobra.Command {
	o := &Options{
		IOStreams:  streams,
		cliOptions: options.NewCliOptions(),
		output:     "table",
		timeout:    defaultTimeout,
	}
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Show KubeClipper platform status",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Root().SilenceErrors = true
			if err := o.complete(); err != nil {
				return &ExitError{code: 2, err: err}
			}
			if err := o.validate(); err != nil {
				return &ExitError{code: 2, err: err}
			}
			return o.run(cmd.Context())
		},
	}
	o.cliOptions.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.output, "output", "o", o.output, "Output format: table, json, or yaml")
	cmd.Flags().DurationVar(&o.timeout, "timeout", o.timeout, "Maximum time to wait for platform status")
	return cmd
}

func (o *Options) complete() error {
	if err := o.cliOptions.Complete(); err != nil {
		return err
	}
	client, err := kc.FromConfigWithoutValidation(o.cliOptions.ToRawConfig())
	if err != nil {
		return err
	}
	o.client = client
	return nil
}

func (o *Options) validate() error {
	switch o.output {
	case "table", "json", "yaml":
	default:
		return fmt.Errorf("unsupported output format %q", o.output)
	}
	if o.timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	return nil
}

func (o *Options) run(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, o.timeout)
	defer cancel()
	result, err := o.client.PlatformStatus(ctx)
	if err != nil {
		return &ExitError{code: 2, err: fmt.Errorf("unable to get platform status: %w", err)}
	}
	if err := printStatus(o.Out, result, o.output); err != nil {
		return &ExitError{code: 2, err: err}
	}
	if result.Status != platformstatus.Healthy {
		return &ExitError{code: 1}
	}
	return nil
}

func printStatus(out io.Writer, result *platformstatus.PlatformStatus, output string) error {
	switch output {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	case "yaml":
		data, err := yaml.Marshal(result)
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	default:
		return printTable(out, result)
	}
}

func printTable(out io.Writer, result *platformstatus.PlatformStatus) error {
	style := newStatusOutputStyle(out)
	if style.enabled {
		return printTerminalTable(out, result, style)
	}
	return printPlainTable(out, result)
}

func printPlainTable(out io.Writer, result *platformstatus.PlatformStatus) error {
	if _, err := fmt.Fprintf(out, "KubeClipper Platform Status: %s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Checked At: %s\n\n", result.CheckedAt.Local().Format(time.RFC3339)); err != nil {
		return err
	}
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"COMPONENT", "STATUS", "MESSAGE"})
	for _, component := range result.Components {
		table.Append([]string{component.Name, string(component.Status), component.Message})
	}
	table.Render()
	return nil
}

func printTerminalTable(out io.Writer, result *platformstatus.PlatformStatus, style statusOutputStyle) error {
	if _, err := fmt.Fprintln(out, style.title("KubeClipper Platform Status")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "\n%s  %s\n",
		style.label("Overall"), style.status(result.Status, statusMarker(result.Status)+" "+string(result.Status))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%s  %s\n\n",
		style.label("Checked"), result.CheckedAt.Local().Format(time.RFC3339)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%s  %s  %s\n",
		style.heading(padRight("COMPONENT", componentColumnWidth)),
		style.heading(padRight("STATUS", statusColumnWidth)),
		style.heading("DETAILS")); err != nil {
		return err
	}
	for _, component := range result.Components {
		status := statusMarker(component.Status) + " " + string(component.Status)
		if _, err := fmt.Fprintf(out, "%s  %s  %s\n",
			padRight(component.Name, componentColumnWidth),
			style.status(component.Status, padRight(status, statusColumnWidth)),
			component.Message); err != nil {
			return err
		}
	}
	return nil
}

func statusMarker(status platformstatus.Status) string {
	switch status {
	case platformstatus.Healthy:
		return "✓"
	case platformstatus.Degraded:
		return "!"
	case platformstatus.Unhealthy:
		return "✗"
	default:
		return "?"
	}
}

func padRight(value string, width int) string {
	displayWidth := utf8.RuneCountInString(value)
	if displayWidth >= width {
		return value
	}
	return value + strings.Repeat(" ", width-displayWidth)
}

type statusOutputStyle struct {
	enabled bool
}

func newStatusOutputStyle(out io.Writer) statusOutputStyle {
	file, terminal := out.(*os.File)
	_, noColor := os.LookupEnv("NO_COLOR")
	return statusOutputStyle{
		enabled: terminal && term.IsTerminal(int(file.Fd())) && !noColor && os.Getenv("TERM") != "dumb",
	}
}

func (s statusOutputStyle) title(value string) string {
	return s.wrap("1;36", value)
}

func (s statusOutputStyle) heading(value string) string {
	return s.wrap("2", value)
}

func (s statusOutputStyle) label(value string) string {
	return s.wrap("1", padRight(value, labelWidth))
}

func (s statusOutputStyle) status(status platformstatus.Status, value string) string {
	code := "35"
	switch status {
	case platformstatus.Healthy:
		code = "1;32"
	case platformstatus.Degraded:
		code = "1;33"
	case platformstatus.Unhealthy:
		code = "1;31"
	case platformstatus.Skipped:
		code = "2"
	}
	return s.wrap(code, value)
}

func (s statusOutputStyle) wrap(code, value string) string {
	if !s.enabled {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}
