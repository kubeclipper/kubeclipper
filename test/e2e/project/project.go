package project

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	tenantv1 "github.com/kubeclipper/kubeclipper/pkg/scheme/tenant/v1"
	"github.com/kubeclipper/kubeclipper/test/framework"
)

func CreateProject(f *framework.Framework) error {
	project := &tenantv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "tenant.kubeclipper.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: framework.DefaultE2EProject,
			Annotations: map[string]string{
				common.AnnotationDescription: "project for e2e",
			},
		},
		Spec: tenantv1.ProjectSpec{
			Manager: "admin",
			Nodes:   []string{},
		},
	}

	if oldProject, err := f.Client.DescribeProjects(context.Background(), framework.DefaultE2EProject); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
		// not exist,create it
		if _, err = f.Client.CreateProject(context.Background(), project); err != nil {
			return err
		}
	} else {
		// exist,update it
		project.ResourceVersion = oldProject.Items[0].ResourceVersion
		if _, err = f.Client.UpdateProject(context.Background(), project); err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}
	}
	return nil
}

func DeleteProject(f *framework.Framework) error {
	if err := f.Client.DeleteProject(context.Background(), framework.DefaultE2EProject); err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	return nil
}
