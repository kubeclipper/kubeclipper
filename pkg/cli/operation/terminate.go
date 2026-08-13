package operation

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
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
  Cancel a pending or running operation.

  A Task that is already Running is allowed to finish; later Tasks are not started.`

const terminateExample = `
  # Cancel an operation
  kcctl operation cancel <OPERATION_ID>`

// NewCmdTerminate creates the operation terminate subcommand.
func NewCmdTerminate(streams options.IOStreams) *cobra.Command {
	o := NewTerminateOptions(streams)
	cmd := &cobra.Command{
		Use:                   "cancel OPERATION_ID",
		DisableFlagsInUseLine: true,
		Short:                 "Cancel an operation",
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

	if op.Status.Phase != operationsv1alpha1.OperationPending && op.Status.Phase != operationsv1alpha1.OperationRunning {
		return fmt.Errorf("operation cannot be cancelled from phase %s", op.Status.Phase)
	}

	if _, err := o.Client.CancelOperation(ctx, op); err != nil {
		return err
	}

	fmt.Fprintf(o.Out, "Operation %s cancellation requested\n", o.OperationID)
	return nil
}
