package logs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TimeFormat is the standard time format used in log output
	TimeFormat = "2006-01-02 15:04:05"

	// MinDuration is the minimum duration to show (1s)
	MinDuration = time.Second

	// PollInterval is how often to poll for updates when following logs
	PollInterval = 2 * time.Second

	// DefaultMaxLength is the default max log length to display
	DefaultMaxLength = 200

	// TruncatedSuffix is added to truncated logs
	TruncatedSuffix = "... (truncated)"
)

// LogsOptions contains the command line options for logs command
type LogsOptions struct {
	Follow    bool
	Operator  string
	MaxLength int
	Summary   bool
	options.IOStreams
	cliOpts *options.CliOptions
}

// NewLogsOptions creates a new LogsOptions with default values
func NewLogsOptions(streams options.IOStreams) *LogsOptions {
	return &LogsOptions{
		IOStreams: streams,
		cliOpts:   options.NewCliOptions(),
		MaxLength: DefaultMaxLength,
	}
}

// logs command long description
const logsLong = `
  Show detailed logs of a platform operation (e.g., cluster install, node change), grouped by step and node.
  Supports real-time follow (--follow), log truncation (--max-length), and status-only view (--summary).

  Examples:
    kcctl logs <OPERATION_ID>
    kcctl logs --follow <OPERATION_ID>
    kcctl logs --summary <OPERATION_ID>
    kcctl logs --max-length 100 <OPERATION_ID>
`

// logs command examples
const logsExample = `
  # Show all logs of an operation
  kcctl logs <OPERATION_ID>

  # Follow log output in real time
  kcctl logs <OPERATION_ID> --follow

  # Show only step/node status, not log content
  kcctl logs <OPERATION_ID> --summary

  # Set max log length to 100 characters per entry
  kcctl logs <OPERATION_ID> --max-length 100 
`

// NewCmdLogs creates a new logs command
func NewCmdLogs(streams options.IOStreams) *cobra.Command {
	o := NewLogsOptions(streams)
	cmd := &cobra.Command{
		Use:     "logs [--follow] [--max-length N] [--summary] OPERATION_ID",
		Short:   "Show logs of an operation, grouped by step and node.",
		Long:    logsLong,
		Example: logsExample,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.Operator = args[0]
			utils.CheckErr(o.cliOpts.Complete())
			utils.CheckErr(o.Run())
		},
	}
	cmd.Flags().BoolVarP(&o.Follow, "follow", "f", false, "Follow log output in real time")
	cmd.Flags().IntVarP(&o.MaxLength, "max-length", "m", DefaultMaxLength, "Max log length to display per entry (0 for unlimited)")
	cmd.Flags().BoolVarP(&o.Summary, "summary", "s", false, "Show only step/node status, not log content")
	return cmd
}

// Run executes the logs command
func (o *LogsOptions) Run() error {
	c, err := kc.FromConfig(o.cliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	ctx := context.TODO()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	var lastStepCount int
	for {
		operation, err := c.DescribeOperation(ctx, o.Operator)
		if err != nil {
			PrintErrorLine(o.Out, "No steps found in operation %s\n", o.Operator)
			return nil
		}
		steps := operation.Steps
		if len(steps) == 0 {
			PrintErrorLine(o.Out, "No steps found in operation %s\n", o.Operator)
			return nil
		}

		for i := lastStepCount; i < len(steps); i++ {
			step := steps[i]
			stepStatus := string(operation.Status.Status)

			// Get step start time
			startTime := getStepStartTime(operation, step.ID)

			// Display step title with time at the beginning
			PrintStepTitle(o.Out, step.Name, step.ID, stepStatus, startTime)

			for _, node := range step.Nodes {
				// Get detailed node status, start and end time
				nodeStatus, startAt, endAt := getNodeStatusWithTime(operation, step.ID, node.ID)

				// Calculate duration with minimum threshold
				duration := calculateDuration(startAt, endAt)

				// Display node status with IP and duration
				PrintNodeStatus(o.Out, node.Hostname, node.ID, nodeStatus, node.IPv4, duration)

				if o.Summary {
					continue
				}
				log, err := c.GetStepNodeLog(ctx, o.Operator, step.ID, node.ID)
				if err != nil {
					PrintErrorLine(o.ErrOut, "[step:%s][node:%s] log fetch error: %v\n", step.ID, node.ID, err)
					continue
				}
				if log != "" {
					for _, line := range SplitLines(truncateLog(log, o.MaxLength)) {
						PrintLogLine(o.Out, line)
					}
				}
			}
		}
		lastStepCount = len(steps)

		if !o.Follow {
			break
		}
		select {
		case <-interrupt:
			return nil
		case <-time.After(PollInterval):
		}
	}
	return nil
}

// truncateLog truncates log content if it exceeds the maximum length
func truncateLog(log string, maxLen int) string {
	if maxLen <= 0 || len(log) <= maxLen {
		return log
	}
	return log[:maxLen] + TruncatedSuffix
}

// calculateDuration calculates the duration between two times with a minimum threshold
func calculateDuration(startAt, endAt metav1.Time) time.Duration {
	var duration time.Duration
	if !startAt.IsZero() && !endAt.IsZero() {
		duration = endAt.Time.Sub(startAt.Time)
		// Ensure duration is at least MinDuration when start and end times are close
		if duration < MinDuration {
			duration = MinDuration
		}
	}
	return duration
}

// getNodeStatus gets the status of a node within a step
func getNodeStatus(op *v1.Operation, stepID, nodeID string) string {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Node == nodeID {
					return string(st.Status)
				}
			}
		}
	}
	return "unknown"
}

