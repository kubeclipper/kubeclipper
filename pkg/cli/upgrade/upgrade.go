package upgrade

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const (
	longDescription = `
  Upgrade kubeclipper platform from own pkg or online pkg.

  The structure of online upgrade package as followings:
	/your_package
	├── kcctl
	├── kubeclipper-agent
	├── kubeclipper-server
	├── ...
	├── kc-console
	├──── ├── ...
	├──── ├── ...
	  ...
  When you want to upgrade whole platform or console with your own package, your structure must be consistent with above.`
	upgradeExample = `
  # Upgrade whole kubeclipper platform use online pkg
  kcctl upgrade all --online --version ( vX.X.X | branch-name )

  # Upgrade whole kubeclipper platform use your own pkg
  kcctl upgrade all --pkg xxx

  # Upgrade agent of kubeclipper platform use your own pkg
  kcctl upgrade agent --pkg xxx

  # Upgrade agent of kubeclipper platform use your own binary file
  kcctl upgrade agent --pkg xxx --binary`
)

var (
	serviceMap    = make(map[string][]string)
	allowedOnline = sets.NewString("master", "latest")
	onlinePkg     = "https://oss.kubeclipper.io/release/%s/kc-upgrade-%s.tar.gz"
)

type BaseOptions struct {
	deployConfig *options.DeployConfig
	CliOpts      *options.CliOptions
	client       *kc.Client
	SSHConfig    *sshutils.SSH
	options.IOStreams
}

type UpgradeOptions struct {
	BaseOptions
	arch      string
	pkg       string
	binary    bool
	online    bool
	version   string
	component string
	target    string
	serverIPs []string
	agentIPs  []string
}

func NewUpgradeOptions(stream options.IOStreams) *UpgradeOptions {
	return &UpgradeOptions{
		BaseOptions: BaseOptions{
			CliOpts: options.NewCliOptions(),
			client:  nil,
			SSHConfig: &sshutils.SSH{
				User: "root",
			},
			IOStreams:    stream,
			deployConfig: options.NewDeployOptions(),
		},
	}
}

func NewCmdUpgrade(stream options.IOStreams) *cobra.Command {
	o := NewUpgradeOptions(stream)
	cmd := &cobra.Command{
		Use:                   "upgrade ( component ) ( --pkg [--binary] )|( --online --version ) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "upgrade kubeclipper platform or components",
		Long:                  longDescription,
		Example:               upgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.Validate(cmd, args))
			utils.CheckErr(o.RunUpgrade())
		},
	}
	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "deploy-config file path.")
	cmd.Flags().StringVar(&o.pkg, "pkg", o.pkg, "Path to the package used for the upgrade")
	cmd.Flags().BoolVar(&o.binary, "binary", o.binary, "Upgrade with the specified binary file")
	cmd.Flags().BoolVar(&o.online, "online", o.online, "upgrade with online package")
	cmd.Flags().StringVar(&o.version, "version", o.version, "input version or branch name. e.g: v1.12.1 or master、latest")
	options.AddFlagsToSSH(o.deployConfig.SSHConfig, cmd.Flags())

	return cmd
}

func (o *UpgradeOptions) Complete() error {
	if err := o.deployConfig.Complete(); err != nil {
		return nil
	}
	if o.deployConfig.ServerIPs == nil {
		return fmt.Errorf("server node can't be empty, please check deploy-config file")
	}
	if err := o.CliOpts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(o.CliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	o.client = c
	return nil
}

func (o *UpgradeOptions) Validate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return utils.UsageErrorf(cmd, "You must specify the component of kubeclipper to upgrade, support [ agent | server | etcd | console ] now")
	}
	o.component = args[0]

	if o.online && o.binary {
		return fmt.Errorf("cannot use binary for online upgrade")
	}
	if o.online {
		if err := o.checkVersion(); err != nil {
			return err
		}
	} else if o.binary {
		if err := o.checkBinary(); err != nil {
			return err
		}
	}

	if len(o.deployConfig.ServerIPs)%2 == 0 {
		return fmt.Errorf("server node must be even number")
	}
	o.serverIPs = o.deployConfig.ServerIPs
	o.agentIPs = o.deployConfig.Agents.ListIP()
	serviceMap[options.UpgradeServer] = o.serverIPs
	serviceMap[options.UpgradeAll] = append(o.serverIPs, o.agentIPs...)
	serviceMap[options.UpgradeConsole] = o.serverIPs
	serviceMap[options.UpgradeAgent] = o.agentIPs
	serviceMap[options.UpgradeKcctl] = o.serverIPs

	if _, ok := serviceMap[o.component]; !ok {
		return utils.UsageErrorf(cmd, "unsupported upgrade component, support [ all | kcctl | agent | server | console ] now")
	}

	return nil
}

