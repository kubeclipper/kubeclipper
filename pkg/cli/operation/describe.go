package operation

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
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
	tasks, err := o.Client.ListOperationTasks(ctx, string(op.UID))
	if err != nil {
		return fmt.Errorf("list tasks for operation %s: %w", o.OperationID, err)
	}
	renderOperation(o.Out, op, tasks.Items)
	return nil
}

// renderOperation writes a human-readable description of an operation.
func renderOperation(w io.Writer, op *operationsv1alpha1.Operation, tasks []operationsv1alpha1.OperationTask) {
	grouped := tasksByExecution(tasks)

	// Header
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  Name:       %s\n", op.Name)
	fmt.Fprintf(w, "  Action:     %s\n", op.Spec.Action)
	fmt.Fprintf(w, "  Cluster:    %s\n", op.Spec.TargetRef.Name)
	fmt.Fprintf(w, "  Status:     %s\n", statusColor(string(op.Status.Phase)).Sprint(string(op.Status.Phase)))
	fmt.Fprintf(w, "  Created:    %s\n", op.CreationTimestamp.Format(timeFormat))
	fmt.Fprintf(w, "\n")

	// Per-step breakdown
	for stepIndex := range op.Spec.Steps {
		step := &op.Spec.Steps[stepIndex]
		startTime := stepStartTime(step.ID, tasks)
		currentStepStatus := stepStatus(step, grouped, op.Status.Phase)

		timeStr := ""
		if startTime != nil {
			timeStr = startTime.Format(timeFormat)
		}

		fmt.Fprintf(w, "  %s Step: %s [%s] %s\n",
			color.HiBlueString("─────"),
			step.ID,
			statusColor(currentStepStatus).Sprint(currentStepStatus),
			color.HiBlueString("─────"),
		)
		if timeStr != "" {
			fmt.Fprintf(w, "    Started: %s\n", timeStr)
		}

		// Per-node under each step
		for _, node := range step.Targets {
			task := effectiveTask(grouped, step.ID, node.UID)
			nodeStatus := missingTaskPhase(op.Status.Phase)
			var startAt, endAt *metav1.Time
			if task != nil {
				nodeStatus, startAt, endAt = string(task.Status.Phase), task.Status.StartedAt, task.Status.FinishedAt
			}
			duration := calculateDuration(startAt, endAt)

			durationText := ""
			if duration > 0 {
				durationText = fmt.Sprintf(" [%s]", duration.Round(time.Second).String())
			}

			fmt.Fprintf(w, "    %s %s (%s)%s\n",
				statusColor(nodeStatus).Sprint(string(nodeStatus)),
				node.Name,
				node.UID,
				durationText,
			)

			// Show error details for failed nodes
			if task != nil && task.Status.Result != nil && task.Status.Result.Message != "" {
				fmt.Fprintf(w, "      Message: %s\n", color.RedString(task.Status.Result.Message))
			}
		}
		fmt.Fprintf(w, "\n")
	}
}

// statusColor returns a color func based on the status string.
func statusColor(status string) *color.Color {
	switch status {
	case "Succeeded":
		return color.New(color.FgGreen, color.Bold)
	case "Failed", "TimedOut", "error":
		return color.New(color.FgRed, color.Bold)
	case "Running":
		return color.New(color.FgYellow, color.Bold)
	case "Canceled":
		return color.New(color.FgMagenta, color.Bold)
	case "Pending", "unknown":
		return color.New(color.FgCyan)
	default:
		return color.New(color.FgWhite)
	}
}
