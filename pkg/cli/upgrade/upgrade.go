package upgrade

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	"github.com/spf13/cobra"
)

/*

kcctl upgrade all --pkg xxx --deploy-config xxx
kcctl upgrade all --pkg xxx --online
kcctl upgrade etcd --pkg xxx --deploy-config xxx
kcctl upgrade server --pkg xxx --deploy-config xxx

*/

const (
	longDescription = `
  Upgrade kubeclipper platform from own pkg or online pkg.

  The structure of online upgrade package as followings:
	/your_package
	├── bin
	├──├── etcd
	├──├── kubeclipper-agent
	├──├── kubeclipper-server
	├──├── ...
	├── kc-console
	├──── ├── ...
	├──── ├── ...
	  ...
  When you want to upgrade with your own package, your structure must be consistent with above.`
	upgradeExample = `
  # Upgrade whole kubeclipper platform use online pkg
  kcctl upgrade all --online

  # Upgrade whole kubeclipper platform use your own pkg
  kcctl upgrade all --pkg xxx

  # Upgrade agent of kubeclipper platform use your own pkg
  kcctl upgrade agent --pkg xxx

  # Upgrade agent kubeclipper platform use binary pkg
  kcctl upgrade agent --pkg xxx --binary`
)

var (
	serviceMap = make(map[string][]string)
)

type BaseOptions struct {
	deployConfig *options.DeployConfig
	CliOpts      *options.CliOptions
	SSHConfig    *sshutils.SSH
	options.IOStreams
}

type UpgradeOptions struct {
	BaseOptions
	pkg       string
	online    bool
	component string
	location  string
	serverIPs []string
	agentIPs  []string
}

func NewUpgradeOptions(stream options.IOStreams) *UpgradeOptions {
	return &UpgradeOptions{
		BaseOptions: BaseOptions{
			CliOpts: nil,
			SSHConfig: &sshutils.SSH{
				User: "root",
			},
			IOStreams:    stream,
			deployConfig: options.NewDeployOptions(),
		},
		online: false,
	}
}

func NewCmdUpgrade(stream options.IOStreams) *cobra.Command {
	o := NewUpgradeOptions(stream)
	cmd := &cobra.Command{
		Use:                   "upgrade",
		DisableFlagsInUseLine: true,
		Short:                 "upgrade kubeclipper components",
		Long:                  longDescription,
		Example:               upgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.Validate(cmd, args))
			utils.CheckErr(o.RunUpgrade())
		},
	}

	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "deploy-config file path.(Default path is ~/.kc/deploy-config.yaml)")
	cmd.Flags().StringVar(&o.pkg, "pkg", o.pkg, "new pkg path.")
	cmd.Flags().BoolVar(&o.online, "online", o.online, "use online upgrade pkg.")
	options.AddFlagsToSSH(o.deployConfig.SSHConfig, cmd.Flags())

	return cmd
}

func (o *UpgradeOptions) Complete() error {
	if err := o.deployConfig.Complete(); err != nil {
		return nil
	}
	if o.deployConfig.ServerIPs == nil || o.deployConfig.AgentRegions == nil {
		return fmt.Errorf("server node can't be empty, please check deploy-config file")
	}
	if o.online {
		o.pkg = options.DefaultUpgradePkg
	}
	if fileutil.IsDir(o.pkg) {
		return fmt.Errorf("--pkg can't be a directory")
	}
	return nil
}

func (o *UpgradeOptions) Validate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return utils.UsageErrorf(cmd, "You must specify the component of kubeclipper to upgrade, support [ agent | server | etcd | console ] now")
	}
	o.component = args[0]
	if len(o.deployConfig.ServerIPs)%2 == 0 {
		return fmt.Errorf("server node ")
	}
	o.serverIPs = o.deployConfig.ServerIPs
	o.agentIPs = o.deployConfig.AgentRegions.ListIP()
	serviceMap[options.UpgradeServer] = o.serverIPs
	serviceMap[options.UpgradeAll] = append(o.serverIPs, o.agentIPs...)
	serviceMap[options.UpgradeEtcd] = o.serverIPs
	serviceMap[options.UpgradeConsole] = o.serverIPs
	serviceMap[options.UpgradeAgent] = o.agentIPs
	if _, ok := serviceMap[o.component]; !ok {
		return utils.UsageErrorf(cmd, "unsupported upgrade component, support [ agent | server | etcd | console ] now")
	}
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
	tar := fmt.Sprintf("rm -rf %s && tar -xvf %s -C %s",
		filepath.Join(config.DefaultPkgPath, "kc"),
		filepath.Join(config.DefaultPkgPath, path.Base(o.deployConfig.Pkg)),
		config.DefaultPkgPath)
	err := utils.SendPackageV2(o.deployConfig.SSHConfig, o.pkg, serviceMap[o.component], config.DefaultPkgPath, nil, &tar)
	if err != nil {
		return err
	}
	return nil
}

func (o *UpgradeOptions) replaceService(comp string) error {
	before := fmt.Sprintf("systemctl stop kc-%s", comp)
	after := fmt.Sprintf("systemctl start kc-%s", comp)
	cp := o.copyCmd(comp)
	cmds := []string{before, cp, after}
	for _, cmd := range cmds {
		err := sshutils.CmdBatchWithSudo(o.deployConfig.SSHConfig, serviceMap[comp], sshutils.WrapSh(cmd), sshutils.DefaultWalk)
		if err != nil {
			return err
		}
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

func (o *UpgradeOptions) copyCmd(component string) (cp string) {
	switch component {
	case options.UpgradeAgent:
		cp = "cp -rf /tmp/kc/bin/kubeclipper-agent /usr/local/bin/"
	case options.UpgradeServer:
		cp = "cp -rf /tmp/kc/bin/kubeclipper-server /usr/local/bin/"
	case options.UpgradeEtcd:
		cp = "cp -rf /tmp/kc/bin/etcd* /usr/local/bin/"
	case options.UpgradeConsole:
		cp = sshutils.Combine([]string{
			"cp -rf /tmp/kc/bin/caddy /usr/local/bin/",
			"cp -rf /tmp/kc/kc-console/* /etc/kc-console/dist/",
		})
	}
	return
}
