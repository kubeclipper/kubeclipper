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

package k8s

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cni"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
	"github.com/kubeclipper/kubeclipper/pkg/simple/downloader"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/netutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sysutil"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ component.StepRunnable = (*UpgradePackage)(nil)
	_ component.StepRunnable = (*ActBackup)(nil)
	_ component.StepRunnable = (*Recovery)(nil)
)

const (
	upgradePackage = "upgradePackage"
	actBackup      = "actBackup"
	recovery       = "Recovery"
)

func init() {
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, upgradePackage, version, component.TypeStep), &UpgradePackage{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, actBackup, version, component.TypeStep), &ActBackup{}); err != nil {
		panic(err)
	}
	if err := component.RegisterAgentStep(fmt.Sprintf(component.RegisterStepKeyFormat, recovery, version, component.TypeStep), &Recovery{}); err != nil {
		panic(err)
	}
}

type Upgrade struct {
	Kubeadm       *KubeadmConfig `json:"kubeadm"`
	Offline       bool           `json:"offline"`
	Version       string         `json:"version"`
	LocalRegistry string         `json:"localRegistry"`
	installSteps  []v1.Step
}

type UpgradePackage struct {
	*Package
	DownloadImage bool `json:"downloadImage"`
}

type ActBackup struct {
	BackupFileName     string
	BackupPointRootDir string
	StoreType          string
	Bucket             string
	Endpoint           string
	AccessKeyID        string
	AccessKeySecret    string
	Region             string
	SSL                bool

	installSteps   []v1.Step
	uninstallSteps []v1.Step
}

type CheckFile struct {
	BackupFileSize int64
	BackupFileMD5  string
}

type FileDir struct {
	TmpStaticYaml  string
	ManifestsYaml  string
	EtcdDataDir    string
	TmpEtcdDataDir string
}

type Recovery struct {
	NodeNameList       []string
	NodeIPList         []string
	RestoreDir         string
	BackupFileName     string
	BackupPointRootDir string
	StoreType          string
	Bucket             string
	Endpoint           string
	AccessKeyID        string
	AccessKeySecret    string
	Region             string
	SSL                bool
	BackupFileSize     int64
	BackupFileMD5      string
	FileDir

	installSteps []v1.Step
}

type AfterRecovery struct{}

func (stepper *Upgrade) InitStepper(metadata *component.ExtraMetadata, c *v1.Cluster) {
	apiServerDomain := APIServerDomainPrefix + strutil.StringDefaultIfEmpty("cluster.local", c.Networking.DNSDomain)
	cpEndpoint := fmt.Sprintf("%s:6443", apiServerDomain)

	stepper.Kubeadm = &KubeadmConfig{
		ClusterConfigAPIVersion: "",
		ContainerRuntime:        c.ContainerRuntime.Type,
		Etcd:                    c.Etcd,
		Networking:              c.Networking,
		KubeProxy:               c.KubeProxy,
		Kubelet:                 c.Kubelet,
		ClusterName:             metadata.ClusterName,
		KubernetesVersion:       c.KubernetesVersion,
		ControlPlaneEndpoint:    cpEndpoint,
		CertSANs:                c.GetAllCertSANs(),
		LocalRegistry:           c.LocalRegistry,
		FeatureGates:            c.FeatureGates,
	}
	stepper.Offline = metadata.Offline
	stepper.Version = metadata.KubeVersion
	stepper.LocalRegistry = metadata.LocalRegistry
}

func (stepper *Upgrade) Validate() error {
	if stepper.Kubeadm == nil {
		return fmt.Errorf("upgrade kubeadm object is empty")
	}
	if stepper.Kubeadm.KubernetesVersion == "" {
		return fmt.Errorf("upgrade version must be valid")
	}
	logger.Debug("validate upgrade kubeadm struct", zap.Any("kubeadm", stepper.Kubeadm))
	return nil
}

