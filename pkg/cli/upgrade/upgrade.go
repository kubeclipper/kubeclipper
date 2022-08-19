package upgrade

import (
	"context"
	"fmt"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"
	"k8s.io/apimachinery/pkg/util/sets"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

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
  When you want to upgrade whole platform or console with your own package, your structure must be consistent with above.`
	upgradeExample = `
  # Upgrade whole kubeclipper platform use online pkg
  kcctl upgrade all --online

  # Upgrade whole kubeclipper platform use your own pkg
  kcctl upgrade all --pkg xxx

  # Upgrade agent of kubeclipper platform use your own pkg
  kcctl upgrade agent --pkg xxx`
)

var (
	serviceMap    = make(map[string][]string)
	allowedOnline = sets.NewString("master", "latest")
	onlinePkg     = "https://oss.kubeclipper.io/kc/%s/%s"
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
	pkg       string
	online    string
	component string
	pkgDir    string
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
		Use:                   "upgrade",
		DisableFlagsInUseLine: true,
		Short:                 "upgrade kubeclipper components",
		Long:                  longDescription,
		Example:               upgradeExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.Validate(cmd, args))
			utils.CheckErr(o.checkPkgStructure())
			utils.CheckErr(o.RunUpgrade())
		},
	}
	//TODO: add binary pkg upgrade
	cmd.Flags().StringVar(&o.deployConfig.Config, "deploy-config", options.DefaultDeployConfigPath, "deploy-config file path.")
	cmd.Flags().StringVar(&o.pkg, "pkg", o.pkg, "new pkg path.")
	cmd.Flags().StringVar(&o.online, "online", o.online, "input version e.g: v1.12.1 or master、latest")
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
	c, err := o.CliOpts.ToRawConfig().ToKcClient()
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

	if o.online != "" {
		if err := o.checkVersion(); err != nil {
			return err
		}
		o.pkgDir = filepath.Join(config.DefaultPkgPath, "kc")
	} else {
		o.pkgDir = filepath.Join(config.DefaultPkgPath, strings.ReplaceAll(filepath.Base(o.pkg), ".tar.gz", ""))
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

	if _, ok := serviceMap[o.component]; !ok {
		return utils.UsageErrorf(cmd, "unsupported upgrade component, support [ agent | server | console ] now")
	}

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

	if ok := m.MatchString(o.online); ok {
		// 与当前对比，不能往老版本升级
		// 应该是 git 版本  gitversion: v1.1.0-2+93c50dc822cdc
		v := strings.Split(versionInfo.GitVersion[1:], "-")
		current, _ := strconv.Atoi(strings.Join(strings.Split(v[0], "."), ""))
		version, _ := strconv.Atoi(strings.Join(strings.Split(o.online[1:], "."), ""))
		if current > version {
			return fmt.Errorf(" input version is lower than the current version ")
		}
	} else if !allowedOnline.Has(o.online) {
		return fmt.Errorf("wrong version (%s),there is no this branch ", o.online)
	}

	//TODO: check for version compatibility issues

	o.pkg = fmt.Sprintf(onlinePkg, o.online, fmt.Sprintf("kc-%s-%s", platforms[0], platforms[1]))

	return nil
}

func (o *UpgradeOptions) checkPkgStructure() error {
	if _, ok := httputil.IsURL(o.pkg); ok {
		return nil
	}
	cmd, err := cmdutil.RunCmd(false, "tar", "-tf", o.pkg)
	if err != nil {
		return err
	}
	name := filepath.Base(o.pkgDir)
	//TODO: upgrade single component with offline + local package condition
	if !strings.Contains(cmd.StdOut(), fmt.Sprintf("%s/bin", name)) {
		return fmt.Errorf("package structure:\n%s\nNon-standard. standard : 'packake/bin' example: kc/bin/kcctl", cmd.StdOut())
	}
	if o.component == options.UpgradeAll || o.component == options.UpgradeConsole {
		if !strings.Contains(cmd.StdOut(), fmt.Sprintf("%s/kc-console", name)) {
			return fmt.Errorf("package structure:\n%s\nNon-standard. standard : 'packake/kc-console' example: kc/kc-console/... ", cmd.StdOut())
		}
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
	cp, backup := o.copyAndBackup(comp)
	cmds := []string{
		sshutils.Combine([]string{
			"mkdir -p /tmp/kubeclipper/bin",
			"mkdir -p /tmp/kubeclipper/console",
		}),
		fmt.Sprintf("systemctl stop kc-%s", comp),
		backup,
		cp,
		fmt.Sprintf("systemctl start kc-%s", comp),
	}
	for _, cmd := range cmds {
		err := sshutils.CmdBatchWithSudo(o.deployConfig.SSHConfig, serviceMap[comp], sshutils.WrapSh(cmd), sshutils.DefaultWalk)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeOptions) copyAndBackup(component string) (cp, backup string) {
	switch component {
	case options.UpgradeAgent:
		cp = fmt.Sprintf("cp -rf %s/bin/kubeclipper-agent /usr/local/bin/kubeclipper-agent", o.pkgDir)
		backup = "cp /usr/local/bin/kubeclipper-agent /tmp/kubeclipper/bin/kubeclipper-agent"
	case options.UpgradeServer:
		cp = fmt.Sprintf("cp -rf %s/bin/kubeclipper-server /usr/local/bin/kubeclipper-server", o.pkgDir)
		backup = "cp /usr/local/bin/kubeclipper-server /tmp/kubeclipper/bin/kubeclipper-serve"
	case options.UpgradeConsole:
		cp = sshutils.Combine([]string{
			fmt.Sprintf("cp -f %s/bin/caddy /usr/local/bin/caddy", o.pkgDir),
			fmt.Sprintf("cp -rf %s/kc-console/* /etc/kc-console/dist/", o.pkgDir),
		})
		backup = sshutils.Combine([]string{
			"cp -rf /usr/local/bin/caddy /tmp/kubeclipper/bin/caddy",
			"cp -rf /etc/kc-console /tmp/kubeclipper/kc-console",
		})
	}
	return
}
