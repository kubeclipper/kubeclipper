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

package registry

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"

	"github.com/kubeclipper/kubeclipper/pkg/cli/registry/client"

	pkgerr "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"

	"github.com/spf13/cobra"

	"github.com/kubeclipper/kubeclipper/pkg/cli/sudo"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
)

const (
	longDescription = `
  Docker registry operation.

  Currently, you can deploy, clean, push, list and delete docker registry.
  Use docker engine API V2, visit the website(https://docs.docker.com/registry/spec/api/) for more information.`
	registryExample = `
  # Deploy docker registry
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz
  # Deploy docker registry without image load
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz --skip-image-load
  # Clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111
  # Push docker image to registry
  kcctl registry push --pk-file key --node 10.0.0.111 --pkg images.tar.gz
  # List repositories in docker registry
  kcctl registry list --node 10.0.0.111  --type repository
  # Delete docker image
  kcctl registry delete --node 10.0.0.111  --name etcd --tag 1.5.1-0

  Please read 'kcctl registry -h' get more registry flags.`
	deployLongDescription = `
  Deploy docker registry.`
	deployExample = `
  # Deploy docker registry
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz
  # Deploy docker registry specify data directory
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz  --data-root /var/lib/myregistry
  # Deploy docker registry specify port
  # If you used custom port,then must specify it in push、list、delete cmd.
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz --registry-port 6666

  Please read 'kcctl registry deploy -h' get more registry deploy flags.`
	cleanLongDescription = `
  Clean docker registry by flags.`
	cleanExample = `
  # Clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111
  # Clean docker registry, specify data directory.
  # If you used custom data directory when deploy,then must specify it in this cmd to clear data.
  kcctl registry clean --pk-file key --node 10.0.0.111 --registry-volume /opt/registry --data-root /var/lib/docker
  # Clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111
  # Forced to clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111  --force

  Please read 'kcctl registry clean -h' get more registry clean flags.`
	pushLongDescription = `
  Push docker image by flags.`
	pushExample = `
  # Push docker image to registry
  # You can use [docker save  $images > images.tar && gzip -f images.tar] to generate image pkg
  # example: docker save k8s.gcr.io/pause:3.2 k8s.gcr.io/coredns/coredns:1.6.7 > images.tar && gzip -f images.tar
  kcctl registry push --pk-file key --node 10.0.0.111  --pkg images.tar.gz

  Please read 'kcctl registry push -h' get more registry push flags.`
	listLongDescription = `
  Lists docker repositories or images by flags.`
	listExample = `
  # Lists docker repositories
  kcctl registry list --node 10.0.0.111  --type repository
  # Lists docker images
  kcctl registry list --node 10.0.0.111  --type image --name etcd 

  Please read 'kcctl registry list -h' get more registry list flags.`
	deleteLongDescription = `
  Delete the docker image by name and tag.`
	deleteExample = `
  # Delete docker image
  kcctl registry delete --pk-file key --node 10.0.0.111  --name etcd --tag 3.5.1-0

  Please read 'kcctl registry delete -h' get more registry delete flags.`
)

type RegistryOptions struct {
	options.IOStreams
	PrintFlags *printer.PrintFlags
	SSHConfig  *sshutils.SSH

	Node string
	Pkg  string

	DataRoot     string
	RegistryPort int

	Type          string
	Name          string
	Tag           string
	Number        int
	SkipImageLoad bool
}

var (
	allowType = sets.NewString("image", "repository")
)

func NewRegistryOptions(streams options.IOStreams) *RegistryOptions {
	return &RegistryOptions{
		IOStreams:    streams,
		PrintFlags:   printer.NewPrintFlags(),
		SSHConfig:    sshutils.NewSSH(),
		DataRoot:     "/var/lib/registry",
		RegistryPort: 5000,
		Type:         "repository",
	}
}

func NewCmdRegistry(streams options.IOStreams) *cobra.Command {
	o := NewRegistryOptions(streams)
	cmd := &cobra.Command{
		Use:                   "registry",
		DisableFlagsInUseLine: true,
		Short:                 "Registry operation",
		Long:                  longDescription,
		Example:               registryExample,
		Args:                  cobra.NoArgs,
	}

	cmd.AddCommand(NewCmdRegistryDeploy(o))
	cmd.AddCommand(NewCmdRegistryClean(o))
	cmd.AddCommand(NewCmdRegistryPush(o))
	cmd.AddCommand(NewCmdRegistryList(o))
	cmd.AddCommand(NewCmdRegistryDelete(o))

	return cmd
}

