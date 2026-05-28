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

// TerminateOptions holds the options for the operation terminate subcommand.
type TerminateOptions struct {
	BaseOptions
	OperationID string
}

// NewTerminateOptions creates a TerminateOptions with default values.
func NewTerminateOptions(streams options.IOStreams) *TerminateOptions {
	return &TerminateOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

const terminateLong = `
  Terminate a running operation.

  Only operations in "running" status can be terminated.`

const terminateExample = `
  # Terminate a running operation
  kcctl operation terminate <OPERATION_ID>`

// NewCmdTerminate creates the operation terminate subcommand.
func NewCmdTerminate(streams options.IOStreams) *cobra.Command {
	o := NewTerminateOptions(streams)
	cmd := &cobra.Command{
		Use:                   "terminate OPERATION_ID",
		DisableFlagsInUseLine: true,
		Short:                 "Terminate a running operation",
		Long:                  terminateLong,
		Example:               terminateExample,
		Args:                  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.OperationID = args[0]
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunTerminate())
		},
	}
	o.CliOpts.AddFlags(cmd.Flags())
	return cmd
}

// Complete initializes the API client from the CLI options.
func (o *TerminateOptions) Complete(opts *options.CliOptions) error {
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

// RunTerminate checks operation status and terminates if it is running.
func (o *TerminateOptions) RunTerminate() error {
	ctx := context.TODO()
	op, err := o.Client.DescribeOperation(ctx, o.OperationID)
	if err != nil {
		return err
	}

	if op.Status.Status != v1.OperationStatusRunning {
		return fmt.Errorf("operation is not in running status, current status: %s", op.Status.Status)
	}

	if err := o.Client.TerminateOperation(ctx, o.OperationID); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "Operation %s termination requested\n", o.OperationID)
	return nil
}