func (stepper *Upgrade) InitSteps(ctx context.Context) error {
	extraMetadata := component.GetExtraMetadata(ctx)
	if len(extraMetadata.Masters) == 0 {
		return fmt.Errorf("init step error, cluster contains at least one master node")
	}
	if len(stepper.installSteps) != 0 {
		return nil
	}

	masters := extraMetadata.Masters
	workers := extraMetadata.Workers
	packageDownload := &UpgradePackage{
		Package: &Package{
			Version: stepper.Version,
			CriType: stepper.Kubeadm.ContainerRuntime,
			Offline: extraMetadata.Offline,
		},
		DownloadImage: false,
	}
	// master node only in this case will the image package be pulled
	if extraMetadata.Offline && stepper.Kubeadm.LocalRegistry == "" && stepper.LocalRegistry == "" {
		packageDownload.DownloadImage = true
	}
	// When the mirror repository used for the upgrade is valid and not equal to the one used for the cluster creation,
	// the kubeadm configuration file is rendered with the new mirror repository.
	// TODO: During the upgrade, if the image repository changes, synchronize the changes to the docker and containerd configurations
	if stepper.LocalRegistry != "" && stepper.Kubeadm.LocalRegistry != stepper.LocalRegistry {
		stepper.Kubeadm.LocalRegistry = stepper.LocalRegistry
	}
	download, err := json.Marshal(packageDownload)
	if err != nil {
		return err
	}

	stepper.installSteps = []v1.Step{
		{
			ID:        strutil.GetUUID(),
			Name:      "DownloadUpgradePackage",
			Nodes:     utils.UnwrapNodeList(extraMetadata.GetAllNodes()),
			Action:    v1.ActionInstall,
			Timeout:   metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore: false,
			BeforeRunCommands: []v1.Command{
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"rm", "-rf", ManifestDir},
				},
			},
			Commands: []v1.Command{
				{
					Type:          v1.CommandCustom,
					Identity:      fmt.Sprintf(component.RegisterStepKeyFormat, upgradePackage, version, component.TypeStep),
					CustomCommand: download,
				},
				{
					Type:         v1.CommandShell,
					ShellCommand: []string{"chmod", "+x", "/usr/bin/kubelet-pre-start.sh", "/usr/bin/kubelet", "/usr/bin/kubeadm", "/usr/bin/kubectl", "/usr/bin/conntrack"},
				},
			},
			RetryTimes: 1,
		},
	}

	kubeadmBytes, err := json.Marshal(stepper.Kubeadm)
	if err != nil {
		return err
	}
	stepper.installSteps = append(stepper.installSteps, []v1.Step{
		{
			ID:        strutil.GetUUID(),
			Name:      "RenderUpgradeKubeadm",
			Nodes:     []v1.StepNode{utils.UnwrapNodeList(masters)[0]},
			Action:    v1.ActionInstall,
			Timeout:   metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore: false,
			Commands: []v1.Command{
				{
					Type: v1.CommandTemplateRender,
					Template: &v1.TemplateCommand{
						Identity: fmt.Sprintf(component.RegisterTemplateKeyFormat, kubeadmConfig, version, component.TypeTemplate),
						Data:     kubeadmBytes,
					},
				},
			},
			RetryTimes: 1,
		},
		{
			ID:        strutil.GetUUID(),
			Name:      fmt.Sprintf("UpgradeControlPlane-%s", extraMetadata.GetMasterHostname(masters[0].ID)),
			Nodes:     []v1.StepNode{utils.UnwrapNodeList(masters)[0]},
			Action:    v1.ActionInstall,
			Timeout:   metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore: false,
			Commands: []v1.Command{
				{
					Type: v1.CommandShell,
					ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf(`
kubeadm upgrade apply %s -f --ignore-preflight-errors all --config /tmp/.k8s/kubeadm.yaml
sleep 10
kubectl drain %s --ignore-daemonsets || true
systemctl stop kubelet
systemctl daemon-reload && systemctl restart kubelet
kubectl uncordon %s || true`,
						stepper.Version, extraMetadata.GetMasterHostname(masters[0].ID), extraMetadata.GetMasterHostname(masters[0].ID))},
				},
			},
			RetryTimes: 0,
		},
	}...)

	for i := 1; i < len(masters); i++ {
		stepper.installSteps = append(stepper.installSteps, v1.Step{
			ID:        strutil.GetUUID(),
			Name:      fmt.Sprintf("UpgradeControlPlane-%s", extraMetadata.GetMasterHostname(masters[i].ID)),
			Nodes:     []v1.StepNode{utils.UnwrapNodeList(masters)[i]},
			Action:    v1.ActionInstall,
			Timeout:   metav1.Duration{Duration: 10 * time.Minute},
			ErrIgnore: false,
			Commands: []v1.Command{
				{
					Type: v1.CommandShell,
					ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf(`
kubeadm upgrade node
sleep 10
kubectl drain %s --ignore-daemonsets || true
systemctl stop kubelet
systemctl daemon-reload && systemctl restart kubelet
kubectl uncordon %s || true`,
						extraMetadata.GetMasterHostname(masters[i].ID), extraMetadata.GetMasterHostname(masters[i].ID))},
				},
			},
			RetryTimes: 0,
		})
	}

	for i := range workers {
		stepper.installSteps = append(stepper.installSteps, []v1.Step{
			{
				ID:        strutil.GetUUID(),
				Name:      fmt.Sprintf("DrainNode-%s", extraMetadata.GetWorkerHostname(workers[i].ID)),
				Nodes:     []v1.StepNode{utils.UnwrapNodeList(masters)[0]},
				Action:    v1.ActionInstall,
				Timeout:   metav1.Duration{Duration: 1 * time.Minute},
				ErrIgnore: true,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"kubectl", "drain", extraMetadata.GetWorkerHostname(workers[i].ID)},
					},
				},
				RetryTimes: 0,
			},
			{
				ID:        strutil.GetUUID(),
				Name:      fmt.Sprintf("UpgradeWorker-%s", extraMetadata.GetWorkerHostname(workers[i].ID)),
				Nodes:     []v1.StepNode{utils.UnwrapNodeList(workers)[i]},
				Action:    v1.ActionInstall,
				Timeout:   metav1.Duration{Duration: 10 * time.Minute},
				ErrIgnore: false,
				Commands: []v1.Command{
					{
						Type: v1.CommandShell,
						ShellCommand: []string{"/bin/bash", "-c", `
kubeadm upgrade node
systemctl stop kubelet
systemctl daemon-reload && systemctl restart kubelet`},
					},
				},
				RetryTimes: 0,
			},
			{
				ID:        strutil.GetUUID(),
				Name:      fmt.Sprintf("UncordonNode-%s", extraMetadata.GetWorkerHostname(workers[i].ID)),
				Nodes:     []v1.StepNode{utils.UnwrapNodeList(masters)[0]},
				Action:    v1.ActionInstall,
				Timeout:   metav1.Duration{Duration: 1 * time.Minute},
				ErrIgnore: true,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"kubectl", "uncordon", extraMetadata.GetWorkerHostname(workers[i].ID)},
					},
				},
				RetryTimes: 0,
			}}...)
	}
	return nil
}

