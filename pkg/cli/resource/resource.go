/*
 *
 *  * Copyright 2021 KubeClipper Authors.
 *  *
 *  * Licensed under the Apache License, Version 2.0 (the "License");
 *  * you may not use this file except in compliance with the License.
 *  * You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package resource

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

/*
kubeclipper operation resource

Usage:
  kcctl resource list
  kcctl resource push
  kcctl resource delete

Examples:
  kcctl resource list --deploy-config /root/.kc/deploy-config.yaml --pk-file ssh-key

  kcctl resource push --deploy-config /root/.kc/deploy-config.yaml --pk-file key --pkg docker-19.03.12-x86_64.tar.gz --type cri

  kcctl resource delete --deploy-config /root/.kc/deploy-config.yaml --pk-file key --name docker --version 19.03.12 --arch x86_64

Flags:
  -h, --help                   help for registry
*/

const (
	longDescription = `
  Offline resource operation.

  Currently, You can push, delete, and list offline resource packs.`
	resourceExample = `
  # List offline resource packs
  kcctl resource list (--pk-file | --pk-passwd | --passwd) [flags]
  # Push offline resource packs
  kcctl push (--pk-file <file path> | --pk-passwd <pwd> | --passwd <pwd>) (--pkg <file name>) [flags]
  # Delete offline resource packs
  delete (--pk-file <file path> | --pk-passwd <pwd> | --passwd <pwd>) (--name <pkg-name>) (--version <pkg-version>) (--arch <pkg-arch>) [flags]

  Please read 'kcctl resource -h' get more resource flags.`
	listLongDescription = `
  List offline resource packs

  You can list, push, or delete offline resource packs.
  The deploy-config flag is '/root/.kc/deploy-config.yaml' by defualt.`
	resourceListExample = `
  # List offline resource use ssh
  kcctl resource list --pk-file 'PK-FILE PATH'

  # List offline resource use deploy password
  kcctl resource list --passwd 'DEPLOY PASSWORD'

  # List offline resource use deploy user, default user is root
  kcctl resource list --user 'USER' --pk-file 'PK-FILE PATH'

  # List offline resource use specified output format
  kcctl resource list --pk-file 'PK-FILE PATH' --output 'YAML|TABLE|JSON'

  # List offline resource use specified deploy file
  kcctl resource list --deploy-config 'FILE PATH' --pk-file 'PK-FILE PATH'

  Please read 'kcctl resource list -h' get more resource list flags`
	pushLongDescription = `
  Push offline resource packs

  You can push a .tar.gz file of the specified type
  The deploy-config flag is '/root/.kc/deploy-config.yaml' by defualt.

  Naming rules for offline packages: name-version-arch.tar.gz
  Structure of the offline package: 
	name/version/
	name/version/arch/
	name/version/arch/images.tar.gz
	name/version/arch/manifest.json`
	resourcePushExample = `
  # Push offline resource packs use ssh
  kcctl resource push --pk-file 'PK-FILE PATH' --pkg 'PKG NAME' --type 'TYPE'

  # Push offline resource packs use specified deploy file
  kcctl resource push --deploy-config 'DEPLOY FILE PATH' --pk-file 'PK-FILE PATH' --pkg 'PKG NAME' --type 'TYPE'

  Please read 'kcctl resource push -h' get more resource push flags`
	deleteLongDescription = `
  Delete offline resource packs

  You can delete existing offline packages.
  You need to specify the name, type, arch of offline packages before deleting.
  The deploy-config flag is '/root/.kc/deploy-config.yaml' by defualt.`
	resourceDeleteExample = `
  # Delete offline resource packs use ssh
  kcctl resource delete --pk-file 'PK-FILE PATH' --name 'NAME' --version 'VERSION' --arch 'ARCH'

  # Delete offline resource packs use specified deploy file
  kcctl resource delete --deploy-config 'DEPLOY FILE PATH' --pk-file 'PK-FILE PATH' --name 'NAME' --version 'VERSION' --arch 'ARCH'

  Please read 'kcctl resource delete -h' get more resource delete flags`
)

