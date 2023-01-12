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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"

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
  # Clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111
  # Push docker image to registry
  kcctl registry push --pk-file key --node 10.0.0.111 --images-pkg images.tar.gz
  # List docker registry
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
  kcctl registry deploy --pk-file key --node 10.0.0.111 --pkg kc.tar.gz --registry-volume /opt/myregistry --data-root /var/lib/mydocker
  # Deploy docker registry specify port
  # If you used custom port,then must specify it in push、list cmd.
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
  # Clean docker registry and uninstall docker
  kcctl registry clean --pk-file key --node 10.0.0.111 --remove-docker
  # Forced to clean docker registry
  kcctl registry clean --pk-file key --node 10.0.0.111  --force

  Please read 'kcctl registry clean -h' get more registry clean flags.`
	pushLongDescription = `
  Push docker image by flags.`
	pushExample = `
  # Push a docker image to registry
  # You can use docker save > xxx to generate image pkg
  kcctl registry push --pk-file key --node 10.0.0.111  --images-pkg images.tar.gz

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
	Client     *kc.Client

	Deploy string
	Clean  string

	Node string
	Pkg  string

	DataRoot       string
	RegistryVolume string
	RegistryPort   int
	Arch           string

	// no install/uninstall docker
	RemoveDocker bool
	Force        bool

	Type   string
	Name   string
	Tag    string
	Number int

	SSHConfig *sshutils.SSH
}

var (
	allowType = sets.NewString("image", "repository")
)

func NewRegistryOptions(streams options.IOStreams) *RegistryOptions {
	return &RegistryOptions{
		IOStreams:      streams,
		PrintFlags:     printer.NewPrintFlags(),
		SSHConfig:      sshutils.NewSSH(),
		DataRoot:       "/var/lib/docker",
		RegistryVolume: "/opt/registry",
		RegistryPort:   5000,
		Arch:           "amd64",
		Tag:            "",
		Number:         0,
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
		Use:                   "deploy (--user | -u <user>) (--passwd <passwd>) (--pk-file <pk-file>) (--pk-passwd <pk-passwd>) (--node <node>) (--arch <arch>) (--pkg <pkg>) (--data-root <data-root>) (--registry-volume <registry-volume>) (--registry-port <registry-port>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry deploy",
		Long:                  deployLongDescription,
		Example:               deployExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgsDeploy())
			if !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Install())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Arch, "arch", o.Arch, "registry arch.")
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	cmd.Flags().StringVar(&o.Pkg, "pkg", o.Pkg, "docker service and images pkg.")
	cmd.Flags().StringVar(&o.DataRoot, "data-root", o.DataRoot, "set docker data-root value.")
	cmd.Flags().StringVar(&o.RegistryVolume, "registry-volume", o.RegistryVolume, "set registry volume path")
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "set registry container port")

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("pkg"))
	return cmd
}

func NewCmdRegistryClean(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "clean (--node <node>) (--data-root <data-root>) (--registry-port <registry-port>) (--remove-docker <remove-docker>) (--force <force>) [flags]clean (--node <node>) (--data-root <data-root>) (--registry-port <registry-port>) (--remove-docker <remove-docker>) (--force <force>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry clean",
		Long:                  cleanLongDescription,
		Example:               cleanExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgs())
			if !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Uninstall())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	cmd.Flags().StringVar(&o.DataRoot, "data-root", o.DataRoot, "clean docker data-root value.")
	cmd.Flags().StringVar(&o.RegistryVolume, "registry-volume", o.RegistryVolume, "clean registry volume path")
	cmd.Flags().BoolVar(&o.RemoveDocker, "remove-docker", o.RemoveDocker, "no uninstall docker")
	cmd.Flags().BoolVar(&o.Force, "force", o.Force, "force uninstall")

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	return cmd
}

func NewCmdRegistryPush(o *RegistryOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "push (--node <node>) (--arch <arch>) (--registry-port <registry-port>) (--images-pkg <images-pkg>) [flags]push (--node <node>) (--arch <arch>) (--registry-port <registry-port>) (--images-pkg <images-pkg>) [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "registry push image",
		Long:                  pushLongDescription,
		Example:               pushExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgsPush())
			if !o.healthPreCheck() || !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Push())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
	cmd.Flags().StringVar(&o.Node, "node", o.Node, "registry node.")
	cmd.Flags().StringVar(&o.Pkg, "images-pkg", o.Pkg, "docker images pkg,use `docker save $containerIDs > images.tar && gzip -f images.tar` to generate images.tar.gz")
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "set registry container port")

	utils.CheckErr(cmd.MarkFlagRequired("node"))
	utils.CheckErr(cmd.MarkFlagRequired("images-pkg"))
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
			utils.CheckErr(o.Complete())
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
	cmd.Flags().IntVar(&o.RegistryPort, "registry-port", o.RegistryPort, "set registry container port")
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
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.ValidateArgsDelete(cmd))
			if !o.sudoPreCheck() {
				return
			}
			utils.CheckErr(o.Delete())
		},
	}

	options.AddFlagsToSSH(o.SSHConfig, cmd.Flags())
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
	return o.healthz()
}