func (stepper *Upgrade) GetInstallSteps() []v1.Step {
	return stepper.installSteps
}

func (stepper *UpgradePackage) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	// backup kubernetes binary tools
	_, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "mkdir -pv /tmp/.k8s-bak && cp -rf /usr/bin/kubeadm /usr/bin/kubelet /usr/bin/kubectl /tmp/.k8s-bak/")
	if err != nil {
		return nil, err
	}
	instance, err := downloader.NewInstance(ctx, K8s, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if _, err = instance.DownloadAndUnpackConfigs(); err != nil {
		return nil, err
	}
	if stepper.DownloadImage {
		imageSrc, err := instance.DownloadImages()
		if err != nil {
			return nil, err
		}
		if err := utils.LoadImage(ctx, opts.DryRun, imageSrc, stepper.CriType); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (stepper *UpgradePackage) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	instance, err := downloader.NewInstance(ctx, K8s, stepper.Version, runtime.GOARCH, !stepper.Offline, opts.DryRun)
	if err != nil {
		return nil, err
	}
	// remove upgrade package
	if _, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", fmt.Sprintf("rm -rf %s", filepath.Join(downloader.BaseDstDir, K8s))); err != nil {
		return nil, err
	}
	// remove image file
	if stepper.DownloadImage {
		if err = instance.RemoveImages(); err != nil {
			logger.Error("remove k8s upgrade images compressed file failed", zap.Error(err))
		}
	}
	return nil, nil
}

func (stepper *UpgradePackage) NewInstance() component.ObjectMeta {
	return &UpgradePackage{}
}

func (stepper *ActBackup) NewInstance() component.ObjectMeta {
	return &ActBackup{}
}