type ResourceOptions struct {
	options.IOStreams
	PrintFlags   *printer.PrintFlags
	SSHConfig    *sshutils.SSH
	DeployConfig string
	deployConfig *options.DeployConfig

	List   string
	Push   string
	Delete string

	Type    string
	Name    string
	Version string
	Arch    string

	Pkg string
}

func NewResourceOptions(streams options.IOStreams) *ResourceOptions {
	return &ResourceOptions{
		IOStreams:  streams,
		PrintFlags: printer.NewPrintFlags(),
		SSHConfig: &sshutils.SSH{
			User: "root",
		},
		deployConfig: options.NewDeployOptions(),
		Arch:         "amd64",
	}
}

func NewCmdResource(streams options.IOStreams) *cobra.Command {
	o := NewResourceOptions(streams)
	cmd := &cobra.Command{
		Use:                   "resource",
		DisableFlagsInUseLine: true,
		Short:                 "offline resource operation",
		Long:                  longDescription,
		Example:               resourceExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.ResourcePkgRules())
		},
	}

	cmd.AddCommand(NewCmdResourceList(o))
	cmd.AddCommand(NewCmdResourcePush(o))
	cmd.AddCommand(NewCmdResourceDelete(o))

	return cmd
}

func NewCmdResourceList(o *ResourceOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "list (--pk-file | --pk-passwd | --passwd) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "offline resource list",
		Long:                  listLongDescription,
		Example:               resourceListExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs(cmd))
			utils.CheckErr(o.ResourceList())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	o.PrintFlags.AddFlags(cmd)
	cmd.Flags().StringVar(&o.Type, "type", o.Type, "offline resource type.")
	cmd.Flags().StringVar(&o.Name, "name", o.Name, "offline resource name.")
	cmd.Flags().StringVar(&o.Version, "version", o.Name, "offline resource version.")
	cmd.Flags().StringVar(&o.Arch, "arch", o.Arch, "offline resource arch.")
	cmd.Flags().StringVar(&o.DeployConfig, "deploy-config", options.DefaultDeployConfigPath, "kcctl deploy config path")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listType(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listName(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listVersion(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("arch", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listArch(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	return cmd
}

func (o *ResourceOptions) preCheck() bool {
	return sudo.PreCheck("sudo", o.SSHConfig, o.IOStreams, o.deployConfig.ServerIPs)
}

func NewCmdResourcePush(o *ResourceOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "push (--pk-file <file path> | --pk-passwd <pwd> | --passwd <pwd>) (--pkg <file name>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "offline resource push",
		Long:                  pushLongDescription,
		Example:               resourcePushExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgsPush(cmd))
			if !o.preCheck() {
				return
			}
			utils.CheckErr(o.ResourcePush())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Type, "type", o.Type, "offline resource type.")
	cmd.Flags().StringVar(&o.Pkg, "pkg", o.Pkg, "docker service and images pkg.")
	cmd.Flags().StringVar(&o.DeployConfig, "deploy-config", options.DefaultDeployConfigPath, "kcctl deploy config path")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listType(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("type"))
	utils.CheckErr(cmd.MarkFlagRequired("pkg"))
	return cmd
}

func NewCmdResourceDelete(o *ResourceOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "delete (--pk-file <file path> | --pk-passwd <pwd> | --passwd <pwd>) (--name <pkg-name>) (--version <pkg-version>) (--arch <pkg-arch>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "offline resource delete",
		Long:                  deleteLongDescription,
		Example:               resourceDeleteExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgsDelete(cmd))
			if !o.preCheck() {
				return
			}
			utils.CheckErr(o.ResourceDelete())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Name, "name", o.Name, "offline resource name.")
	cmd.Flags().StringVar(&o.Version, "version", o.Name, "offline resource version.")
	cmd.Flags().StringVar(&o.Arch, "arch", o.Arch, "offline resource arch.")
	cmd.Flags().StringVar(&o.DeployConfig, "deploy-config", options.DefaultDeployConfigPath, "kcctl deploy config path")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listName(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listVersion(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("arch", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listArch(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("name"))
	utils.CheckErr(cmd.MarkFlagRequired("version"))
	utils.CheckErr(cmd.MarkFlagRequired("arch"))
	return cmd
}

func (o *ResourceOptions) Complete() error {
	o.deployConfig.Config = o.DeployConfig
	err := o.deployConfig.Complete()
	return err
}

func (o *ResourceOptions) ValidateArgs(cmd *cobra.Command) error {
	if o.deployConfig.Config == "" {
		return utils.UsageErrorf(cmd, "the deploy-config.yaml file path is required")
	}
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return utils.UsageErrorf(cmd, "one of --pk-file or --passwd must be specified")
	}
	return nil
}

func (o *ResourceOptions) ValidateArgsPush(cmd *cobra.Command) error {
	if o.Type == "" {
		return utils.UsageErrorf(cmd, "the type of resource must be specified")
	}
	if o.Pkg == "" {
		return utils.UsageErrorf(cmd, "resource pkg  must be specified")
	}
	if o.deployConfig.Config == "" {
		return utils.UsageErrorf(cmd, "--deploy-config must be specified")
	}
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return utils.UsageErrorf(cmd, "one of --pk-file or --passwd must be specified")
	}
	return nil
}

func (o *ResourceOptions) ValidateArgsDelete(cmd *cobra.Command) error {
	if o.Name == "" {
		return utils.UsageErrorf(cmd, "the name of resource must be specified")
	}
	if o.Version == "" {
		return utils.UsageErrorf(cmd, "the version of resource must be specified")
	}
	if o.Arch == "" {
		return utils.UsageErrorf(cmd, "the arch of resource must be specified")
	}
	if o.deployConfig.Config == "" {
		return utils.UsageErrorf(cmd, "--deploy-config is required")
	}
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return utils.UsageErrorf(cmd, "one of --pk-file or --passwd must be specified")
	}
	return nil
}

