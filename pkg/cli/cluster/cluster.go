package cluster

import (
	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	longDescription = `
	Cluster operations
`
	clusterExample = `
	# upgrade k8s cluster version 
	cluster upgrade --cluster-name clu-1 --version v1.23.9 --offline
`
)

type BaseOptions struct {
	PrintFlags *printer.PrintFlags
	CliOpts    *options.CliOptions
	options.IOStreams
	Client *kc.Client
}

func NewCmdCluster(streams options.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "cluster (command) [options]",
		DisableFlagsInUseLine: true,
		Short:                 "k8s cluster operation",
		Long:                  longDescription,
		Example:               clusterExample,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(NewCmdClusterUpgrade(streams))
	return cmd
}