func NewCmdRegistryDeploy(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "deploy (--user | -u <user>) (--passwd <passwd>) (--pk-file <pk-file>) (--pk-passwd <pk-passwd>) (--node <node>) (--pkg <pkg>) (--data-root <data-root>)  (--registry-port <registry-port>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry deploy",
		Long:                  deployLongDescription,
		Example:               deployExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgsDeploy())
			if !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Install())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "node to deploy registry.")
	cmd.Flags().StringVar(&o.Pkg, "pkg", o.Pkg, "registry service and images pkg.")
	cmd.Flags().StringVar(&o.DataRoot, "data-root", o.DataRoot, "set registry data root directory.")
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "set registry port")
	cmd.Flags().BoolVar(&o.SkipImageLoad, "skip-image-load", o.SkipImageLoad, "set to skip image load,if set true will skip image load when deploy registry")

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("pkg"))
	return cmd
}

func NewCmdRegistryClean(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "clean (--user | -u <user>) (--passwd <passwd>) (--pk-file <pk-file>) (--pk-passwd <pk-passwd>) (--node <node>)",
		DisableFlagsInUseLine: true,
		Short:                 "registry clean",
		Long:                  cleanLongDescription,
		Example:               cleanExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgs())
			if !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Uninstall())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	utils.CheckErr(cmd.MarkFlagRequired("node"))
	return cmd
}

func NewCmdRegistryPush(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "push (--node <node>) (--pkg <pkg>) [--registry-port <registry-port>] [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry push image",
		Long:                  pushLongDescription,
		Example:               pushExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgsPush())
			if !o.healthPreCheck() {
				return
			}
			utils.CheckErr(o.Push())
		},
	}

	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	cmd.Flags().StringVar(&o.Pkg, "pkg", o.Pkg, "docker images pkg,use `docker save $images > images.tar && gzip -f images.tar` to generate images.tar.gz")
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "registry port.")

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("pkg"))
	return cmd
}

func NewCmdRegistryList(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "list (--node <node>) (--name <name>) (--registry-port <registry-port>) (--type <type>) (--number <number>) [flags]list (--node <node>) (--name <name>) (--registry-port <registry-port>) (--type <type>) (--number <number>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry list repository or image",
		Long:                  listLongDescription,
		Example:               listExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgsList())
			if !o.healthPreCheck() {
				return
			}
			utils.CheckErr(o.List())
		},
	}
	o.PrintFlags.AddFlags(cmd)
	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node")
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "registry port")

	cmd.Flags().StringVar(&o.Type, "type", o.Type, "image or repository")
	cmd.Flags().StringVar(&o.Name, "name", o.Name, "image name")
	cmd.Flags().IntVar(&o.Number, "number", o.Number, "number of entries in each response. It not present, all entries will be returned.")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return allowType.List(), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listRepos(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("type"))
	return cmd
}

func NewCmdRegistryDelete(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "delete (--pk-file <file path>) (--node <node>) (--name <name>) (--tag <tag>) [flags]delete (--pk-file <file path>) (--node <node>) (--name <name>)  (--tag <tag>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry delete image",
		Long:                  deleteLongDescription,
		Example:               deleteExample,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.ValidateArgsDelete(cmd))
			if !o.healthPreCheck() {
				return
			}
			utils.CheckErr(o.Delete())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "registry port")
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	cmd.Flags().StringVar(&o.Name, "name", o.Name, "image name")
	cmd.Flags().StringVar(&o.Tag, "tag", o.Tag, "image tag")

	utils.CheckErr(cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listRepos(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))
	utils.CheckErr(cmd.RegisterFlagCompletionFunc("tag", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return o.listTags(toComplete), cobra.ShellCompDirectiveNoFileComp
	}))

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("name"))
	utils.CheckErr(cmd.MarkFlagRequired("tag"))
	return cmd
}

func (o *RegistryOptions) healthPreCheck() bool {
	return o.healthz(true)
}

func (o *RegistryOptions) sudoPreCheck() bool {
	return sudo.PreCheck("sudo", o.SSHConfig, o.IOStreams, []string{o.Node})
}