func (o *ResourceOptions) ResourcePkgRules() error {
	logger.Info(">>> package name rule: 'name-version-arch.tar.gz'  example: 'k8s-v1.20.13-x86_64.tar.gz'")
	logger.Infof(">>> package struct rule: \ntar -tf k8s-v1.20.13-x86_64.tar.gz \nv1.20.13/x86_64/10-kubeadm.conf\nv1.20.13/x86_64/checksum.md5\nv1.20.13/x86_64/conntrack\nv1.20.13/x86_64/images.tar.gz\nv1.20.13/x86_64/kubeadm\nv1.20.13/x86_64/kubectl\nv1.20.13/x86_64/kubelet\nv1.20.13/x86_64/kubelet-pre-start.sh\nv1.20.13/x86_64/kubelet.service\nv1.20.13/x86_64/README.md")
	return nil
}

func (o *ResourceOptions) ResourceList() error {
	var (
		errMap   = make(map[string]string)
		metaList = make([]*kc.ComponentMetas, 0)
	)

	for _, node := range o.deployConfig.ServerIPs {
		metas, err := o.ReadMetadata(node)
		if err != nil {
			errMap[node] = fmt.Sprintf("node(%s) read metadata.json error: %s", node, err.Error())
			continue
		}
		metaList = append(metaList, metas)
	}

	for node, metas := range o.filter(metaList) {
		err := o.PrintFlags.Print(metas, o.IOStreams.Out)
		if err != nil {
			errMap[node] = fmt.Sprintf("node(%s) read metadata.json error: %s", node, err.Error())
		}
	}

	for _, val := range errMap {
		logger.Warnf(val)
	}

	return nil
}

