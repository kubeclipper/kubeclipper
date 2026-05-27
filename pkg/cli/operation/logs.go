package operation

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"golang.org/x/term"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/operation/tui"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// LogsOptions holds the options for the operation logs subcommand.
type LogsOptions struct {
	BaseOptions
	OperationID string
	Follow      bool
	MaxLength   int
	Cluster     string
	Summary     bool
}

// NewLogsOptions creates a LogsOptions with default values.
func NewLogsOptions(streams options.IOStreams) *LogsOptions {
	return &LogsOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
		MaxLength: defaultMaxLength,
	}
}

const logsLong = `
  Show plain-text logs of an operation, grouped by step and node.

  When --cluster is provided, an interactive TUI log viewer is launched
  showing operations for that cluster. Use the operation ID directly
  for plain-text logs.`

const logsExample = `
  # Show logs of an operation
  kcctl operation logs <OPERATION_ID>

  # Follow log output in real time
  kcctl operation logs <OPERATION_ID> --follow

  # Show only step/node status, not log content
  kcctl operation logs <OPERATION_ID> --summary

  # Set max log length to 100 characters per entry
  kcctl operation logs <OPERATION_ID> --max-length 100

  # TUI mode for cluster-wide operation logs
  kcctl operation logs --cluster my-cluster`

// NewCmdLogs creates the operation logs subcommand.
func NewCmdLogs(streams options.IOStreams) *cobra.Command {
	o := NewLogsOptions(streams)
	cmd := &cobra.Command{
		Use:                   "logs [OPERATION_ID] [--cluster CLUSTER] [--follow] [--summary] [--max-length N]",
		DisableFlagsInUseLine: true,
		Short:                 "Show logs of an operation",
		Long:                  logsLong,
		Example:               logsExample,
		Args:                  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				o.OperationID = args[0]
			}
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.Validate())
			utils.CheckErr(o.Run())
		},
	}
	cmd.Flags().BoolVarP(&o.Follow, "follow", "f", false, "Follow log output in real time")
	cmd.Flags().IntVarP(&o.MaxLength, "max-length", "m", defaultMaxLength, "Max log length to display per entry (0 for unlimited)")
	cmd.Flags().StringVarP(&o.Cluster, "cluster", "c", "", "Cluster name (launches TUI log viewer)")
	cmd.Flags().BoolVarP(&o.Summary, "summary", "s", false, "Show only step/node status, not log content")
	o.CliOpts.AddFlags(cmd.Flags())
	return cmd
}

// Validate checks that the options are valid.
func (o *LogsOptions) Validate() error {
	if o.Cluster != "" && o.OperationID != "" {
		return fmt.Errorf("cannot specify both OPERATION_ID and --cluster")
	}
	if o.Cluster == "" && o.OperationID == "" {
		return fmt.Errorf("either OPERATION_ID or --cluster must be provided")
	}
	return nil
}