func (stepper *ActBackup) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	agentConfig, err := config.TryLoadFromDisk()
	if err != nil {
		return nil, errors.WithMessage(err, "load agent config")
	}
	ip, err := netutil.GetDefaultIP(true, agentConfig.IPDetect)
	if err != nil {
		logger.Errorf("get node ip failed: %s", err.Error())
		return nil, err
	}

	if stepper.StoreType == bs.FSStorage {
		if !fileutil.PathExist(stepper.BackupPointRootDir) {
			fErr := fmt.Errorf("'%s' directory does not exist. Please go to all node create this directory and mount it", stepper.BackupPointRootDir)
			// call RunCmdWithContext function in order to log the operation
			_, _ = cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(fmt.Sprintf("[%s] + %s %s\n\n", time.Now().Format(time.RFC3339), "backup pre-check", fErr.Error())))
			return nil, fErr
		}
	}

	// etcdctl snapshot save
	cmd := fmt.Sprintf("etcdctl --endpoints=https://%s:2379 --cacert=/etc/kubernetes/pki/etcd/ca.crt  --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key snapshot save %s",
		ip.String(), stepper.BackupFileName)
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
	if err != nil {
		if ec != nil {
			logger.Errorf("etcdctl snapshot save failed: %s", ec.StdErr())
		}
		return nil, err
	}

	render, err := os.Open(stepper.BackupFileName)
	if err != nil {
		logger.Errorf("open backup file %s failed: %s", stepper.BackupFileName, err.Error())
		return nil, err
	}
	defer render.Close()

	store, err := stepper.BackupStoreCreate()
	if err != nil {
		logger.Errorf("create backup store failed: %s", err.Error())
		return nil, err
	}
	err = store.Save(ctx, render, stepper.BackupFileName)
	if err != nil {
		logger.Errorf("save backup file %s failed: %s", stepper.BackupFileName, err.Error())
		return nil, err
	}

	logger.Info("etcd backup file save successfully")

	// get the backup file's file size and md5 value
	fileInfo, err := render.Stat()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(stepper.BackupPointRootDir, stepper.BackupFileName))
	if err != nil {
		return nil, err
	}
	md5Value := md5.Sum(data)
	checkFile := CheckFile{
		BackupFileSize: fileInfo.Size(),
		BackupFileMD5:  fmt.Sprintf("%x", md5Value),
	}
	cfJSON, err := json.Marshal(checkFile)
	if err != nil {
		return nil, err
	}

	// delete the local backup file
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "rm", "-rf", stepper.BackupFileName)
	if err != nil {
		if ec != nil {
			logger.Errorf("delete local backup file failed: %s", ec.StdErr())
		}
		return nil, err
	}

	logger.Info("etcd backup file save successfully")

	return cfJSON, nil
}

func (stepper *ActBackup) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	store, err := stepper.BackupStoreCreate()
	if err != nil {
		logger.Errorf("create %v backup store failed: %s", stepper.StoreType, err.Error())
		return nil, err
	}
	err = store.Delete(ctx, filepath.Base(stepper.BackupFileName))
	if err != nil {
		logger.Errorf("delete backup file %s failed: %s", stepper.BackupFileName, err.Error())
		return nil, err
	}

	logger.Info("etcd backup file delete successfully")

	return nil, nil
}

func (stepper *ActBackup) InitSteps(ctx context.Context) error {
	extraMetadata := component.GetExtraMetadata(ctx)
	if len(extraMetadata.Masters) == 0 {
		return fmt.Errorf("init step error, create/delete backup cluster contains at least one master node")
	}

	if len(stepper.installSteps) == 0 {
		if err := stepper.makeInstallSteps(&extraMetadata); err != nil {
			return err
		}
	}
	if len(stepper.uninstallSteps) == 0 {
		if err := stepper.makeUninstallSteps(&extraMetadata); err != nil {
			return err
		}
	}

	return nil
}

func (stepper *ActBackup) GetStep(action v1.StepAction) []v1.Step {
	switch action {
	case v1.ActionInstall:
		return stepper.GetInstallSteps()
	case v1.ActionUninstall:
		return stepper.GetUninstallSteps()
	}

	return nil
}

func (stepper *ActBackup) GetInstallSteps() []v1.Step {
	return stepper.installSteps
}

func (stepper *ActBackup) GetUninstallSteps() []v1.Step {
	return stepper.uninstallSteps
}

func (stepper *ActBackup) makeInstallSteps(metadata *component.ExtraMetadata) error {
	rBytes, err := json.Marshal(stepper)
	if err != nil {
		return err
	}
	avaMasters, err := metadata.Masters.AvailableKubeMasters()
	if err != nil {
		return err
	}
	step := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "createBackup",
		Timeout:    metav1.Duration{Duration: 5 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 0,
		Nodes:      utils.UnwrapNodeList(avaMasters[:1]),
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, actBackup, version, component.TypeStep),
				CustomCommand: rBytes,
			},
		},
	}

	stepper.installSteps = append(stepper.installSteps, step)

	return nil
}