func (o *ResourceOptions) filter(data []*kc.ComponentMetas) map[string]printer.ResourcePrinter {
	var metaMap = make(map[string]printer.ResourcePrinter)

	for _, metas := range data {
		n := &kc.ComponentMetas{
			Node: metas.Node,
		}

		for _, resource := range metas.ComponentMetaList {
			if o.Type != "" && resource.Type != o.Type {
				continue
			}
			if o.Name != "" && resource.Name != o.Name {
				continue
			}
			if o.Version != "" && resource.Version != o.Version {
				continue
			}
			if o.Arch != "" && resource.Arch != o.Arch {
				continue
			}
			n.ComponentMetaList = append(n.ComponentMetaList, resource)
		}
		n.TotalCount = len(n.ComponentMetaList)
		metaMap[n.Node] = n
	}
	return metaMap
}

func (o *ResourceOptions) ResourcePush() error {
	/*
		step
		1.write tmp metadata.json
		2.send pgk to remote
		3.clean old pkg
		4.tar new pkg
		5.send tmp metadata.json to remote
		6.del tmp metadata.json
	*/
	// check local package
	name, version, arch, err := o.parsePackageName()
	if err != nil {
		return err
	}
	if _, ok := httputil.IsURL(o.Pkg); !ok {
		ec, err := cmdutil.RunCmd(false, "tar", "-tf", o.Pkg)
		if err != nil {
			return err
		}
		if !strings.Contains(ec.StdOut(), fmt.Sprintf("%s/%s", version, arch)) {
			return fmt.Errorf("package structure(%s) Non-standard. standard : 'version/arch/file' example: v4.0.2/amd64/images.tar.gz", ec.StdOut())
		}
	}

	// read metadata.json
	for _, node := range o.deployConfig.ServerIPs {
		metas, err := o.ReadMetadata(node)
		if err != nil {
			return err
		}

		metas.AppendOnly(o.Type, name, version, arch)

		metaBytes, err := json.MarshalIndent(&metas.ComponentMetaList, "", "  ")
		if err != nil {
			return err
		}

		err = ioutil.WriteFile("metadata.json", metaBytes, 0755)
		if err != nil {
			return err
		}

		// send pkg
		hook := fmt.Sprintf(`cd %s && rm -rf $(tar tf %s | awk -F\/ '{print $1}' | uniq | sed '/^$/d')`, config.DefaultPkgPath, filepath.Base(o.Pkg))
		err = utils.SendPackageV2(o.SSHConfig, o.Pkg, []string{node}, config.DefaultPkgPath, nil, &hook)
		if err != nil {
			return err
		}
		// checkout download pkg
		if _, ok := httputil.IsURL(o.Pkg); ok {
			hook = fmt.Sprintf("tar -tf %s", filepath.Join(config.DefaultPkgPath, filepath.Base(o.Pkg)))
			ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, node, hook)
			if err != nil {
				logger.Errorf("node(%s) push resource failed: %s", node, err.Error())
				return err
			}
			if !strings.Contains(ret.Stdout, fmt.Sprintf("%s/%s", version, arch)) {
				return fmt.Errorf("package structure(%s) Non-standard. standard : 'version/arch/file' example: v4.0.2/amd64/images.tar.gz", ret.Stdout)
			}
		}

		// clean up old file
		clean := fmt.Sprintf(`rm -rf %s/%s/%s/%s && mkdir -p %s/%s/%s/%s`, o.deployConfig.StaticServerPath, name, version, arch, o.deployConfig.StaticServerPath, name, version, arch)
		ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, node, clean)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}

		// tar decompress new file
		hook = fmt.Sprintf(`tar -zxvf %s -C %s`, filepath.Join(config.DefaultPkgPath, filepath.Base(o.Pkg)), o.deployConfig.StaticServerPath)
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, node, hook)
		if err != nil {
			logger.Errorf("node(%s) push resource failed: %s", node, err.Error())
			return err
		}
		if err = ret.Error(); err != nil {
			logger.Errorf("node(%s) push resource failed: %s", node, err.Error())
			return err
		}
	}

	// send metadata.json
	err = utils.SendPackageV2(o.SSHConfig, "metadata.json", o.deployConfig.ServerIPs, o.deployConfig.StaticServerPath, nil, nil)
	if err != nil {
		return err
	}
	_ = os.RemoveAll("metadata.json")

	logger.Info("resource push successfully")
	return nil
}