func (o *RegistryOptions) sudoPreCheck() bool {
	return sudo.PreCheck("sudo", o.SSHConfig, o.IOStreams, []string{o.Node})
}

func (o *RegistryOptions) Complete() error {
	if o.Arch == "" {
		o.Arch = "amd64"
	}
	opts := options.NewCliOptions()
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
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
	}
	if o.Node == "" {
		return fmt.Errorf("--node must be specified")
	}
	if o.Pkg == "" {
		return fmt.Errorf("--image-pkg must be specified")
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
	if o.SSHConfig.PkFile == "" && o.SSHConfig.Password == "" {
		return fmt.Errorf("one of --pk-file or --passwd must be specified")
	}
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

func (o *RegistryOptions) Install() error {
	if err := o.processPackage(); err != nil {
		return fmt.Errorf("process package error: %s", err.Error())
	}

	if err := o.installDocker(); err != nil {
		return fmt.Errorf("install docker error: %s", err.Error())
	}

	if err := o.installRegistry(); err != nil {
		return fmt.Errorf("install registry error: %s", err.Error())
	}

	// load images
	if err := o.loadImages(); err != nil {
		return fmt.Errorf("load images error: %s", err.Error())
	}

	// remove pkg
	if err := o.removePkg(); err != nil {
		return fmt.Errorf("remove pkg error: %s", err.Error())
	}

	if err := o.push(); err != nil {
		return fmt.Errorf("push images error: %s", err.Error())
	}

	logger.Info("registry and images install successfully")
	return nil
}

func (o *RegistryOptions) Uninstall() error {
	// dockerd or docker sometimes gets stuck
	if o.Force {
		err := o.killDocker()
		if err != nil {
			return err
		}
	}

	// clean registry container
	err := o.stopRegistry()
	if err != nil {
		return err
	}

	// remove docker if you want
	if o.RemoveDocker {
		err := o.cleanDocker()
		if err != nil {
			return err
		}
	}

	// clean registry volume and kc package
	err = o.cleanRegistry()
	if err != nil {
		return err
	}
	logger.Info("registry uninstall successfully")
	return nil
}

func (o *RegistryOptions) cleanDocker() error {
	// stop docker service
	cmdList := []string{
		"systemctl disable docker --now",                                          // stop docker
		`rm -rf /usr/bin/docker* /usr/bin/containerd* /usr/bin/ctr /usr/bin/runc`, // remove docker binary
		fmt.Sprintf("rm -rf %s", o.DataRoot),                                      // remote docker data
		`rm -rf /etc/systemd/system/docker.service`,                               // remove docker systemd
		"systemctl reset-failed docker || true",
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

	// remove docker data-root
	hook := "mount | grep /run/docker/netns/default | wc -l"
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}

	if ret.StdoutToString("") == "1" {
		// umount if mounted
		hook = "umount /var/run/docker/netns/default"
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	hook = fmt.Sprintf(`rm -rf /var/run/docker* %s/kc`, o.DataRoot)
	ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	return ret.Error()
}

func (o *RegistryOptions) cleanRegistry() error {
	// clean registry volume and kc package
	cmdList := []string{
		fmt.Sprintf(`rm -rf %s %s/kc*`, o.RegistryVolume, config.DefaultPkgPath),
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

func (o *RegistryOptions) stopRegistry() error {
	hook := `docker stop registry && docker rm registry`
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	return ret.Error()
}

func (o *RegistryOptions) killDocker() error {
	hook := `ps -ef | grep /usr/bin/docker | grep -v color=auto | awk '{print  "kill -9 " $2}'`
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		logger.Warnf("clean registry container error: %s", err.Error())
	}
	if err = ret.Error(); err != nil {
		logger.Warnf("clean registry container error: %s", err.Error())
	}
	logger.V(4).Info("kill docker out:", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("kill docker cmd count:", len(split))
	logger.V(4).Info("kill docker cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
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
	// send image pkg
	imagesPkg := filepath.Join(config.DefaultPkgPath, filepath.Base(o.Pkg))
	err := utils.SendPackageV2(o.SSHConfig, o.Pkg, []string{o.Node}, config.DefaultPkgPath, nil, nil)
	if err != nil {
		return err
	}
	hook := fmt.Sprintf("docker load -i %s && rm -rf %s", imagesPkg, imagesPkg)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	return o.push()
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
	imagePath := fmt.Sprintf("%s/docker/registry/v2/repositories/%s/_manifests/tags/%s", o.RegistryVolume, o.Name, o.Tag)
	if ok, _ := o.SSHConfig.IsFileExistV2(o.Node, imagePath); !ok {
		return errors.New("there is an error in the image name or tag, please check the input")
	}
	hook := fmt.Sprintf("rm -rf %s", imagePath)
	_, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	return err
}

func (o *RegistryOptions) listRepositories() error {
	url := fmt.Sprintf("http://%s:%d/v2/_catalog", o.Node, o.RegistryPort)
	params := make(map[string]string)
	if o.Number != 0 {
		params["n"] = strconv.Itoa(o.Number)
	}
	resp, code, respErr := httputil.CommonRequest(url, "GET", nil, params, nil)
	if respErr != nil {
		return respErr
	}
	body, codeErr := httputil.CodeDispose(resp, code)
	if codeErr != nil {
		return codeErr
	}
	repository := new(Repositories)
	err := json.Unmarshal(body, repository)
	if err != nil {
		return err
	}
	return o.PrintFlags.Print(repository, o.IOStreams.Out)
}

// healthz check is the node or port ok
func (o *RegistryOptions) healthz() bool {
	url := fmt.Sprintf("http://%s:%d/v2", o.Node, o.RegistryPort)
	_, code, respErr := httputil.CommonRequest(url, "GET", nil, nil, nil)
	if respErr != nil {
		logger.Error("health check failed,err:", respErr)
		// if request failed and port is default，print a hit for user to specify registry-port
		if isConnectErr(respErr) && o.RegistryPort == NewRegistryOptions(o.IOStreams).RegistryPort {
			logger.Infof("connect failed, maybe the default registry port(%v) is wrong,if you used custom port when deploy,you can used --registry-port to specify it", o.RegistryPort)
		}
		return false
	}
	logger.V(2).Infof("registry health check,http code is:%v", code)
	return code == http.StatusOK
}

func isConnectErr(err error) bool {
	return strings.Contains(err.Error(), "connect: connection refused")
}

func (o *RegistryOptions) listImages() error {
	url := fmt.Sprintf("http://%s:%d/v2/%s/tags/list", o.Node, o.RegistryPort, o.Name)
	resp, code, respErr := httputil.CommonRequest(url, "GET", nil, nil, nil)
	if respErr != nil {
		return respErr
	}
	body, codeErr := httputil.CodeDispose(resp, code)
	if codeErr != nil {
		return codeErr
	}
	image := new(Image)
	err := json.Unmarshal(body, image)
	if err != nil {
		return err
	}
	return o.PrintFlags.Print(image, o.IOStreams.Out)
}

func (o *RegistryOptions) getDaemonTemplateContent() (string, error) {
	tmpl, err := template.New("text").Parse(config.DockerDaemonTmpl)
	if err != nil {
		return "", fmt.Errorf("template parse failed: %s", err.Error())
	}

	var data = make(map[string]interface{})
	data["Node"] = fmt.Sprintf(`%s:%d`, o.Node, o.RegistryPort)
	data["DataRoot"] = o.DataRoot
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("template execute failed: %s", err.Error())
	}
	return buffer.String(), nil
}

func (o *RegistryOptions) processPackage() error {
	// send pkg
	hook := fmt.Sprintf("rm -rf %s/kc && tar -xvf %s -C %s", config.DefaultPkgPath,
		filepath.Join(config.DefaultPkgPath, path.Base(o.Pkg)), config.DefaultPkgPath)
	logger.V(3).Info("processPackage hook:", hook)
	err := utils.SendPackageV2(o.SSHConfig, o.Pkg, []string{o.Node}, config.DefaultPkgPath, nil, &hook)
	if err != nil {
		return err
	}
	logger.Info("process package successfully")
	return nil
}

func (o *RegistryOptions) installDocker() error {
	// install docker, if not exist
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, `systemctl status docker|grep "active (running)"|wc -l`)
	if err != nil {
		return err
	}
	if ret.StdoutToString("") != "0" { // if already exist,do nothing.
		return nil
	}
	data, err := o.getDaemonTemplateContent()
	if err != nil {
		return err
	}
	metas, err := o.Client.GetComponentMeta(context.TODO(), nil)
	if err != nil {
		return fmt.Errorf("get component meta failed: %s. please check .kc/config", err)
	}
	var dockerVersion string
	for _, v := range metas.Addons {
		if v.Name == "docker" && v.Arch == o.Arch {
			dockerVersion = v.Version
			break
		}
	}
	if dockerVersion == "" {
		return fmt.Errorf("docker version does not meet installation requirements")
	}
	cmdList := []string{
		// cp docker service file
		fmt.Sprintf("tar -zxvf %s/kc/resource/docker/%s/%s/configs.tar.gz -C /", config.DefaultPkgPath, dockerVersion, o.Arch),
		"mkdir -pv /etc/docker",
		// write daemon.json
		sshutils.WrapEcho(data, "/etc/docker/daemon.json"),
		// start docker
		"systemctl daemon-reload && systemctl enable docker --now",
	}
	for _, cmd := range cmdList {
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}

	return nil
}

func (o *RegistryOptions) installRegistry() error {
	cmdList := []string{
		fmt.Sprintf("gzip -df %s/kc/registry/v2/%s/images.tar.gz", config.DefaultPkgPath, o.Arch),
		fmt.Sprintf("docker load -i %s/kc/registry/v2/%s/images.tar", config.DefaultPkgPath, o.Arch), // load images
		fmt.Sprintf("docker run -d -v %s:/var/lib/registry -p %d:5000 --restart=always --name registry registry:2",
			o.RegistryVolume, o.RegistryPort), // running registry
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

	logger.Info("install registry successfully")
	return nil
}

func (o *RegistryOptions) loadImages() error {
	// docker load images
	// find /root/kc/pkg/kc/resource -name images.tar.gz | grep -v kc-extension | grep 'x86-64' | awk '{print}' | sed -r 's#(.*)#docker load -i \1#'
	// TODO: for historical reasons and version stability, kcctl temporarily skips the load of the kc-extension image when deploying a private repository
	hook := fmt.Sprintf("find %s/kc/resource -name images*.tar.gz | grep -v kc-extension | grep '%s' | awk '{print}' | sed -r 's#(.*)#docker load -i \\1#'", config.DefaultPkgPath, o.Arch)
	logger.V(3).Info("loadImages hook :", hook)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.V(4).Info("loadImages out :", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("loadImages out cmd count:", len(split))
	logger.V(4).Info("loadImages out cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}

	logger.Info("image load successfully")
	return nil
}

func (o *RegistryOptions) removePkg() error {
	hook := fmt.Sprintf(`rm -rf %s/kc`, config.DefaultPkgPath)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, hook)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.Info("remove pkg successfully")
	return nil
}

func (o *RegistryOptions) push() error {
	// if image has organization,just retag to 'ip:port/',e.g. kubeclipper/kubectl:latest
	// if image without organization,infact it has default organization library,so we retag to 'ip:port/library/',e.g. nginx:latest,
	// we use `/` to distinguish them.

	// image re-tag 'ip:port/library/'
	err := o.specialTag()
	if err != nil {
		return err
	}
	// image re-tag 'ip:port/'
	err = o.normalTag()
	if err != nil {
		return err
	}

	//  docker push
	err = o.pushImage()
	if err != nil {
		return err
	}

	// docker rmi images
	err = o.removeImage()
	if err != nil {
		return err
	}

	logger.Info("image push successfully")
	return nil
}

func (o *RegistryOptions) removeImage() error {
	rmi := `docker images | awk '{print $1":"$2}' | grep -v registry | grep -v REPOSITORY`
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, rmi)
	if err != nil {
		logger.Warnf("docker remove image error: %s", err.Error())
	}
	if err = ret.Error(); err != nil {
		logger.Warnf("docker remove image error: %s", err.Error())
	}
	logger.V(4).Info("docker rmi out", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("docker rmi cmd count:", len(split))
	logger.V(4).Info("docker rmi cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, "docker rmi "+cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (o *RegistryOptions) pushImage() error {
	push := fmt.Sprintf(`docker images | grep %s:%d | awk '{print "docker push "$1":"$2}'`, o.Node, o.RegistryPort)
	logger.V(3).Info("docker push hook:", push)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, push)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.V(4).Info("docker push out:", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("docker push cmd count:", len(split))
	logger.V(4).Info("docker push cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (o *RegistryOptions) normalTag() error {
	retag := fmt.Sprintf(`docker images | grep / | grep -v k8s.gcr.io | grep -v registry.k8s.io | grep -v %s:%d | grep -v REPOSITORY | awk '{print "docker tag "$3" %s:%d/"$1":"$2}'`, o.Node, o.RegistryPort, o.Node, o.RegistryPort)
	logger.V(3).Info("normalTag hook:", retag)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, retag)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.V(4).Info("normalTag out:", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("normalTag cmd count:", len(split))
	logger.V(4).Info("normalTag cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (o *RegistryOptions) specialTag() error {
	// add 'ip:port/library'
	dockerTag := fmt.Sprintf(`docker images | grep -v registry | grep -v / | grep -v k8s.gcr.io | grep -v registry.k8s.io | grep -v REPOSITORY | awk '{print "docker tag "$3" %s:%d/library/"$1":"$2}'`, o.Node, o.RegistryPort)
	logger.V(3).Info("specialTag hook:", dockerTag)
	ret, err := sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, dockerTag)
	if err != nil {
		return err
	}
	if err = ret.Error(); err != nil {
		return err
	}
	logger.V(4).Info("specialTag out:", ret.Stdout)
	split := strings.Split(ret.Stdout, "\n")
	logger.V(4).Info("specialTag cmd count:", len(split))
	logger.V(4).Info("specialTag cmd list:", split)
	for _, cmd := range split {
		if cmd == "" {
			continue
		}
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}

	// remove tag 'k8s.gcr.io' and 'registry.k8s.io'
	dockerTagCmds := []string{
		fmt.Sprintf(`docker images | grep k8s.gcr.io | sed 's/k8s.gcr.io\///' | awk '{print "docker tag "$3" %s:%d/"$1":"$2}'`, o.Node, o.RegistryPort),
		fmt.Sprintf(`docker images | grep registry.k8s.io | sed 's/registry.k8s.io\///' | awk '{print "docker tag "$3" %s:%d/"$1":"$2}'`, o.Node, o.RegistryPort),
	}
	for _, tagCmd := range dockerTagCmds {
		logger.V(3).Info("dockerTag2 hook:", tagCmd)
		ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, tagCmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
		logger.V(4).Info("dockerTag2 out:", ret.Stdout)
		split = strings.Split(ret.Stdout, "\n")
		logger.V(4).Info("dockerTag2 out cmd count:", len(split))
		logger.V(4).Info("dockerTag2 out cmd list:", split)
		for _, cmd := range split {
			if cmd == "" {
				continue
			}
			ret, err = sshutils.SSHCmdWithSudo(o.SSHConfig, o.Node, cmd)
			if err != nil {
				return err
			}
			if err = ret.Error(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *RegistryOptions) listTags(toComplete string) []string {
	utils.CheckErr(o.Complete())

	if o.Name == "" {
		return nil
	}
	tags, err := o.tags()
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

func (o *RegistryOptions) tags() ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/v2/%s/tags/list", o.Node, o.RegistryPort, o.Name)
	resp, code, respErr := httputil.CommonRequest(url, "GET", nil, nil, nil)
	if respErr != nil {
		return nil, pkgerr.WithMessage(respErr, "request failed")
	}
	body, codeErr := httputil.CodeDispose(resp, code)
	if codeErr != nil {
		return nil, pkgerr.WithMessage(respErr, "code err failed")
	}
	img := new(Image)
	err := json.Unmarshal(body, img)
	return img.Tags, err
}

func (o *RegistryOptions) listRepos(toComplete string) []string {
	utils.CheckErr(o.Complete())
	repositories, err := o.repos()
	if err != nil {
		logger.V(2).Warnf("list repositories error: %s", err.Error())
		return nil
	}
	set := sets.NewString()
	for _, values := range repositories {
		for _, value := range values {
			if strings.HasPrefix(value, toComplete) {
				set.Insert(value)
			}
		}
	}
	return set.List()
}

func (o *RegistryOptions) repos() (map[string][]string, error) {
	url := fmt.Sprintf("http://%s:%d/v2/_catalog", o.Node, o.RegistryPort)
	params := make(map[string]string)
	if o.Number != 0 {
		params["n"] = strconv.Itoa(o.Number)
	}
	resp, code, respErr := httputil.CommonRequest(url, "GET", nil, params, nil)
	if respErr != nil {
		return nil, respErr
	}
	body, codeErr := httputil.CodeDispose(resp, code)
	if codeErr != nil {
		return nil, codeErr
	}
	repository := make(map[string][]string)
	err := json.Unmarshal(body, &repository)
	return repository, err
}