func (stepper *ActBackup) makeUninstallSteps(metadata *component.ExtraMetadata) error {
	rBytes, err := json.Marshal(stepper)
	if err != nil {
		return err
	}

	avaMasters, err := metadata.Masters.AvailableKubeMasters()
	if err != nil {
		return err
	}
	step := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "deleteBackup",
		Timeout:    metav1.Duration{Duration: 2 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 0,
		Nodes:      utils.UnwrapNodeList(avaMasters[:1]),
		Action:     v1.ActionUninstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, actBackup, version, component.TypeStep),
				CustomCommand: rBytes,
			},
		},
	}

	stepper.uninstallSteps = append(stepper.uninstallSteps, step)

	return nil
}

func (stepper *ActBackup) BackupStoreCreate() (bs.BackupStore, error) {
	if stepper.StoreType == bs.S3Storage {
		store := &bs.ObjectStore{
			Bucket:          stepper.Bucket,
			Endpoint:        stepper.Endpoint,
			AccessKeyID:     stepper.AccessKeyID,
			AccessKeySecret: stepper.AccessKeySecret,
		}
		return store.Create()
	}
	store := &bs.FilesystemStore{
		RootDir: stepper.BackupPointRootDir,
	}
	return store.Create()
}

func (stepper *Recovery) InitSteps(ctx context.Context) error {
	extraMetadata := component.GetExtraMetadata(ctx)
	if len(extraMetadata.Masters) == 0 {
		return fmt.Errorf("init step error, recovery cluster contains at least one master node")
	}

	err := stepper.Validate()
	if err != nil {
		return err
	}

	if len(stepper.installSteps) == 0 {
		if err := stepper.MakeInstallSteps(ctx, &extraMetadata); err != nil {
			return err
		}
	}

	return nil
}

func (stepper *Recovery) Validate() error {
	if len(stepper.NodeIPList) == 0 || len(stepper.NodeNameList) == 0 {
		return fmt.Errorf("number of master nodes cannot be empty")
	}
	if stepper.RestoreDir == "" {
		return fmt.Errorf("restore cannot be empty")
	}
	if stepper.EtcdDataDir == "" {
		return fmt.Errorf("etcd data dir cannot be empty")
	}
	if stepper.BackupFileName == "" {
		return fmt.Errorf("backup file cannot be empty")
	}

	return nil
}

func (stepper *Recovery) GetInstallSteps() []v1.Step {
	return stepper.installSteps
}

