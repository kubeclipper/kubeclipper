package operation

import (
	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// BaseOptions contains common options shared by all operation subcommands.
type BaseOptions struct {
	PrintFlags *printer.PrintFlags
	CliOpts    *options.CliOptions
	options.IOStreams
	Client *kc.Client
}

const (
	longDescription = `
  Manage platform operations (install, upgrade, backup, etc.)

  Subcommands:
    list       List operations, optionally filtered by cluster
    describe   Show detailed information about an operation
    logs       Show plain-text logs of an operation
    retry      Retry a failed operation
    terminate  Terminate a running operation`

	example = `
  # List all operations
  kcctl operation list

  # List operations for a specific cluster
  kcctl operation list -c my-cluster

  # Describe an operation
  kcctl operation describe <OPERATION_ID>

  # Show operation logs
  kcctl operation logs <OPERATION_ID>

  # Follow operation logs in real time
  kcctl operation logs <OPERATION_ID> --follow

  # Retry a failed operation
  kcctl operation retry <OPERATION_ID>

  # Terminate a running operation
  kcctl operation terminate <OPERATION_ID>`
)

// NewCmdOperation creates the operation command with all subcommands.
func NewCmdOperation(streams options.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "operation [subcommand]",
		DisableFlagsInUseLine: true,
		Short:                 "Manage platform operations",
		Long:                  longDescription,
		Example:               example,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(NewCmdList(streams))
	cmd.AddCommand(NewCmdDescribe(streams))
	cmd.AddCommand(NewCmdLogs(streams))
	cmd.AddCommand(NewCmdRetry(streams))
	cmd.AddCommand(NewCmdTerminate(streams))

	return cmd
}
