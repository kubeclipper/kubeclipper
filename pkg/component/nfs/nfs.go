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

package nfsprovisioner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component/common"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/component/validation"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

func init() {
	n := &NFSProvisioner{}
	if err := component.Register(fmt.Sprintf(component.RegisterFormat, name, version), n); err != nil {
		panic(err)
	}

	if err := component.RegisterTemplate(fmt.Sprintf(component.RegisterTemplateKeyFormat, name, version, nfs), n); err != nil {
		panic(err)
	}

	if err := initI18nForComponentMeta(); err != nil {
		panic(err)
	}
}

var (
	_ component.Interface      = (*NFSProvisioner)(nil)
	_ component.TemplateRender = (*NFSProvisioner)(nil)
)

const (
	nfs              = "nfs"
	name             = "nfs-provisioner"
	version          = "v1"
	namespace        = "kube-system"
	manifestsDir     = "/tmp/.nfs"
	scName           = "nfs-sc"
	filenameFormat   = "nfsprovisioner-%s.yaml"
	reclaimPolicy    = "Delete"
	AgentImageLoader = "ImageLoader"
)

var (
	errEmptyServerAddr   = errors.New("NFS server address must be provided")
	errInvalidServerAddr = errors.New("invalid NFS server address")
	errEmptySharedPath   = errors.New("NFS shared path must be provided")
	errInvalidSharedPath = errors.New("invalid NFS shared path")
)

type NFSProvisioner struct {
	ImageRepoMirror  string `json:"imageRepoMirror"` // optional
	Namespace        string `json:"namespace"`       // optional
	Replicas         int    `json:"replicas"`
	ManifestsDir     string `json:"manifestsDir"`    // optional
	ServerAddr       string `json:"serverAddr"`      // required
	SharedPath       string `json:"sharedPath"`      // required
	StorageClassName string `json:"scName"`          // optional
	IsDefault        bool   `json:"isDefaultSC"`     // optional
	ReclaimPolicy    string `json:"reclaimPolicy"`   // optional
	ArchiveOnDelete  bool   `json:"archiveOnDelete"` // optional
	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with these mountOptions, e.g. ["ro", "soft"].
	MountOptions                               []string `json:"mountOptions"` // optional
	installSteps, uninstallSteps, upgradeSteps []v1.Step
}

func (n *NFSProvisioner) Ns() string {
	return ""
}

func (n *NFSProvisioner) Svc() string {
	return ""
}

func (n *NFSProvisioner) RequestPath() string {
	return ""
}

// Supported TODO: add nfs healthy condition check
func (n *NFSProvisioner) Supported() bool {
	return false
}

func (n *NFSProvisioner) GetInstanceName() string {
	return n.StorageClassName
}

func (n *NFSProvisioner) RequireExtraCluster() []string {
	return nil
}

func (n *NFSProvisioner) CompleteWithExtraCluster(extra map[string]component.ExtraMetadata) error {
	return nil
}

func (n *NFSProvisioner) Validate() error {
	// namespace
	if !validation.MatchKubernetesNamespace(n.Namespace) {
		return validation.ErrInvalidNamespace
	}
	// server address
	if n.ServerAddr == "" {
		return errEmptyServerAddr
	}
	if !validation.IsHostNameRFC952(n.ServerAddr) && !netutil.IsValidIP(n.ServerAddr) {
		return errInvalidServerAddr
	}
	// shared path
	if n.SharedPath == "" {
		return errEmptySharedPath
	}
	if !validation.MatchLinuxFilePath(n.SharedPath) {
		return errInvalidSharedPath
	}
	// storage class name
	if !validation.MatchKubernetesStorageClass(n.StorageClassName) {
		return validation.ErrInvalidSCName
	}
	// reclaim policy
	return validation.MatchKubernetesReclaimPolicy(n.ReclaimPolicy)
}

