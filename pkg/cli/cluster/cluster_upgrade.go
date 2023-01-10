package cluster

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	v1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

const (
	upgradeLongDescription = `
	Upgrade cluster version
`
	clusterUpgradeExample = `
	# offline upgrade cluster 
	kcctl cluster upgrade --cluster-name clu-1 --version v1.21.3 --local-registry 172.20.150.138:5000
	# online upgrade cluster 
	kcctl cluster upgrade --cluster-name clu-1 --version v1.21.3 --online
`
)

const (
	versionRule = "^[v]([1-9]\\d|[1-9])(.([1-9]\\d|\\d)){2}$"
	addonType   = "k8s"
)

type ClusterUpgradeOpts struct {
	BaseOptions
	ClusterName   string
	Version       string
	Online        bool
	LocalRegistry string
}

func NewClusterUpgradeOpts(streams options.IOStreams) *ClusterUpgradeOpts {
	return &ClusterUpgradeOpts{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func NewCmdClusterUpgrade(streams options.IOStreams) *cobra.Command {
	c := NewClusterUpgradeOpts(streams)
	cmd := &cobra.Command{
		Use:     "upgrade (--cluster-name) (--version) [flags]",
		Short:   "upgrade kubernetes cluster version",
		Long:    upgradeLongDescription,
		Example: clusterUpgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(c.Complete())
			utils.CheckErr(c.Validates())
			utils.CheckErr(c.Run())
		},
	}

	cmd.Flags().StringVarP(&c.ClusterName, "cluster-name", "c", c.ClusterName, "cluster name")
	cmd.Flags().StringVarP(&c.Version, "version", "v", c.Version, "target version")
	cmd.Flags().BoolVar(&c.Online, "online", c.Online, "The way to upgrade")
	cmd.Flags().StringVarP(&c.LocalRegistry, "local-registry", "r", c.LocalRegistry, "image registry address")

	return cmd
}

func (c *ClusterUpgradeOpts) Complete() error {
	if err := c.CliOpts.Complete(); err != nil {
		return err
	}
	client, err := kc.FromConfig(c.CliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	c.Client = client
	return nil
}

func (c *ClusterUpgradeOpts) Validates() error {
	if c.Version == "" {
		return errors.New("please specify version")
	}
	if err := c.checkVersionFormat(); err != nil {
		return err
	}

	if err := c.checkVersionExist(); err != nil {
		return err
	}

	clu, err := c.checkClusterExist()
	if err != nil {
		return err
	}
	if c.Version <= clu.KubernetesVersion {
		return fmt.Errorf("your current version [%s] is lower than the version [%s] you specify", clu.KubernetesVersion, c.Version)
	}

	return c.checkVersionSpan(&clu)
}

func (c *ClusterUpgradeOpts) Run() error {
	clusterUpgrade := &v1.ClusterUpgrade{
		Version:       c.Version,
		Offline:       !c.Online,
		LocalRegistry: c.LocalRegistry,
	}

	return c.Client.UpgradeCluster(context.TODO(), c.ClusterName, clusterUpgrade)
}

func (c *ClusterUpgradeOpts) checkClusterExist() (corev1.Cluster, error) {
	var err error
	var clusterList *kc.ClustersList
	clusterList, err = c.Client.DescribeCluster(context.TODO(), c.ClusterName)
	if err != nil {
		return corev1.Cluster{}, err
	}
	return clusterList.Items[0], nil
}

func (c *ClusterUpgradeOpts) checkVersionFormat() error {
	m, err := regexp.Compile(versionRule)
	if err != nil {
		return err
	}
	if !m.MatchString(c.Version) {
		return errors.New("wrong version format")
	}
	return nil
}

func (c *ClusterUpgradeOpts) checkVersionExist() error {
	var err error
	var componentMeta *kc.ComponentMeta
	if c.Online {
		componentMeta, err = c.Client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"true"}})
	} else {
		componentMeta, err = c.Client.GetComponentMeta(context.TODO(), map[string][]string{"online": {"false"}})
	}
	if err != nil {
		return err
	}
	versions := sets.NewString()
	for _, addon := range componentMeta.Addons {
		if addon.Type == addonType {
			versions.Insert(addon.Version)
		}
	}
	if !versions.Has(c.Version) {
		return fmt.Errorf("version [%s] you specify is not avalible", c.Version)
	}
	return nil
}

func (c *ClusterUpgradeOpts) checkVersionSpan(clu *corev1.Cluster) error {
	currentVersion := strings.Split(clu.KubernetesVersion[1:], ".")
	targetVersion := strings.Split(c.Version[1:], ".")
	for index := 0; index < len(targetVersion)-1; index++ {
		a, err := strconv.Atoi(targetVersion[index])
		if err != nil {
			return err
		}
		b, err := strconv.Atoi(currentVersion[index])
		if err != nil {
			return err
		}
		if a-b > 1 {
			return fmt.Errorf("version [%s] you specify is too high", c.Version)
		}
	}
	return nil
}