func (stepper *Recovery) MakeInstallSteps(ctx context.Context, metadata *component.ExtraMetadata) error {
	avaMasters, _ := metadata.Masters.AvailableKubeMasters()
	if len(avaMasters) != len(metadata.Masters) {
		return errors.New("there is an unavailable master node in the cluster, please check the cluster master node status")
	}
	rBytes, err := json.Marshal(stepper)
	if err != nil {
		return err
	}

	stepper.installSteps = append(stepper.installSteps, v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "recovery",
		Timeout:    metav1.Duration{Duration: 7 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 0,
		Nodes:      utils.UnwrapNodeList(metadata.Masters),
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      fmt.Sprintf(component.RegisterTemplateKeyFormat, recovery, version, component.TypeStep),
				CustomCommand: rBytes,
			},
		},
	})
	if len(metadata.Workers) > 0 {
		stepper.installSteps = append(stepper.installSteps,
			v1.Step{
				ID:         strutil.GetUUID(),
				Name:       "restartWorkerKubelet",
				Timeout:    metav1.Duration{Duration: 1 * time.Minute},
				ErrIgnore:  false,
				RetryTimes: 1,
				Nodes:      utils.UnwrapNodeList(metadata.Workers),
				Action:     v1.ActionInstall,
				Commands: []v1.Command{
					{
						Type:         v1.CommandShell,
						ShellCommand: []string{"bash", "-c", "systemctl restart kubelet"},
					},
				},
			})
	}

	restart := v1.Step{
		ID:         strutil.GetUUID(),
		Name:       "restartCniAndKubeProxy",
		Timeout:    metav1.Duration{Duration: 5 * time.Minute},
		ErrIgnore:  false,
		RetryTimes: 1,
		Nodes:      []v1.StepNode{utils.UnwrapNodeList(avaMasters)[0]},
		Action:     v1.ActionInstall,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", "while true; do kubectl get po -n kube-system && break;sleep 5; done"},
			},
			{
				Type:         v1.CommandShell,
				ShellCommand: []string{"/bin/bash", "-c", "while true; do if [ 0 == $(kubectl get po -n kube-system | grep kube | grep -v Running | wc -l) ]; then break; else sleep 5 && kubectl get po -n kube-system | grep kube | grep -v Running ;fi ; done"},
			},
		},
	}

	cmdList, err := cni.RecoveryCNICmd(metadata)
	if err == nil {
		restart.Commands = append(restart.Commands, v1.Command{
			Type:         v1.CommandShell,
			ShellCommand: []string{"/bin/bash", "-c", cmdList["restart"]},
		})
	}

	restart.Commands = append(restart.Commands, v1.Command{
		Type:         v1.CommandShell,
		ShellCommand: []string{"/bin/bash", "-c", "kubectl rollout restart ds kube-proxy -n kube-system"},
	})

	if err == nil {
		restart.Commands = append(restart.Commands, v1.Command{
			Type: v1.CommandShell,
			// since the daemon-set controller restarts the pods asynchronously, we try to get the pod status running 3 times in a row and the restart is considered complete
			// for((i=1;i<=3;i++));do sleep 5;while true; do if [ 0 == $(kubectl get po -n kube-system | grep calico | grep -v Running | wc -l) ]; then break; else sleep 5 && kubectl get po -n kube-system | grep calico | grep -v Running ; fi ; done;done;
			ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("for((i=1;i<=3;i++));do sleep 5;while true; do if [ 0 == $(%s | grep -v Running | wc -l) ]; then break; else sleep 5 && %s | grep -v Running ; fi ; done;done;", cmdList["get"], cmdList["get"])},
		})
	}

	restart.Commands = append(restart.Commands, v1.Command{
		Type: v1.CommandShell,
		// Since the daemon-set controller restarts the pods asynchronously, we try to get the pod status running 3 times in a row and the restart is considered complete
		ShellCommand: []string{"/bin/bash", "-c", "for((i=1;i<=3;i++));do sleep 5;while true; do if [ 0 == $(kubectl get po -n kube-system | grep kube-proxy | grep -v Running | wc -l) ]; then break; else sleep 5 && kubectl get po -n kube-system | grep kube-proxy | grep -v Running ; fi ; done;done;"},
	})

	if err != nil {
		note := fmt.Sprintf("===== WARNING: The daemon-set of this cni-%s cannot be restarted, please use kubectl to restart the daemon-set service of this cni-%s- =====", metadata.CNI, metadata.CNI)
		restart.Commands = append(restart.Commands, v1.Command{
			Type: v1.CommandShell,
			// Since the daemon-set controller restarts the pods asynchronously, we try to get the pod status running 3 times in a row and the restart is considered complete
			ShellCommand: []string{"/bin/bash", "-c", fmt.Sprintf("echo %s", note)},
		})
	}

	stepper.installSteps = append(stepper.installSteps, restart)
	return nil
}