func (n *NFSProvisioner) InitSteps(ctx context.Context) error {
	metadata := component.GetExtraMetadata(ctx)
	n.Replicas = len(metadata.Masters.GetNodeIDs())
	// when the component does not specify an ImageRepoMirror, the cluster LocalRegistry is inherited
	if n.ImageRepoMirror == "" {
		n.ImageRepoMirror = metadata.LocalRegistry
	}
	if metadata.Offline && n.ImageRepoMirror == "" {
		// TODO: arch is unnecessary, version can be configured
		imager := &common.Imager{
			PkgName: nfs,
			Version: "v4.0.2",
			CriName: metadata.CRI,
			Offline: metadata.Offline,
		}
		steps, err := imager.InstallSteps(metadata.GetAllNodes())
		if err != nil {
			return err
		}
		n.installSteps = append(n.installSteps, steps...)
	}

	bytes, err := json.Marshal(n)
	if err != nil {
		return err
	}

	master := utils.UnwrapNodeList(metadata.Masters[:1])
	if len(metadata.Masters) > 1 && metadata.ClusterStatus == v1.ClusterRunning {
		avaMasters, err := metadata.Masters.AvailableKubeMasters()
		if err != nil {
			return err
		}
		master = utils.UnwrapNodeList(avaMasters[:1])
	}
	rs := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "renderNFSProvisionerManifests",
		Timeout:    metav1.Duration{Duration: 3 * time.Second},
		ErrIgnore:  true,
		RetryTimes: 1,
		Nodes:      master,
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type: v1.CommandTemplateRender,
				Template: &v1.TemplateCommand{
					Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, name, version, nfs),
					Data:     bytes,
				},
			},
		},
	}

	n.installSteps = append(n.installSteps, []v1.Step{
		rs,
		{
			ID:         strutil.GetUUID(),
			Name:       "deployNFSProvisionerNameSpace",
			Timeout:    metav1.Duration{Duration: 30 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, fmt.Sprintf(filenameFormat, namespace))},
				},
			},
		},
		{
			ID:         strutil.GetUUID(),
			Name:       "deployNFSProvisioner",
			Timeout:    metav1.Duration{Duration: 30 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, fmt.Sprintf(filenameFormat, n.StorageClassName))},
				},
			},
		},
	}...)

	c := new(common.CSIHealthCheck)
	checkCSIHealthStep, err := c.GetCheckCSIHealthStep(master, n.StorageClassName)
	if err != nil {
		return err
	}
	n.installSteps = append(n.installSteps, checkCSIHealthStep...)

	// uninstall
	if metadata.OperationType != v1.OperationDeleteCluster {
		n.uninstallSteps = []v1.Step{
			rs,
			{
				ID:         strutil.GetUUID(),
				Name:       "removeNFSProvisioner",
				Timeout:    metav1.Duration{Duration: 10 * time.Minute},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes:      master,
				Action:     v1.ActionUninstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"kubectl", "delete", "-f", filepath.Join(n.ManifestsDir, fmt.Sprintf(filenameFormat, n.StorageClassName))},
					},
				},
			},
		}
	}

	return nil
}

func (n *NFSProvisioner) GetName() string {
	return name
}

func (n *NFSProvisioner) GetVersion() string {
	return version
}