func (o *RegistryOptions) ValidateArgs() error {
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
	}
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	return nil
}

func (o *RegistryOptions) ValidateArgsPush() error {
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	if o.Pkg == "" {
		return fmt.Errorf("--pkg must be specified")
	}
	return nil
}

func (o *RegistryOptions) ValidateArgsDeploy() error {
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
	}
	if o.Pkg == "" {
		return fmt.Errorf("--pkg must be specified")
	}
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	return nil
}

func (o *RegistryOptions) ValidateArgsList() error {
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	if o.Type != "image" && o.Type != "repository" {
		return fmt.Errorf("--type must be one of image,repository")
	}
	if o.Type == "image" && o.Name == "" {
		return fmt.Errorf("when type=image,--name is required")
	}
	return nil
}

func (o *RegistryOptions) ValidateArgsDelete(cmd *cobra.Command) error {
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	if o.Name == "" {
		return utils.UsageErrorf(cmd, "image name must be specified")
	}
	if o.Tag == "" {
		return utils.UsageErrorf(cmd, "image tag must be specified")
	}
	return nil
}

func (o *RegistryOptions) deployRegistry() error {
	data, err := o.GetKcRegistryConfigTemplateContent()
	if err != nil {
		return pkgerr.WithMessage(err, "render kc registry config failed")
	}
	cmdList := []string{
		"mkdir -pv /etc/kubeclipper-registry",
		sshutils.WrapEcho(data, "/etc/kubeclipper-registry/kubeclipper-registry.yaml"),
		sshutils.WrapEcho(config.KcRegistryService, "/usr/lib/systemd/system/kc-registry.service"),
		"systemctl daemon-reload && systemctl enable kc-registry --now",
	}
	for _, cmd := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return pkgerr.WithMessagef(err, "run deploy kc registry cmd[%s] failed", cmd)
		}
		if err = ret.Error(); err != nil {
			return pkgerr.WithMessagef(err, "deploy kc registry cmd[%s] failed", cmd)
		}
	}

	// check registry health
	err = wait.PollImmediate(time.Second, time.Second*15, func() (done bool, err error) {
		ok := o.healthz(false)
		if ok {
			logger.Info("registry is health.")
		} else {
			logger.V(2).Info("waiting for registry health.")
		}
		return ok, nil
	})
	if err != nil {
		logger.Warnf("waiting for registry health timeout,"+
			"please ssh to node [%s] and run cmd [journalctl -xu kc-registry] for more information.", o.Node)
	}
	return nil
}

func (o *RegistryOptions) GetKcRegistryConfigTemplateContent() (string, error) {
	tmpl, err := template.New("text").Parse(config.KcRegistryConfigTmpl)
	if err != nil {
		return "", fmt.Errorf("template parse failed: %s", err.Error())
	}

	var data = make(map[string]interface{})
	data["RegistryPort"] = o.RegistryPort
	data["DataRoot"] = o.DataRoot
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("template execute failed: %s", err.Error())
	}
	return buffer.String(), nil
}

func (o *RegistryOptions) Install() error {
	if err := o.processPackage(); err != nil {
		return fmt.Errorf("process package error: %s", err.Error())
	}

	if err := o.deployRegistry(); err != nil {
		return fmt.Errorf("deploy registry error: %s", err.Error())
	}

	// load images
	if err := o.loadImages(); err != nil {
		return fmt.Errorf("load images error: %s", err.Error())
	}

	// remove tmp file
	if err := o.removePkg(); err != nil {
		return fmt.Errorf("remove pkg error: %s", err.Error())
	}

	o.printUsage()
	return nil
}

func (o *RegistryOptions) printUsage() {
	success := "deploy registry and push images successfully.\nthere is some tips to visit your repository: \n"
	usage1 := fmt.Sprintf("\t1. visit http://%s/v2/_catalog\n", o.registry())
	usage2 := ""
	if o.RegistryPort != 5000 {
		usage2 = fmt.Sprintf("\t2. run cmd [kcctl registry list --node %s --registry-port %v --type repository]", o.Node, o.RegistryPort)
	} else {
		usage2 = fmt.Sprintf("\t2. run cmd [kcctl registry list --node %s --type repository]", o.Node)
	}
	fmt.Printf("\033[1;36;36m%s\033[0m\n", success+usage1+usage2)
}

