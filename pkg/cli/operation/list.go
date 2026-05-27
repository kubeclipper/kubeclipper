package operation

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

// ListOptions holds the options for the operation list subcommand.
type ListOptions struct {
	BaseOptions
	Cluster string
}

// NewListOptions creates a ListOptions with default values.
func NewListOptions(streams options.IOStreams) *ListOptions {
	return &ListOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

const listLong = `
  List platform operations.

  When --cluster is provided, operations are filtered by cluster name using a label selector.`

const listExample = `
  # List all operations
  kcctl operation list

  # List operations for a specific cluster
  kcctl operation list -c my-cluster

  # List operations in JSON format
  kcctl operation list -o json`

// NewCmdList creates the operation list subcommand.
func NewCmdList(streams options.IOStreams) *cobra.Command {
	o := NewListOptions(streams)
	cmd := &cobra.Command{
		Use:                   "list [-c CLUSTER]",
		DisableFlagsInUseLine: true,
		Short:                 "List operations",
		Long:                  listLong,
		Example:               listExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete(o.CliOpts))
			utils.CheckErr(o.RunList())
		},
	}
	cmd.Flags().StringVarP(&o.Cluster, "cluster", "c", o.Cluster, "Filter operations by cluster name")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	// Shell completion for --cluster flag
	cmd.RegisterFlagCompletionFunc("cluster", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		utils.CheckErr(o.Complete(o.CliOpts))
		return o.listClusterNames(toComplete), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// Complete initializes the API client from the CLI options.
func (o *ListOptions) Complete(opts *options.CliOptions) error {
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

// RunList fetches and displays the operation list.
func (o *ListOptions) RunList() error {
	q := query.New()
	if o.Cluster != "" {
		q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, o.Cluster)
	}

	result, err := o.Client.ListOperation(context.TODO(), kc.Queries(*q))
	if err != nil {
		return err
	}
	if len(result.Items) == 0 {
		if o.Cluster != "" {
			fmt.Fprintf(o.Out, "No operations found for cluster %s\n", o.Cluster)
		} else {
			fmt.Fprintln(o.Out, "No operations found")
		}
		return nil
	}
	return o.PrintFlags.Print(result, o.Out)
}

// listClusterNames returns cluster names for shell completion.
func (o *ListOptions) listClusterNames(toComplete string) []string {
	q := query.New()
	data, err := o.Client.ListClusters(context.TODO(), kc.Queries(*q))
	if err != nil {
		return nil
	}
	var names []string
	for _, v := range data.Items {
		if strings.HasPrefix(v.Name, toComplete) {
			names = append(names, v.Name)
		}
	}
	return names
}
