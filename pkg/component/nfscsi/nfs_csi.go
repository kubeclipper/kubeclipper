package nfscsi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/common"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/component/validation"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	tmplutil "github.com/kubeclipper/kubeclipper/pkg/utils/template"
)

var (
	errEmptyServerAddr   = errors.New("NFS server address must be provided")
	errInvalidServerAddr = errors.New("invalid NFS server address")
	errEmptySharedPath   = errors.New("NFS shared path must be provided")
	errInvalidSharedPath = errors.New("invalid NFS shared path")
)

const (
	nfs                   = "nfs"
	name                  = "nfs-csi"
	version               = "v1"
	AgentImageLoader      = "ImageLoader"
	defaultSCName         = "nfs-sc"
	defaultReclaimPolicy  = "Delete"
	defaultNamespace      = "kube-system"
	defaultManifestsDir   = "/tmp/.nfs-csi"
	defaultFilenameFormat = "nfs-csi-%s.yaml"
)

func init() {
	n := &NFS{}
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
	_ component.Interface      = (*NFS)(nil)
	_ component.TemplateRender = (*NFS)(nil)
)

type NFS struct {
	ImageRepoMirror                            string   `json:"imageRepoMirror"` // optional
	Namespace                                  string   `json:"namespace"`       // optional
	Replicas                                   int      `json:"replicas"`
	ManifestsDir                               string   `json:"manifestsDir"`  // optional
	ServerAddr                                 string   `json:"serverAddr"`    // required
	SharedPath                                 string   `json:"sharedPath"`    // required
	StorageClassName                           string   `json:"scName"`        // optional
	IsDefault                                  bool     `json:"isDefaultSC"`   // optional
	ReclaimPolicy                              string   `json:"reclaimPolicy"` // optional
	MountOptions                               []string `json:"mountOptions"`  // optional
	KubeletRootDir                             string   `json:"kubeletRootDir"`
	installSteps, uninstallSteps, upgradeSteps []v1.Step
}

func (n NFS) GetComponentMeta(lang component.Lang) component.Meta {
	loc := component.GetLocalizer(lang)

	f := component.JSON(false)
	sc := component.JSON(defaultSCName)

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
			Default:      component.JSON(defaultReclaimPolicy),
			Description:  "Storage Class reclaim policy",
			Priority:     6,
			Dependencies: []string{"enabled"},
			EnumNames:    []string{"Retain", "Delete"},
			Enum:         []component.JSON{"Retain", "Delete"},
		},
		"mountOptions": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.mountOptions"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeArray,
			Default:      nil,
			Description:  "mount options (e.g. 'nfsvers=3')",
			Priority:     7,
			Dependencies: []string{"enabled"},
		},
		"replicas": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.replicas"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeInt,
			Default:      component.JSON(1),
			Description:  "nfs provisioner replicas. It should only run on the master nodes",
			Priority:     8,
			Dependencies: []string{"enabled"},
		},
		"imageRepoMirror": {
			Title:        loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs.imageRepoMirror"}),
			Properties:   nil,
			Type:         component.JSONSchemaTypeString,
			Default:      nil,
			Description:  "nfs image repository mirror, the component official repository is used by default",
			Priority:     9,
			Dependencies: []string{"enabled"},
		},
	}

	return component.Meta{
		Title:      loc.MustLocalize(&i18n.LocalizeConfig{MessageID: "nfs-csi.metaTitle"}),
		Name:       name,
		Version:    version,
		Unique:     true,
		Template:   true,
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

func (n *NFS) Render(ctx context.Context, opts component.Options) error {
	if err := os.MkdirAll(n.ManifestsDir, 0755); err != nil {
		return err
	}
	manifestsFile := filepath.Join(n.ManifestsDir, fmt.Sprintf(defaultFilenameFormat, n.StorageClassName))
	return fileutil.WriteFileWithContext(ctx, manifestsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
		n.renderTo, opts.DryRun)
}

func (n *NFS) renderTo(w io.Writer) error {
	at := tmplutil.New()
	_, err := at.RenderTo(w, csiControllerTemplate, n)
	return err
}

func (n *NFS) NewInstance() component.ObjectMeta {
	return &NFS{
		Namespace:        defaultNamespace,
		ManifestsDir:     defaultManifestsDir,
		StorageClassName: defaultSCName,
		ReclaimPolicy:    defaultReclaimPolicy,
	}
}

func (n *NFS) Ns() string {
	return ""
}

func (n *NFS) Svc() string {
	return ""
}

func (n *NFS) RequestPath() string {
	return ""
}

func (n *NFS) Supported() bool {
	return false
}

func (n *NFS) GetInstanceName() string {
	return n.StorageClassName
}

func (n *NFS) GetDependence() []string {
	return []string{component.InternalCategoryKubernetes}
}

func (n *NFS) RequireExtraCluster() []string {
	return nil
}

func (n *NFS) CompleteWithExtraCluster(extra map[string]component.ExtraMetadata) error {
	return nil
}

func (n *NFS) Validate() error {
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

func (n *NFS) InitSteps(ctx context.Context) error {
	metadata := component.GetExtraMetadata(ctx)
	n.Replicas = len(metadata.Masters.GetNodeIDs())
	n.KubeletRootDir = metadata.KubeletDataDir
	// when the component does not specify an ImageRepoMirror, the cluster LocalRegistry is inherited
	if n.ImageRepoMirror == "" {
		n.ImageRepoMirror = metadata.LocalRegistry
	}
	if metadata.Offline && n.ImageRepoMirror == "" {
		// TODO: arch is unnecessary, version can be configured
		imager := &common.Imager{
			PkgName: nfs,
			Version: "v4.1.0",
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
		Name:       "renderNFSCSIManifests",
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
			Name:       "deployNFSProvisioner",
			Timeout:    metav1.Duration{Duration: 30 * time.Second},
			ErrIgnore:  false,
			RetryTimes: 1,
			Nodes:      master,
			Action:     v1.ActionInstall,
			Commands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"kubectl", "apply", "-f", filepath.Join(n.ManifestsDir, fmt.Sprintf(defaultFilenameFormat, n.StorageClassName))},
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

	if metadata.OperationType != v1.OperationDeleteCluster {
		n.uninstallSteps = append(n.uninstallSteps, []v1.Step{
			rs,
			{
				ID:         strutil.GetUUID(),
				Name:       "removeNFSProvisioner",
				Timeout:    metav1.Duration{Duration: 10 * time.Minute},
				ErrIgnore:  true,
				RetryTimes: 1,
				Nodes:      master,
				Action:     v1.ActionInstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"kubectl", "delete", "-f", filepath.Join(n.ManifestsDir, fmt.Sprintf(defaultFilenameFormat, n.StorageClassName))},
					},
				},
			},
		}...)
	}

	return nil
}

func (n *NFS) GetInstallSteps() []v1.Step {
	return n.installSteps
}

func (n *NFS) GetUninstallSteps() []v1.Step {
	return n.uninstallSteps
}

func (n *NFS) GetUpgradeSteps() []v1.Step {
	return n.upgradeSteps
}

func (n *NFS) Install(ctx context.Context) error {
	return nil
}

func (n *NFS) UnInstall(ctx context.Context) error {
	return nil
}

// GetImageRepoMirror return ImageRepoMirror
func (n *NFS) GetImageRepoMirror() string {
	return n.ImageRepoMirror
}

func initI18nForComponentMeta() error {
	return component.AddI18nMessages(component.I18nMessages{
		{
			ID:      "nfs-csi.metaTitle",
			English: "NFS CSI Setting",
			Chinese: "NFS CSI 设置",
		},
	})
}
