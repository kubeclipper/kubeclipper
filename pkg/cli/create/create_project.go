package create

import (
	"context"
	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/cli/printer"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
	"github.com/spf13/cobra"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	projectLongDescription = `
  Create project using command line`
	createProjectExample = `
  # Create project
  kcctl create project -p project-1 -m admin

  # Create cluster with description
  kcctl create project -p project-2 -m admin -d description-detail

  Please read 'kcctl create project -h' get more create project flags.`
)

type CreateProjectOptions struct {
	BaseOptions
	ProjectName    string
	ProjectManager string
	Description    string
}

func NewCreateProjectOptions(streams options.IOStreams) *CreateProjectOptions {
	return &CreateProjectOptions{
		BaseOptions: BaseOptions{
			PrintFlags: printer.NewPrintFlags(),
			CliOpts:    options.NewCliOptions(),
			IOStreams:  streams,
		},
	}
}

func NewCmdCreateProject(streams options.IOStreams) *cobra.Command {
	o := NewCreateProjectOptions(streams)
	cmd := &cobra.Command{
		Use:                   "project (--project-name) <project_name> (--project-manager) <project_manager> [flags]",
		DisableFlagsInUseLine: true,
		Short:                 "create kubeclipper project",
		Long:                  projectLongDescription,
		Example:               createProjectExample,
		Args:                  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckErr(o.Complete())
			utils.CheckErr(o.PreRun())
			utils.CheckErr(o.Run())
		},
	}

	cmd.Flags().StringVarP(&o.ProjectName, "project-name", "p", o.ProjectName, "project name")
	cmd.Flags().StringVarP(&o.ProjectManager, "project-manager", "m", o.ProjectManager, "project manager")
	cmd.Flags().StringVarP(&o.Description, "description", "d", o.Description, "project description")
	o.CliOpts.AddFlags(cmd.Flags())
	o.PrintFlags.AddFlags(cmd)

	return cmd
}

func (o *CreateProjectOptions) Complete() error {
	if err := o.CliOpts.Complete(); err != nil {
		return err
	}
	c, err := kc.FromConfig(o.CliOpts.ToRawConfig())
	if err != nil {
		return err
	}
	o.Client = c
	return nil
}

func (o *CreateProjectOptions) PreRun() error {
	_, err := o.Client.DescribeUser(context.TODO(), o.ProjectManager)
	if !apimachineryErrors.IsNotFound(err) {
		return err
	}
	_, err = o.Client.DescribeProjects(context.TODO(), o.ProjectName)
	if !apimachineryErrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (o *CreateProjectOptions) Run() error {
	newProject := &v1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.ProjectName,
		},
		Spec: v1.ProjectSpec{
			Manager: o.ProjectManager,
		},
	}
	if o.Description != "" {
		newProject.Annotations = map[string]string{
			common.AnnotationDescription: o.Description,
		}
	}
	_, err := o.Client.CreateProject(context.TODO(), newProject)
	return err
}