func (o *RegistryOptions) Uninstall() error {
	// clean registry container
	err := o.stopRegistry()
	if err != nil {
		return err
	}

	// clean registry volume and kc package
	err = o.cleanRegistry()
	if err != nil {
		return err
	}
	logger.Info("registry uninstall successfully")
	return nil
}

func (o *RegistryOptions) cleanRegistry() error {
	c, err := o.readConfig()
	if err != nil {
		logger.Warn("read registry conf failed, use default value,err: ", err)
	} else {
		o.DataRoot = c.Storage.Filesystem.Rootdirectory
	}

	// clean registry volume and kc package
	cmdList := []string{
		fmt.Sprintf(`rm -rf %s`, o.DataRoot),
		"rm -rf /etc/kubeclipper-registry",
	}
	for _, cmd := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	return nil
}

type Configuration struct {
	// Version is the version which defines the format of the rest of the configuration
	Version string `yaml:"version"`

	// Storage is the configuration for the registry's storage driver
	Storage struct {
		Filesystem struct {
			Rootdirectory string `yaml:"rootdirectory"`
		} `yaml:"filesystem"`
	} `yaml:"storage"`
}

func (o *RegistryOptions) readConfig() (Configuration, error) {
	var c Configuration
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, "cat /etc/kubeclipper-registry/kubeclipper-registry.yaml")
	if err != nil {
		return c, err
	}
	if err = ret.Error(); err != nil {
		return c, err
	}

	if err = yaml.Unmarshal([]byte(ret.Stdout), &c); err != nil {
		return c, err
	}
	return c, nil
}

func (o *RegistryOptions) stopRegistry() error {
	cmdList := []string{
		"systemctl disable kc-registry --now",
		"rm -rf /usr/lib/systemd/system/kc-registry.service",
		"systemctl reset-failed kc-registry || true",
		"rm -rf /usr/local/bin/registry",
	}

	for _, cmd := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (o *RegistryOptions) Push() error {
	logger.Info("process package")
	err := utils.SendPackageLocal(o.Pkg, config.DefaultPkgPath, nil)
	if err != nil {
		return err
	}
	fullPath := path.Join(config.DefaultPkgPath, path.Base(o.Pkg))

	logger.Info("unzip package")
	if err = sshutils.Cmd("gzip", "-df", fullPath); err != nil {
		return err
	}
	fullPath = strings.ReplaceAll(fullPath, ".tar.gz", ".tar")

	logger.V(3).Infof("push %s to %s", fullPath, o.registry())
	logger.Info("waiting for push image")
	if err = client.Push(fullPath, o.registry()); err != nil {
		return err
	}
	logger.Info("push image successful")

	return o.removePushPkg()
}

func (o *RegistryOptions) List() error {
	var err error
	switch o.Type {
	case "image":
		err = o.listImages()
	case "repository":
		err = o.listRepositories()
	}
	return err
}

func (o *RegistryOptions) Delete() error {
	if o.Tag == "" {
		return errors.New("missing required arguments: 'tag'")
	}
	return client.Delete(o.registry(), o.Name, o.Tag)
}

func (o *RegistryOptions) listRepositories() error {
	catalog, err := client.Catalog(o.registry())
	if err != nil {
		return err
	}
	repository := &Repositories{
		Repositories: catalog,
	}
	return o.PrintFlags.Print(repository, o.IOStreams.Out)
}

// healthz check is the node or port ok
func (o *RegistryOptions) healthz(log bool) bool {
	url := fmt.Sprintf("http://%s:%d/v2/", o.Node, o.RegistryPort)
	_, code, respErr := httputil.CommonRequest(url, "GET", nil, nil, nil)
	if respErr != nil {
		if !log {
			return false
		}
		logger.Error("health check failed,err:", respErr)
		// if request failed and port is default，print a hit for user to specify registry-port
		if isConnectErr(respErr) && o.RegistryPort == NewRegistryOptions(o.IOStreams).RegistryPort {
			logger.Infof("connect failed, maybe the default registry port(%v) is wrong,if you used custom port when deploy,you can used --registry-port to specify it", o.RegistryPort)
		}
		return false
	}
	logger.V(2).Infof("registry health check,url [%s],http code is:%v", url, code)
	return code == http.StatusOK
}

func isConnectErr(err error) bool {
	return strings.Contains(err.Error(), "connect: connection refused")
}

func (o *RegistryOptions) listImages() error {
	tags, err := client.ListTags(o.registry(), o.Name)
	if err != nil {
		return err
	}
	image := &Image{
		Name: o.Name,
		Tags: tags,
	}
	return o.PrintFlags.Print(image, o.IOStreams.Out)
}

func (o *RegistryOptions) processPackage() error {
	hook := fmt.Sprintf("rm -rf %s/kc && tar -xvf %s -C %s", config.DefaultPkgPath,
		filepath.Join(config.DefaultPkgPath, path.Base(o.Pkg)), config.DefaultPkgPath)
	logger.V(3).Info("processPackage hook:", hook)
	if err := utils.SendPackageLocal(o.Pkg, config.DefaultPkgPath, &hook); err != nil {
		return err
	}

	logger.V(3).Info("send binary file")
	file := fmt.Sprintf("%s/kc/bin/registry", config.DefaultPkgPath)
	chmod := "chmod +x /usr/local/bin/registry"
	err := utils.SendPackage(o.SSHConfig, file, []string{o.Node}, "/usr/local/bin", nil, &chmod)
	if err != nil {
		return err
	}
	logger.Info("process package successfully")
	return nil
}

func (o *RegistryOptions) loadImages() error {
	if o.SkipImageLoad {
		logger.Info("skip image load")
		return nil
	}
	// TODO: for historical reasons and version stability, kcctl temporarily skips the load of the kc-extension image when deploying a private repository
	// e.g. find /tmp/kc/resource -name images*.tar.gz ｜ grep -v kc-extension  | awk '{print}'

	findPkg := fmt.Sprintf("find %s/kc/resource -name images*.tar.gz | grep -v kc-extension | awk '{print}'", config.DefaultPkgPath)
	logger.V(3).Info("loadImages hook :", findPkg)
	ret, err := sshutils.RunCmdAsSSH(findPkg)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}

	logger.V(4).Info("find pkg out :", ret.Stdout)
	pkgs := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("pkg count:", len(pkgs))
	logger.V(4).Info("pkg list:", pkgs)
	for _, pkg := range pkgs {
		if pkg == "" {
			continue
		}
		unzip := fmt.Sprintf("gzip -df %s", pkg)
		ret, err = sshutils.RunCmdAsSSH(unzip)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
		tar := strings.ReplaceAll(pkg, ".tar.gz", ".tar")

		logger.V(3).Infof("push %s to %s", tar, o.registry())
		if err = client.Push(tar, o.registry()); err != nil {
			return err
		}
	}

	logger.Info("image push successfully")
	return nil
}

