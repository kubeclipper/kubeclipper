package operation

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// DescribeOptions holds the options for the operation describe subcommand.
type DescribeOptions struct {
	BaseOptions
	OperationID string
}

// NewDescribeOptions creates a DescribeOptions with default values.
func NewDescribeOptions(streams options.IOStreams) *DescribeOptions {
	return &DescribeOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

const describeLong = `
  Show detailed information about an operation, including per-step status
  and per-node status with duration.`

const describeExample = `
  # Describe an operation
  kcctl operation describe <OPERATION_ID>`

// NewCmdDescribe creates the operation describe subcommand.
func NewCmdDescribe(streams options.IOStreams) *cobra.Command {
	o := NewDescribeOptions(streams)
	cmd := &cobra.Command{
		Use:                   "describe OPERATION_ID",
		DisableFlagsInUseLine: true,
		Short:                 "Show details of an operation",
		Long:                  describeLong,
		Example:               describeExample,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.OperationID = args[0]
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunDescribe())
		},
	}
	o.CliOpts.AddFlags(cmd.Flags())
	return cmd
}

// Complete initializes the API client from the CLI options.
func (o *DescribeOptions) Complete(opts *options.CliOptions) error {
	if err := opts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(opts.ToRawConfig())
	if err != nil {
		return err
	}
	o.Client = c
	return nil
}

// RunDescribe fetches and renders operation details.
func (o *DescribeOptions) RunDescribe() error {
	ctx := context.TODO()
	op, err := o.Client.DescribeOperation(ctx, o.OperationID)
	if err != nil {
		return fmt.Errorf("operation %s not found: %w", o.OperationID, err)
	}
	renderOperation(o.Out, op)
	return nil
}

// renderOperation writes a human-readable description of an operation.
func renderOperation(w io.Writer, op *v1.Operation) {
	clusterName := op.Labels[common.LabelClusterName]
	sponsor := op.Labels[common.LabelOperationSponsor]
	action := op.Labels[common.LabelOperationAction]

	// Header
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  Name:       %s\n", op.Name)
	fmt.Fprintf(w, "  Action:     %s\n", action)
	fmt.Fprintf(w, "  Cluster:    %s\n", clusterName)
	fmt.Fprintf(w, "  Sponsor:    %s\n", sponsor)
	fmt.Fprintf(w, "  Status:     %s\n", statusColor(string(op.Status.Status)).Sprint(string(op.Status.Status)))
	fmt.Fprintf(w, "  Created:    %s\n", op.CreationTimestamp.Format(timeFormat))
	fmt.Fprintf(w, "\n")

	// Per-step breakdown
	for _, step := range op.Steps {
		startTime := getStepStartTime(op, step.ID)
		stepStatus := getStepStatus(op, step.ID)

		timeStr := ""
		if !startTime.IsZero() {
			timeStr = startTime.Format(timeFormat)
		}

		fmt.Fprintf(w, "  %s Step: %s (%s) [%s] %s\n",
			color.HiBlueString("─────"),
			step.Name,
			step.ID,
			statusColor(stepStatus).Sprint(stepStatus),
			color.HiBlueString("─────"),
		)
		if timeStr != "" {
			fmt.Fprintf(w, "    Started: %s\n", timeStr)
		}

		// Per-node under each step
		for _, node := range step.Nodes {
			nodeStatus, startAt, endAt := getNodeStatusWithTime(op, step.ID, node.ID)
			duration := calculateDuration(startAt, endAt)

			durationText := ""
			if duration > 0 {
				durationText = fmt.Sprintf(" [%s]", duration.Round(time.Second).String())
			}

			fmt.Fprintf(w, "    %s %s (%s) %s%s\n",
				statusColor(nodeStatus).Sprint(string(nodeStatus)),
				node.Hostname,
				node.IPv4,
				node.ID,
				durationText,
			)

			// Show error details for failed nodes
			if nodeStatus == string(v1.StepStatusFailed) {
				reason, message := getStepNodeErrorDetail(op, step.ID, node.ID)
				if reason != "" {
					fmt.Fprintf(w, "      Reason:  %s\n", color.RedString(reason))
				}
				if message != "" {
					fmt.Fprintf(w, "      Message: %s\n", color.RedString(message))
				}
			}
		}
		fmt.Fprintf(w, "\n")
	}
}

// getStepStatus returns the overall status for a step based on its node statuses.
func getStepStatus(op *v1.Operation, stepID string) string {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Status == v1.StepStatusFailed {
					return string(v1.StepStatusFailed)
				}
			}
			for _, st := range cond.Status {
				if st.Status == "" {
					return "pending"
				}
			}
			return string(v1.StepStatusSuccessful)
		}
	}
	return "pending"
}

// getStepNodeErrorDetail returns reason and message for a failed node.
func getStepNodeErrorDetail(op *v1.Operation, stepID, nodeID string) (reason string, message string) {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Node == nodeID {
					return st.Reason, st.Message
				}
			}
		}
	}
	return "", ""
}

// statusColor returns a color func based on the status string.
func statusColor(status string) *color.Color {
	switch status {
	case "successful":
		return color.New(color.FgGreen, color.Bold)
	case "failed", "error":
		return color.New(color.FgRed, color.Bold)
	case "running":
		return color.New(color.FgYellow, color.Bold)
	case "termination":
		return color.New(color.FgMagenta, color.Bold)
	case "pending", "unknown":
		return color.New(color.FgCyan)
	default:
		return color.New(color.FgWhite)
	}
}
