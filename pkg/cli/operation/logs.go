package operation

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/operation/tui"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
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
	return tui.RunTUI(o.Client, o.Cluster, o.In, o.Out)
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
	return fmt.Errorf("non-interactive terminal: use 'kcctl operation logs <ID>' instead")
}

// runSingleOperation handles the plain-text log output for a single operation ID.
func (o *LogsOptions) runSingleOperation() error {
	ctx := context.TODO()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	offsets := make(map[string]int64)
	seenTasks := make(map[string]bool)
	for {
		operation, err := o.Client.DescribeOperation(ctx, o.OperationID)
		if err != nil {
			printErrorLine(o.Out, "Failed to describe operation %s: %v\n", o.OperationID, err)
			return nil
		}
		steps := operation.Spec.Steps
		if len(steps) == 0 {
			printErrorLine(o.Out, "No steps found in operation %s\n", o.OperationID)
			return nil
		}

		taskList, err := o.Client.ListOperationTasks(ctx, string(operation.UID))
		if err != nil {
			return fmt.Errorf("list tasks for operation %s: %w", o.OperationID, err)
		}
		for _, step := range steps {
			for _, task := range tasksForStep(taskList.Items, step.ID) {
				if !seenTasks[task.Name] {
					printTaskTitle(o.Out, &task)
					seenTasks[task.Name] = true
				}
				if o.Summary {
					continue
				}
				offset := offsets[task.Name]
				taskLog, logErr := o.Client.GetOperationTaskLog(ctx, task.Name, offset)
				if logErr != nil {
					printErrorLine(o.ErrOut, "[task:%s] log fetch error: %v\n", task.Name, logErr)
					continue
				}
				if taskLog.Content != "" {
					for _, line := range splitLines(truncateLog(taskLog.Content, o.MaxLength)) {
						printLogLine(o.Out, line)
					}
					offsets[task.Name] = offset + taskLog.DeliverySize
				}
			}
		}

		if !o.Follow {
			break
		}

		// Check if the operation has reached a terminal state
		if operation.Status.Phase.IsTerminal() {
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

func tasksForStep(tasks []operationsv1alpha1.OperationTask, stepID string) []operationsv1alpha1.OperationTask {
	result := make([]operationsv1alpha1.OperationTask, 0)
	for i := range tasks {
		if tasks[i].Spec.StepID == stepID {
			result = append(result, tasks[i])
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Spec.RetryGeneration != result[j].Spec.RetryGeneration {
			return result[i].Spec.RetryGeneration < result[j].Spec.RetryGeneration
		}
		if result[i].Spec.Attempt != result[j].Spec.Attempt {
			return result[i].Spec.Attempt < result[j].Spec.Attempt
		}
		return result[i].Spec.NodeRef.Name < result[j].Spec.NodeRef.Name
	})
	return result
}

func printTaskTitle(w io.Writer, task *operationsv1alpha1.OperationTask) {
	timeStr := ""
	if task.Status.StartedAt != nil {
		timeStr = task.Status.StartedAt.Format(timeFormat)
	}
	status := string(task.Status.Phase)
	statusText := statusColor(status).Sprintf("[%s]", status)

	var title string
	if timeStr != "" {
		title = fmt.Sprintf(" Step: %s Node: %s Attempt: %d %s %s ", task.Spec.StepID, task.Spec.NodeRef.Name, task.Spec.Attempt, statusText, timeStr)
	} else {
		title = fmt.Sprintf(" Step: %s Node: %s Attempt: %d %s ", task.Spec.StepID, task.Spec.NodeRef.Name, task.Spec.Attempt, statusText)
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

// printLogLine prints a log line with 4-space indent in dark gray.
func printLogLine(w io.Writer, line string) {
	fmt.Fprintf(w, "%s\n", color.New(color.FgHiBlack).Sprintf("    %s", line))
}

// printErrorLine prints an error message in red.
func printErrorLine(w io.Writer, format string, a ...interface{}) {
	redMsg := fmt.Sprintf(format, a...)
	fmt.Fprint(w, color.RedString(redMsg))
}
