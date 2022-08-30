package fed

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	"github.com/spf13/cobra"
)

type FederalOptions struct {
	options.IOStreams
	CliOpts *options.CliOptions
	Client  *kc.Client
	SSH     *sshutils.SSH

	ClusterName    string
	ClusterType    string
	FedType        string
	KubeConfigFile string
	APIServer      string
	Region         string

	Unbind string
}

func NewFederalOptions(streams options.IOStreams) *FederalOptions {
	return &FederalOptions{
		IOStreams: streams,
		CliOpts:   options.NewCliOptions(),
		SSH:       &sshutils.SSH{User: "root"},
	}
}

func NewCmdFederal(streams options.IOStreams) *cobra.Command {
	o := NewFederalOptions(streams)
	cmd := &cobra.Command{
		Use:                   "fed [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "fed cluster",
		//Long:                  longDescription,
		//Example:               joinExample,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs())
			if !o.preCheck() {
				return
			}
			utils.CheckErr(o.Run())
		},
	}
	o.CliOpts.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&o.SSH.User, "user", o.SSH.User, "ssh user")
	cmd.Flags().StringVar(&o.SSH.Password, "passwd", o.SSH.Password, "ssh password")
	cmd.Flags().StringVar(&o.SSH.PkFile, "pk-file", o.SSH.PkFile, "ssh pk file which used to remote access other agent nodes")
	cmd.Flags().StringVar(&o.ClusterType, "cluster-type", o.ClusterType, "types of nano-managed clusters.")
	cmd.Flags().StringVar(&o.FedType, "fed-type", o.FedType, "types of nano-managed clusters. type: agent/pod")
	cmd.Flags().StringVar(&o.KubeConfigFile, "kubeconfig-file", o.KubeConfigFile, "the kubeconfig file of the managed cluster.")
	cmd.Flags().StringVar(&o.APIServer, "api-server", o.APIServer, "k8s apiserver addr of the managed cluster")
	cmd.Flags().StringVar(&o.Region, "region", o.Region, "set node region. default region: 'default'")
	cmd.Flags().StringVar(&o.Unbind, "unbind", o.Unbind, "unbind cluster")

	return cmd
}

func (o *FederalOptions) Complete() error {
	if err := o.CliOpts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(o.CliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	o.Client = c

	if o.Region == "" {
		o.Region = "default"
	}

	// deploy config Complete
	return nil
}

func (o *FederalOptions) ValidateArgs() error {
	if o.Unbind == "" {
		if o.ClusterName == "" {
			return fmt.Errorf("invalid cluster-name:%s", o.ClusterName)
		}
		if o.ClusterType != v1.ClusterKubeadm {
			return fmt.Errorf("invalid cluster-type:%s", o.ClusterType)
		}
		if o.FedType != v1.FedTypeAgent && o.FedType != v1.FedTypePod {
			return fmt.Errorf("invalid fed-type:%s", o.ClusterType)
		}
		if o.KubeConfigFile == "" {
			return fmt.Errorf("kubeconfig-file is empty")
		}
	}

	return nil
}

func (o *FederalOptions) preCheck() bool {
	return true
}

func (o *FederalOptions) Run() error {
	if o.Unbind != "" {
		return o.UnbindRun()
	}
	return o.FedRun()
}

func (o *FederalOptions) FedRun() error {
	config, err := ioutil.ReadFile(o.KubeConfigFile)
	if err != nil {
		return err
	}
	pkData, err := ioutil.ReadFile(o.SSH.PkFile)
	if err != nil && o.SSH.Password == "" {
		return err
	}
	c := &v1.Cluster{
		Provider: v1.ProviderSpec{
			Name:       o.ClusterType,
			FedType:    o.FedType,
			NodeRegion: o.Region,
			SSHConfig: v1.SSH{
				User:         o.SSH.User,
				Password:     o.SSH.Password,
				PkDataEncode: base64.StdEncoding.EncodeToString(pkData),
			},
			Kubeadm: v1.KubeadmSpec{
				APIServer:  o.APIServer,
				KubeConfig: base64.StdEncoding.EncodeToString(config),
			},
		},
	}
	c.Name = o.ClusterName

	_, err = o.Client.FedCluster(context.TODO(), c)

	// TODO: write deploy config
	return err
}

func (o *FederalOptions) UnbindRun() error {
	return o.Client.UnbindCluster(context.TODO(), o.Unbind)
}