func (o *RegistryOptions) registry() string {
	return fmt.Sprintf("%s:%v", o.Node, o.RegistryPort)
}

func (o *RegistryOptions) removePkg() error {
	hook := fmt.Sprintf(`rm -rf %s/kc %s`, config.DefaultPkgPath, path.Join(config.DefaultPkgPath, o.Pkg))
	ret, err := sshutils.RunCmdAsSSH(hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.Info("remove pkg successfully")
	return nil
}

func (o *RegistryOptions) removePushPkg() error {
	hook := fmt.Sprintf(`rm -rf %s/kc %s %s`,
		config.DefaultPkgPath,
		path.Join(config.DefaultPkgPath, o.Pkg),
		strings.Replace(path.Join(config.DefaultPkgPath, o.Pkg), ".tar.gz", ".tar", 1))
	ret, err := sshutils.RunCmdAsSSH(hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.Info("remove pkg successfully")
	return nil
}

func (o *RegistryOptions) listTags(toComplete string) []string {
	if o.Name == "" {
		return nil
	}
	tags, err := client.ListTags(o.registry(), o.Name)
	if err != nil {
		logger.V(2).Warnf("list tags error: %s", err.Error())
	}
	set := sets.NewString()
	for _, v := range tags {
		if strings.HasPrefix(v, toComplete) {
			set.Insert(v)
		}
	}
	return set.List()
}

func (o *RegistryOptions) listRepos(toComplete string) []string {
	repositories, err := client.Catalog(o.registry())
	if err != nil {
		logger.V(2).Warnf("list repositories error: %s", err.Error())
		return nil
	}
	set := sets.NewString()
	for _, value := range repositories {
		if strings.HasPrefix(value, toComplete) {
			set.Insert(value)
		}
	}
	return set.List()
}