func (o *UpgradeOptions) checkBinary() error {
	if o.component == options.UpgradeAll || o.component == options.UpgradeConsole {
		return fmt.Errorf("can not upgrade kc and console using binary file")
	}
	cmd, err := cmdutil.RunCmd(false, "file", o.pkg)
	if err != nil {
		return err
	}
	if !strings.Contains(cmd.StdOut(), "ELF") {
		return fmt.Errorf("pkg [%s] is not a binary file", o.pkg)
	}
	tar := fmt.Sprintf("mkdir -p /tmp/kc && cp %s /tmp/kc && cd /tmp && tar -cf /tmp/kc-%s.tar.gz kc", o.pkg, o.component)
	sshutils.Cmd("/bin/sh", "-c", tar)
	o.pkg = fmt.Sprintf("/tmp/kc-%s.tar.gz", o.component)
	return nil
}

func (o *UpgradeOptions) checkVersion() error {
	role := "^[v]([1-9]\\d|[1-9])(.([1-9]\\d|\\d)){2}$"
	m, err := regexp.Compile(role)
	if err != nil {
		return err
	}

	versionInfo, err := o.client.Version(context.TODO())
	if err != nil {
		return err
	}
	platforms := strings.Split(versionInfo.Platform, "/")
	o.arch = platforms[1]

	//TODO: check for version compatibility issues
	if ok := m.MatchString(o.version); ok {
		v := strings.Split(versionInfo.GitVersion[1:], "-")
		current, _ := strconv.Atoi(strings.Join(strings.Split(v[0], "."), ""))
		version, _ := strconv.Atoi(strings.Join(strings.Split(o.version[1:], "."), ""))
		if current > version {
			return fmt.Errorf(" input version is lower than the current version ")
		}
	} else if !ok && !allowedOnline.Has(o.version) {
		return fmt.Errorf("wrong version format")
	}
	o.pkg = fmt.Sprintf(onlinePkg, o.version, o.arch)

	return nil
}

func (o *UpgradeOptions) RunUpgrade() error {
	err := o.sendPackage()
	if err != nil {
		return err
	}
	switch o.component {
	case options.UpgradeAll:
		err = o.replaceAllService()
	default:
		err = o.replaceService(o.component)
	}
	if err != nil {
		return err
	}
	return nil
}

func (o *UpgradeOptions) sendPackage() error {
	tar := fmt.Sprintf("tar -xvf %s -C %s",
		filepath.Join(config.DefaultPkgPath, path.Base(o.pkg)),
		config.DefaultPkgPath)
	err := utils.SendPackageV2(o.deployConfig.SSHConfig, o.pkg, serviceMap[o.component], config.DefaultPkgPath, nil, &tar)
	if err != nil {
		return err
	}
	return nil
}

func (o *UpgradeOptions) replaceAllService() error {
	for component := range serviceMap {
		if component == options.UpgradeAll {
			continue
		}
		if err := o.replaceService(component); err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeOptions) replaceService(comp string) error {
	cmds := o.replaceServiceCmds(comp)
	cmds = append([]string{"mkdir -p /tmp/kubeclipper"}, cmds...)
	for _, cmd := range cmds {
		err := sshutils.CmdBatchWithSudo(o.deployConfig.SSHConfig, serviceMap[comp], sshutils.WrapSh(cmd), sshutils.DefaultWalk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeOptions) replaceServiceCmds(component string) []string {
	cmds := make([]string, 0)
	switch component {
	case options.UpgradeKcctl:
		cmds = append(cmds, []string{
			"cp /usr/local/bin/kcctl /tmp/kubeclipper/kcctl",
			"cp -rf /tmp/kc/kcctl /usr/local/bin/kcctl",
		}...)
	case options.UpgradeConsole:
		cmds = append(cmds, []string{
			"systemctl stop kc-console",
			"cp -rf /etc/kc-console /tmp/kubeclipper/",
			"cp -rf /tmp/kc/kc-console/* /etc/kc-console/dist/",
			"systemctl start kc-console",
		}...)
	default:
		cmds = append(cmds, []string{
			fmt.Sprintf("systemctl stop kc-%s", component),
			fmt.Sprintf("cp /usr/local/bin/kubeclipper-%s /tmp/kubeclipper/kubeclipper-%s", component, component),
			fmt.Sprintf("cp -rf /tmp/kc/kubeclipper-%s /usr/local/bin/kubeclipper-%s", component, component),
			fmt.Sprintf("systemctl start kc-%s", component),
		}...)
	}
	return cmds
}