func (stepper *Recovery) Install(ctx context.Context, opts component.Options) ([]byte, error) {
	_, err := stepper.Download(ctx, opts)
	if err != nil {
		return nil, err
	}
	_, err = stepper.BeforeRecovery(ctx, opts)
	if err != nil {
		return nil, err
	}
	_, err = stepper.Recovering(ctx, opts)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (stepper *Recovery) Uninstall(ctx context.Context, opts component.Options) ([]byte, error) {
	return nil, fmt.Errorf("recovery dose not support uninstall")
}

func (stepper *Recovery) NewInstance() component.ObjectMeta {
	return stepper
}

func (stepper *Recovery) Download(ctx context.Context, opts component.Options) ([]byte, error) {
	err := os.MkdirAll(stepper.RestoreDir, os.ModePerm)
	if err != nil {
		logger.Errorf("mkdir restore dir failed: %s", err.Error())
		return nil, err
	}
	downloadFile := filepath.Join(stepper.RestoreDir, stepper.BackupFileName)

	writer, err := os.OpenFile(downloadFile, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		logger.Errorf("create restore file failed: %s", err.Error())
		return nil, err
	}

	b, err := stepper.BackupStoreCreate()
	if err != nil {
		logger.Errorf("create backup store failed: %s", err.Error())
		return nil, err
	}
	err = b.Download(ctx, stepper.BackupFileName, writer)
	if err != nil {
		logger.Errorf("download backup file failed: %s", err.Error())
		return nil, err
	}

	logger.Info("download backup file successfully")

	return nil, nil
}

func (stepper *Recovery) BeforeRecovery(ctx context.Context, opts component.Options) ([]byte, error) {
	var err error
	var ec *cmdutil.ExecCmd
	defer func() {
		if err != nil {
			stepper.failedOperate(ctx, opts)
		}
	}()

	// check the download file's size and md5 value
	downloadFile := filepath.Join(stepper.RestoreDir, filepath.Base(stepper.BackupFileName))
	render, err := os.Open(downloadFile)
	if err != nil {
		logger.Errorf("open backup file %s failed: %s", downloadFile, err.Error())
		return nil, err
	}
	defer render.Close()

	fileInfo, err := render.Stat()
	if err != nil {
		return nil, err
	}
	if fileInfo.Size() != stepper.BackupFileSize {
		logger.Errorf("download backup file size is different from backup file size")
		return nil, fmt.Errorf("download backup file size is different from backup file size")
	}

	data, err := os.ReadFile(downloadFile)
	if err != nil {
		logger.Errorf("read download backup file failed: %v", err)
		return nil, err
	}
	md5Value := fmt.Sprintf("%x", md5.Sum(data))
	if md5Value != stepper.BackupFileMD5 {
		logger.Errorf("download backup file md5 value is different from backup file md5 value")
		return nil, fmt.Errorf("download backup file md5 value is different from backup file md5 value")
	}

	cmd := fmt.Sprintf(`rm -rf %s && mv -bf %s %s`,
		stepper.TmpStaticYaml, stepper.ManifestsYaml, stepper.TmpStaticYaml)
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
	if err != nil {
		if ec != nil {
			logger.Errorf("move static yaml(manifests) failed: %s", ec.StdErr())
		}
		return nil, err
	}

	// wait static pod close
	err = retryFunc(ctx, 5*time.Second, "check cluster status", func(ctx context.Context) error {
		ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "kubectl get po")
		if ec != nil {
			if strings.Contains(ec.StdErr(), "6443 was refused") {
				return nil
			}
			err = fmt.Errorf(ec.StdErr())
		}
		return err
	})
	if err != nil {
		logger.Errorf("check static pod cmd failed: %s", err.Error())
		return nil, err
	}

	// mv etcd data
	cmd = fmt.Sprintf(`rm -rf %s && mv -bf %s %s`,
		stepper.TmpEtcdDataDir, stepper.EtcdDataDir, stepper.TmpEtcdDataDir)
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
	if err != nil {
		if ec != nil {
			logger.Errorf("move static yaml(manifests) failed: %s", ec.StdErr())
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	logger.Info("before-recovery successfully")

	return nil, nil
}

func (stepper *Recovery) Recovering(ctx context.Context, opts component.Options) ([]byte, error) {
	// etcd snapshot
	hostInfo, err := sysutil.HostInfo()
	if err != nil {
		logger.Errorf("get host info failed: %s", err.Error())
		return nil, err
	}
	agentConfig, err := config.TryLoadFromDisk()
	if err != nil {
		return nil, errors.WithMessage(err, "load agent config")
	}
	ip, err := netutil.GetDefaultIP(true, agentConfig.IPDetect)
	if err != nil {
		logger.Errorf("get node ip failed: %s", err.Error())
		return nil, err
	}

	var ec *cmdutil.ExecCmd
	defer func() {
		if err != nil {
			stepper.failedOperate(ctx, opts)
		}
	}()

	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "mkdir", "-p", stepper.EtcdDataDir)
	if err != nil {
		if ec != nil {
			logger.Errorf("mkdir etcd data dir failed: %s", ec.StdErr())
		}
		return nil, err
	}

	etcdCluster := fmt.Sprintf("%s=https://%s:2380",
		stepper.NodeNameList[len(stepper.NodeNameList)-1], stepper.NodeIPList[len(stepper.NodeIPList)-1])
	etcdHost := fmt.Sprintf("https://%s:2379", stepper.NodeIPList[len(stepper.NodeIPList)-1])
	for i := len(stepper.NodeNameList) - 2; i >= 0; i-- {
		etcdCluster = fmt.Sprintf("%s=https://%s:2380,%s",
			stepper.NodeNameList[i], stepper.NodeIPList[i], etcdCluster)
		etcdHost = fmt.Sprintf("https://%s:2379,%s", stepper.NodeIPList[i], etcdHost)
	}

	// etcd snapshot cmd
	downloadFile := filepath.Join(stepper.RestoreDir, filepath.Base(stepper.BackupFileName))
	cmd := fmt.Sprintf("etcdctl --name %s --initial-cluster %s --initial-cluster-token etcd-cluster --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key --cacert=/etc/kubernetes/pki/etcd/ca.crt --initial-advertise-peer-urls https://%s:2380 snapshot restore %s --data-dir %s",
		hostInfo.Hostname, etcdCluster, ip, downloadFile, stepper.EtcdDataDir)
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
	if err != nil {
		if ec != nil {
			logger.Errorf("etcd snapshot cmd failed: %s", ec.StdErr())
		}
		return nil, err
	}

	// cp static pod yaml
	cmd = fmt.Sprintf(`\cp -rf %s %s`, stepper.TmpStaticYaml, stepper.ManifestsYaml)
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c",
		cmd, stepper.TmpStaticYaml, stepper.ManifestsYaml)
	if err != nil {
		if ec != nil {
			logger.Errorf("move static yaml(manifests) failed: %s", ec.StdErr())
		}
		return nil, err
	}

	// check etcd cluster health
	err = retryFunc(ctx, 5*time.Second, "check etcd cluster status", func(ctx context.Context) error {
		checkCmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl --endpoints=%s --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key endpoint health | grep -v unhealthy",
			etcdHost)
		ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", checkCmd)
		if ec != nil && ec.StdErr() == "" {
			return nil
		}
		return err
	})
	if err != nil {
		logger.Errorf("check etcd cmd failed: %s", err.Error())
		return nil, err
	}

	// systemctl restart kubelet
	ec, err = cmdutil.RunCmdWithContext(ctx, opts.DryRun, "systemctl", "restart", "kubelet")
	if err != nil {
		if ec != nil {
			logger.Errorf("restart kubelet cmd failed: %s", ec.StdErr())
		}
		return nil, err
	}

	err = retryFunc(ctx, 5*time.Second, "check kubelet status", func(ctx context.Context) error {
		ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", "systemctl status kubelet")
		if ec != nil && !strings.Contains(ec.StdOut(), "active (running)") {
			return nil
		}
		return err
	})
	if err != nil {
		logger.Errorf("check kubelet status cmd failed: %s", err.Error())
		return nil, err
	}

	logger.Info("recovering successfully")

	return nil, nil
}