// getNodeStatusWithTime returns status and time information for a node in a step
func getNodeStatusWithTime(op *v1.Operation, stepID, nodeID string) (status string, startAt metav1.Time, endAt metav1.Time) {
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if st.Node == nodeID {
					return string(st.Status), st.StartAt, st.EndAt
				}
			}
		}
	}
	return "unknown", metav1.Time{}, metav1.Time{}
}

// getStepStartTime returns the earliest start time from all nodes in a step
func getStepStartTime(op *v1.Operation, stepID string) metav1.Time {
	var startTime metav1.Time
	for _, cond := range op.Status.Conditions {
		if cond.StepID == stepID {
			for _, st := range cond.Status {
				if startTime.IsZero() || st.StartAt.Before(&startTime) {
					startTime = st.StartAt
				}
			}
		}
	}
	return startTime
}

// PrintStepTitle prints the step title with start time at the beginning
func PrintStepTitle(w io.Writer, name, id, status string, startTime metav1.Time) {
	timeStr := ""
	if !startTime.IsZero() {
		timeStr = startTime.Format(TimeFormat)
	}

	var title string
	if timeStr != "" {
		title = fmt.Sprintf("===== Step: [%s] %s (%s) [%s]", timeStr, name, id, status)
	} else {
		title = fmt.Sprintf("===== Step: %s (%s) [%s]", name, id, status)
	}

	fmt.Fprintf(w, "\n%s\n", color.New(color.FgHiBlue, color.Bold).Sprint(title))
}

// PrintNodeStatus prints the node status with color, IP, and duration
func PrintNodeStatus(w io.Writer, hostname, nodeID, status, ip string, duration time.Duration) {
	var statusColor *color.Color
	switch status {
	case "successful":
		statusColor = color.New(color.FgGreen, color.Bold)
	case "failed", "error":
		statusColor = color.New(color.FgRed, color.Bold)
	case "running":
		statusColor = color.New(color.FgYellow, color.Bold)
	default:
		statusColor = color.New(color.FgWhite)
	}

	// Build node display info with IP and duration
	nodeInfo := fmt.Sprintf("%s (%s) %s", hostname, ip, nodeID)
	statusText := statusColor.Sprintf("%s", status)
	durationText := ""
	if duration > 0 {
		durationText = fmt.Sprintf(" [%s]", duration.Round(time.Second).String())
	}

	fmt.Fprintf(w, "  Node: %s [%s]%s\n", nodeInfo, statusText, durationText)
}

// PrintLogLine prints a log line in light gray with indent
func PrintLogLine(w io.Writer, line string) {
	fmt.Fprintf(w, "%s\n", color.New(color.FgHiBlack).Sprintf("    %s", line))
}

// PrintErrorLine prints error message in red
func PrintErrorLine(w io.Writer, format string, a ...interface{}) {
	fmt.Fprintf(w, color.RedString(format, a...))
}

// SplitLines splits a string into lines
func SplitLines(s string) []string {
	return colorizeSplitLines(s)
}

// colorizeSplitLines splits string by line, compatible with Windows/Linux line endings
func colorizeSplitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			end := i
			// handle \r\n
			if end > start && s[end-1] == '\r' {
				end--
			}
			lines = append(lines, s[start:end])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
