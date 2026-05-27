package operation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// RetryOptions holds the options for the operation retry subcommand.
type RetryOptions struct {
	BaseOptions
	OperationID string
}

// NewRetryOptions creates a RetryOptions with default values.
func NewRetryOptions(streams options.IOStreams) *RetryOptions {
	return &RetryOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

const retryLong = `
  Retry a failed operation.

  Only operations in "failed" status can be retried.`

const retryExample = `
  # Retry a failed operation
  kcctl operation retry <OPERATION_ID>`

// NewCmdRetry creates the operation retry subcommand.
func NewCmdRetry(streams options.IOStreams) *cobra.Command {
	o := NewRetryOptions(streams)
	cmd := &cobra.Command{
		Use:                   "retry OPERATION_ID",
		DisableFlagsInUseLine: true,
		Short:                 "Retry a failed operation",
		Long:                  retryLong,
		Example:               retryExample,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.OperationID = args[0]
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunRetry())
		},
	}
	o.CliOpts.AddFlags(cmd.Flags())
	return cmd
}

// Complete initializes the API client from the CLI options.
func (o *RetryOptions) Complete(opts *options.CliOptions) error {
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

// RunRetry checks operation status and retries if it is failed.
func (o *RetryOptions) RunRetry() error {
	ctx := context.TODO()
	op, err := o.Client.DescribeOperation(ctx, o.OperationID)
	if err != nil {
		return err
	}

	if op.Status.Status != v1.OperationStatusFailed {
		return fmt.Errorf("operation is not in failed status, current status: %s", op.Status.Status)
	}

	if err := o.Client.RetryOperation(ctx, o.OperationID); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "Operation %s retry requested\n", o.OperationID)
	return nil
}