func (stepper *Recovery) BackupStoreCreate() (bs.BackupStore, error) {
	if stepper.StoreType == bs.S3Storage {
		store := &bs.ObjectStore{
			Bucket:          stepper.Bucket,
			Endpoint:        stepper.Endpoint,
			AccessKeyID:     stepper.AccessKeyID,
			AccessKeySecret: stepper.AccessKeySecret,
		}
		return store.Create()
	}
	store := &bs.FilesystemStore{
		RootDir: stepper.BackupPointRootDir,
	}
	return store.Create()
}

func (stepper *Recovery) failedOperate(ctx context.Context, opts component.Options) {
	logger.Infof("recovery failed, reset to before recovery...")
	cmd := fmt.Sprintf(`rm -rf %s %s && \cp -rf %s %s && \cp -rf %s %s`,
		stepper.ManifestsYaml, stepper.EtcdDataDir, stepper.TmpEtcdDataDir,
		stepper.EtcdDataDir, stepper.TmpStaticYaml, stepper.ManifestsYaml)
	ec, err := cmdutil.RunCmdWithContext(ctx, opts.DryRun, "bash", "-c", cmd)
	if err != nil {
		if ec != nil {
			logger.Errorf("recover stderr error: %s", ec.StdErr())
		}
		logger.Errorf("recover error: %s", err.Error())
	}
}

func retryFunc(ctx context.Context, intervalTime time.Duration, funcName string, fn func(ctx context.Context) error) error {
	for {
		select {
		case <-ctx.Done():
			logger.Warnf("retry function '%s' timeout...", funcName)
			return ctx.Err()
		case <-time.After(intervalTime):
			err := fn(ctx)
			if err == nil {
				return nil
			}
			logger.Warnf("function '%s' running error: %s. about to enter retry", funcName, err.Error())
		}
	}
}