// Complete initializes the API client from the CLI options.
func (o *LogsOptions) Complete(opts *options.CliOptions) error {
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

// Run executes the logs command.
func (o *LogsOptions) Run() error {
	// Cluster mode: launch TUI or show non-TTY fallback
	if o.Cluster != "" {
		return o.runClusterMode()
	}

	// Single operation mode: plain-text output
	return o.runSingleOperation()
}

// runClusterMode handles the --cluster flag by launching TUI or falling back
// to a non-TTY listing.
func (o *LogsOptions) runClusterMode() error {
	if !isTTY() {
		return o.nonTTYClusterFallback()
	}
	return tui.RunTUI(o.Client, o.Cluster)
}

// nonTTYClusterFallback lists operations for the cluster when no TTY is available.
func (o *LogsOptions) nonTTYClusterFallback() error {
	ctx := context.Background()
	labelSelector := fmt.Sprintf("kubeclipper.io/cluster=%s", o.Cluster)
	list, err := o.Client.ListOperation(ctx, kc.Queries{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list operations for cluster %s: %w", o.Cluster, err)
	}

	if len(list.Items) == 0 {
		return fmt.Errorf("no operations found for cluster %s", o.Cluster)
	}

	fmt.Fprintln(o.Out, "TUI requires an interactive terminal. Use 'kcctl operation logs <ID>' instead.")
	fmt.Fprintln(o.Out, "Operations:")
	for _, op := range list.Items {
		action := op.Labels["kubeclipper.io/operation"]
		if action == "" {
			action = op.Name
		}
		fmt.Fprintf(o.Out, "  %s (%s)\n", op.Name, action)
	}
	os.Exit(1)
	return nil
}

// runSingleOperation handles the plain-text log output for a single operation ID.
func (o *LogsOptions) runSingleOperation() error {
	ctx := context.TODO()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Track offsets for incremental reading when following
	offsets := make(map[string]int64)

	var lastStepCount int
	for {
		operation, err := o.Client.DescribeOperation(ctx, o.OperationID)
		if err != nil {
			printErrorLine(o.Out, "Failed to describe operation %s: %v\n", o.OperationID, err)
			return nil
		}
		steps := operation.Steps
		if len(steps) == 0 {
			printErrorLine(o.Out, "No steps found in operation %s\n", o.OperationID)
			return nil
		}

		for i := lastStepCount; i < len(steps); i++ {
			step := steps[i]

			// Get step start time and status
			startTime := getStepStartTime(operation, step.ID)
			stepStatus := getStepStatus(operation, step.ID)

			// Display step header with separator
			printStepTitle(o.Out, step.Name, step.ID, stepStatus, startTime)

			for _, node := range step.Nodes {
				// Get node status and timing
				nodeStatus, startAt, endAt := getNodeStatusWithTime(operation, step.ID, node.ID)
				duration := calculateDuration(startAt, endAt)

				// Display node status
				printNodeStatus(o.Out, node.Hostname, node.ID, nodeStatus, node.IPv4, duration)

				if o.Summary {
					continue
				}

				// Fetch and display logs
				key := step.ID + "|" + node.ID
				offset := offsets[key]

				stepLog, err := o.Client.GetStepNodeLog(ctx, o.OperationID, step.ID, node.ID, offset)
				if err != nil {
					printErrorLine(o.ErrOut, "[step:%s][node:%s] log fetch error: %v\n", step.ID, node.ID, err)
					continue
				}
				if stepLog.Content != "" {
					for _, line := range splitLines(truncateLog(stepLog.Content, o.MaxLength)) {
						printLogLine(o.Out, line)
					}
					offsets[key] = offset + int64(len(stepLog.Content))
				}
			}
		}
		lastStepCount = len(steps)

		if !o.Follow {
			break
		}

		// Check if the operation has reached a terminal state
		if operation.Status.Status == v1.OperationStatusSuccessful ||
			operation.Status.Status == v1.OperationStatusFailed ||
			operation.Status.Status == v1.OperationStatusTermination {
			break
		}

		select {
		case <-interrupt:
			return nil
		case <-time.After(pollInterval):
		}
	}
	return nil
}

// isTTY checks if stdout is connected to an interactive terminal.
func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// printStepTitle prints a step header with a separator line, step name, ID, and color-coded status.
func printStepTitle(w io.Writer, name, id, status string, startTime metav1.Time) {
	timeStr := ""
	if !startTime.IsZero() {
		timeStr = startTime.Format(timeFormat)
	}

	statusText := statusColor(status).Sprintf("[%s]", status)

	var title string
	if timeStr != "" {
		title = fmt.Sprintf(" Step: %s (%s) %s %s ", name, id, statusText, timeStr)
	} else {
		title = fmt.Sprintf(" Step: %s (%s) %s ", name, id, statusText)
	}

	separator := "─────"
	padLen := 60 - len(title)
	if padLen < 0 {
		padLen = 0
	}
	line := separator + title + separator + repeatStr("─", padLen)

	fmt.Fprintf(w, "\n%s\n", color.HiBlueString(line))
}

// repeatStr repeats a string n times.
func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// printNodeStatus prints a node line with colored status and duration.
func printNodeStatus(w io.Writer, hostname, nodeID, status, ip string, duration time.Duration) {
	durationText := ""
	if duration > 0 {
		durationText = fmt.Sprintf(" [%s]", duration.Round(time.Second).String())
	}

	fmt.Fprintf(w, "  %s %s (%s) %s%s\n",
		statusColor(status).Sprint(status),
		hostname,
		ip,
		nodeID,
		durationText,
	)
}

// printLogLine prints a log line with 4-space indent in dark gray.
func printLogLine(w io.Writer, line string) {
	fmt.Fprintf(w, "%s\n", color.New(color.FgHiBlack).Sprintf("    %s", line))
}

// printErrorLine prints an error message in red.
func printErrorLine(w io.Writer, format string, a ...interface{}) {
	redMsg := fmt.Sprintf(format, a...)
	fmt.Fprint(w, color.RedString(redMsg))
}