func (o *ResourceOptions) ResourceDelete() error {
	for _, node := range o.deployConfig.ServerIPs {
		metas, err := o.ReadMetadata(node)
		if err != nil {
			return err
		}
		exist := metas.Exist(o.Name, o.Version, o.Arch)
		if !exist {
			logger.Warnf("resource %s-%s-%s not exists", o.Name, o.Version, o.Arch)
			return nil
		}
		err = metas.Delete(o.Name, o.Version, o.Arch)
		if err != nil {
			return err
		}

		err = metas.WriteFile("", true)
		if err != nil {
			return err
		}
		err = utils.SendPackageV2(o.SSHConfig, "metadata.json", []string{node}, o.deployConfig.StaticServerPath, nil, nil)
		if err != nil {
			return err
		}
		err = metas.WriteFile("", true)
		if err != nil {
			return err
		}
		ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, node, fmt.Sprintf("rm -rf %s/%s/%s/%s", o.deployConfig.StaticServerPath, o.Name, o.Version, o.Arch))
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}

	logger.Info("resource delete successfully")
	return nil
}

func (o *ResourceOptions) ReadMetadata(node string) (*kc.ComponentMetas, error) {
	ret, err := sshutils.SSHCmd(o.SSHConfig, node, fmt.Sprintf("cat %s/%s", o.deployConfig.StaticServerPath, "metadata.json"))
	if err != nil {
		return nil, err
	}
	var metas kc.ComponentMetas
	err = json.Unmarshal([]byte(ret.Stdout), &metas.ComponentMetaList)
	metas.Node = node
	return &metas, err
}

func (o *ResourceOptions) parsePackageName() (string, string, string, error) {
	array := strings.Split(strings.ReplaceAll(filepath.Base(o.Pkg), ".tar.gz", ""), "-")
	if len(array) != 3 {
		return "", "", "", fmt.Errorf("package name '%s' Non-standard. example: 'name-version-amd64.tar.gz'", o.Pkg)
	}

	return array[0], array[1], array[2], nil
}

func (o *ResourceOptions) listType(toComplete string) []string {
	set := sets.NewString()
	resources := o.resourceList()
	for _, v := range resources {
		if strings.HasPrefix(v.Type, toComplete) {
			set.Insert(v.Type)
		}
	}
	return set.List()
}

func (o *ResourceOptions) listName(toComplete string) []string {
	set := sets.NewString()
	resources := o.resourceList()
	for _, v := range resources {
		if strings.HasPrefix(v.Name, toComplete) {
			set.Insert(v.Name)
		}
	}
	return set.List()
}

func (o *ResourceOptions) listVersion(toComplete string) []string {
	set := sets.NewString()
	resources := o.resourceList()
	for _, v := range resources {
		if strings.HasPrefix(v.Version, toComplete) {
			set.Insert(v.Version)
		}
	}
	return set.List()
}

func (o *ResourceOptions) listArch(toComplete string) []string {
	set := sets.NewString()
	resources := o.resourceList()
	for _, v := range resources {
		if strings.HasPrefix(v.Arch, toComplete) {
			set.Insert(v.Arch)
		}
	}
	return set.List()
}

func (o *ResourceOptions) resourceList() []v1.MetaResource {
	utils.CheckErr(o.Complete())

	list := make([]v1.MetaResource, 0)
	for _, node := range o.deployConfig.ServerIPs {
		metas, err := o.ReadMetadata(node)
		if err != nil {
			continue
		}
		for _, v := range metas.ComponentMetaList {
			if o.Type != "" && v.Type != o.Type {
				continue
			}
			if o.Name != "" && v.Name != o.Name {
				continue
			}
			if o.Version != "" && v.Version != o.Version {
				continue
			}
			if o.Arch != "" && v.Arch != o.Arch {
				continue
			}
			list = append(list, v)
		}
	}
	return list
}