func (n *NFSProvisioner) GetComponentMeta(lang component.Lang) component.Meta {
	loc := component.GetLocalizer(lang)

	f := component.JSON(false)
	sc := component.JSON(scName)

	propMap := map[string]component.JSONSchemaProps{
		"serverAddr": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.serverAddr"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      nil,
			Description:  "NFS server address",
			Priority:     2,
			Dependencies: []string{"enabled"},
		},
		"sharedPath": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.sharedPath"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      nil,
			Description:  "NFS shared path",
			Priority:     3,
			Dependencies: []string{"enabled"},
		},
		"scName": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.scName"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      &sc,
			Description:  "Storage Class name",
			Priority:     4,
			Dependencies: []string{"enabled"},
		},
		"isDefaultSC": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.isDefaultSC"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeBool,
			Default:      &f,
			Description:  "set as default Storage Class",
			Priority:     5,
			Dependencies: []string{"enabled"},
		},
		"reclaimPolicy": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.reclaimPolicy"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      component.JSON(reclaimPolicy),
			Description:  "Storage Class reclaim policy",
			Priority:     6,
			Dependencies: []string{"enabled"},
			EnumNames:    []string{"Retain", "Delete"},
			Enum:         []component.JSON{"Retain", "Delete"},
		},
		"archiveOnDelete": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.archiveOnDelete"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeBool,
			Default:      &f,
			Description:  "archive PVC when deleting",
			Priority:     7,
			Dependencies: []string{"enabled"},
		},
		"mountOptions": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.mountOptions"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeArray,
			Default:      nil,
			Description:  "mount options (e.g. 'nfsvers=3')",
			Priority:     8,
			Dependencies: []string{"enabled"},
		},
		"replicas": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.replicas"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeInt,
			Default:      component.JSON(1),
			Description:  "nfs provisioner replicas. It should only run on the master nodes",
			Priority:     11,
			Dependencies: []string{"enabled"},
		},
		"imageRepoMirror": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.imageRepoMirror"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      nil,
			Description:  "nfs image repository mirror, the component official repository is used by default",
			Priority:     12,
			Dependencies: []string{"enabled"},
		},
	}

	return component.Meta{
		Title:      loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.metaTitle"}),
		Name:       name,
		Version:    version,
		Unique:     false,
		Template:   true,
		Deprecated: true,
		Dependence: []string{component.InternalCategoryKubernetes},
		Category:   component.InternalCategoryStorage,
		Priority:   3,
		Schema: &component.JSONSchemaProps{
			Properties: propMap,
			Required:   []string{"serverAddr", "sharedPath", "scName"},
			Type:       component.JSONSchemaTypeObject,
			Default:    nil,
		},
	}
}

func (n *NFSProvisioner) NewInstance() component.ObjectMeta {
	return &NFSProvisioner{
		Namespace:        namespace,
		ManifestsDir:     manifestsDir,
		StorageClassName: scName,
		ReclaimPolicy:    reclaimPolicy,
	}
}

func (n *NFSProvisioner) GetDependence() []string {
	return []string{component.InternalCategoryKubernetes}
}

func (n *NFSProvisioner) GetInstallSteps() []v1.Step {
	return n.installSteps
}

func (n *NFSProvisioner) GetUninstallSteps() []v1.Step {
	return n.uninstallSteps
}

func (n *NFSProvisioner) GetUpgradeSteps() []v1.Step {
	return n.upgradeSteps
}

func (n *NFSProvisioner) Install(ctx context.Context) error {
	// TODO:
	return nil
}

func (n *NFSProvisioner) UnInstall(ctx context.Context) error {
	// TODO:
	return nil
}

func (n *NFSProvisioner) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, manifestsTemplate, n)
	return err
}

func (n *NFSProvisioner) renderNameSpace(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, nameSpaceTemplate, n)
	return err
}

func (n *NFSProvisioner) Render(ctx context.Context, opts component.Options) error {
	// storage namespace
	n.Namespace = namespace
	if err := os.MkdirAll(n.ManifestsDir, 0755); err != nil {
		return err
	}
	nameSpace := filepath.Join(n.ManifestsDir, fmt.Sprintf(filenameFormat, namespace))
	if err := fileutil.WriteFileWithContext(ctx, nameSpace, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderNameSpace, opts.DryRun); err != nil {
		return err
	}
	manifestsFile := filepath.Join(n.ManifestsDir, fmt.Sprintf(filenameFormat, n.StorageClassName))
	return fileutil.WriteFileWithContext(ctx, manifestsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderTo, opts.DryRun)
}

// GetImageRepoMirror return ImageRepoMirror
func (n *NFSProvisioner) GetImageRepoMirror() string {
	return n.ImageRepoMirror
}
