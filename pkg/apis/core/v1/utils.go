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

package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
	"github.com/kubeclipper/kubeclipper/pkg/utils/httputil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	"github.com/pkg/errors"
)

func reverseComponents(components []v1.Addon) {
	length := len(components)
	for i := 0; i < length/2; i++ {
		components[i], components[length-(i+1)] = components[length-(i+1)], components[i]
	}
}

func getCriStep(ctx context.Context, c *v1.ContainerRuntime, action v1.StepAction, nodes []v1.StepNode) ([]v1.Step, error) {
	switch c.Type {
	case v1.CRIDocker:
		r := cri.DockerRunnable{}
		err := r.InitStep(ctx, c, nodes)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	case v1.CRIContainerd:
		r := cri.ContainerdRunnable{}
		err := r.InitStep(ctx, c, nodes)
		if err != nil {
			return nil, err
		}
		return r.GetActionSteps(action), nil
	}
	return nil, fmt.Errorf("no support %v type cri", c.Type)
}

func getK8sSteps(ctx context.Context, c *v1.Cluster, action v1.StepAction) ([]v1.Step, error) {
	runnable := k8s.Runnable(*c)

	return runnable.GetStep(ctx, action)
}

func getSteps(c component.Interface, action v1.StepAction) ([]v1.Step, error) {
	switch action {
	case v1.ActionInstall:
		return c.GetInstallSteps(), nil
	case v1.ActionUninstall:
		return c.GetUninstallSteps(), nil
	case v1.ActionUpgrade:
		return c.GetUpgradeSteps(), nil
	default:
		return nil, fmt.Errorf("unsupported step action %s", action)
	}
}

func (h *handler) parseOperationFromCluster(extraMetadata *component.ExtraMetadata, c *v1.Cluster, action v1.StepAction) (*v1.Operation, error) {
	var steps []v1.Step
	region := extraMetadata.Masters[0].Region
	if c.Labels == nil {
		c.Labels = map[string]string{
			common.LabelTopologyRegion: region,
		}
	}
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = map[string]string{
		common.LabelClusterName:    c.Name,
		common.LabelTopologyRegion: region,
	}

	// Container runtime should be installed on all nodes.
	ctx := component.WithExtraMetadata(context.TODO(), *extraMetadata)
	stepNodes := utils.UnwrapNodeList(extraMetadata.GetAllNodes())
	cSteps, err := getCriStep(ctx, &c.ContainerRuntime, action, stepNodes)
	if err != nil {
		return nil, err
	}

	// kubernetes
	k8sSteps, err := getK8sSteps(ctx, c, action)
	if err != nil {
		return nil, err
	}

	carr := make([]v1.Addon, len(c.Addons))
	copy(carr, c.Addons)
	if action == v1.ActionUninstall {
		// reverse order of the addons
		reverseComponents(carr)
	} else {
		steps = append(steps, cSteps...)
		steps = append(steps, k8sSteps...)
	}

	for _, com := range carr {
		comp, ok := component.Load(fmt.Sprintf(component.RegisterFormat, com.Name, com.Version))
		if !ok {
			continue
		}
		compMeta := comp.NewInstance()
		if err := json.Unmarshal(com.Config.Raw, compMeta); err != nil {
			return nil, err
		}
		newComp, _ := compMeta.(component.Interface)
		if err := h.initComponentExtraCluster(ctx, newComp); err != nil {
			return nil, err
		}
		if err := newComp.Validate(); err != nil {
			return nil, err
		}
		if err := newComp.InitSteps(ctx); err != nil {
			return nil, err
		}
		s, err := getSteps(newComp, action)
		if err != nil {
			return nil, err
		}
		steps = append(steps, s...)
	}

	if action == v1.ActionUninstall {
		steps = append(steps, k8sSteps...)
		steps = append(steps, cSteps...)
	}

	op.Steps = steps
	return op, nil
}

func (h *handler) initComponentExtraCluster(ctx context.Context, p component.Interface) error {
	cluNames := p.RequireExtraCluster()
	extraClulsterMeta := make(map[string]component.ExtraMetadata, len(cluNames))
	for _, cluName := range cluNames {
		if clu, err := h.clusterOperator.GetClusterEx(ctx, cluName, "0"); err == nil {
			extra, err := h.getClusterMetadata(ctx, clu)
			if err != nil {
				return err
			}
			extraClulsterMeta[cluName] = *extra
		} else {
			return err
		}
	}
	return p.CompleteWithExtraCluster(extraClulsterMeta)
}

func (h *handler) parseRecoverySteps(c *v1.Cluster, b *v1.Backup, restoreDir string, action v1.StepAction) ([]v1.Step, error) {
	steps := make([]v1.Step, 0)

	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, c.Name)
	nodeList, err := h.clusterOperator.ListNodes(context.TODO(), q)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	ips := make([]string, 0)
	var masters []component.Node
	for _, node := range nodeList.Items {
		if node.Labels[common.LabelNodeRole] != "master" {
			continue
		}
		names = append(names, node.Status.NodeInfo.Hostname)
		ips = append(ips, node.Status.Ipv4DefaultIP)
		masters = append(masters, component.Node{
			ID:       node.Name,
			IPv4:     node.Status.Ipv4DefaultIP,
			Hostname: node.Status.NodeInfo.Hostname,
		})
	}

	bp, err := h.clusterOperator.GetBackupPoint(context.TODO(), b.BackupPointName, "0")
	if err != nil {
		return nil, err
	}

	recoveryStep, err := getRecoveryStep(c, bp, b, restoreDir, masters, names, ips, action)
	if err != nil {
		return nil, err
	}

	steps = append(steps, recoveryStep...)

	return steps, nil
}

func getRecoveryStep(c *v1.Cluster, bp *v1.BackupPoint, b *v1.Backup, restoreDir string, masters []component.Node, nodeNames, nodeIPs []string, action v1.StepAction) (steps []v1.Step, err error) {
	meta := component.ExtraMetadata{
		ClusterName: c.Name,
		Masters:     masters,
	}
	ctx := component.WithExtraMetadata(context.TODO(), meta)

	f := k8s.FileDir{
		EtcdDataDir:    c.Etcd.DataDir,
		ManifestsYaml:  k8s.KubeManifestsDir,
		TmpEtcdDataDir: filepath.Join(restoreDir, "etcd"),
		TmpStaticYaml:  filepath.Join(restoreDir, "manifests"),
	}

	r := k8s.Recovery{
		StoreType:      bp.StorageType,
		RestoreDir:     restoreDir,
		BackupFileName: b.Status.FileName,
		NodeNameList:   nodeNames,
		NodeIPList:     nodeIPs,
		BackupFileSize: b.Status.BackupFileSize,
		BackupFileMD5:  b.Status.BackupFileMD5,
		FileDir:        f,
	}

	switch bp.StorageType {
	case bs.FSStorage:
		r.BackupPointRootDir = bp.FsConfig.BackupRootDir
	case bs.S3Storage:
		r.AccessKeyID = bp.S3Config.AccessKeyID
		r.AccessKeySecret = bp.S3Config.AccessKeySecret
		r.Bucket = bp.S3Config.Bucket
		r.Endpoint = bp.S3Config.Endpoint
	}

	err = r.InitSteps(ctx)
	if err != nil {
		return nil, err
	}

	switch action {
	case v1.ActionInstall:
		return r.GetInstallSteps(), nil
	}
	return nil, nil
}

// parseOperationFromComponent parse operation instance from component
func (h *handler) parseOperationFromComponent(extraMetadata *component.ExtraMetadata, addons []v1.Addon, c *v1.Cluster, action v1.StepAction) (*v1.Operation, error) {
	var steps []v1.Step
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = map[string]string{
		common.LabelClusterName:    c.Name,
		common.LabelTopologyRegion: extraMetadata.Masters[0].Region,
	}
	// with extra meta data
	ctx := component.WithExtraMetadata(context.TODO(), *extraMetadata)
	for _, comp := range addons {
		cInterface, ok := component.Load(fmt.Sprintf(component.RegisterFormat, comp.Name, comp.Version))
		if !ok {
			continue
		}
		instance := cInterface.NewInstance()
		if err := json.Unmarshal(comp.Config.Raw, instance); err != nil {
			return nil, err
		}
		newComp, ok := instance.(component.Interface)
		if !ok {
			continue
		}
		if err := newComp.Validate(); err != nil {
			return nil, err
		}
		if err := newComp.InitSteps(ctx); err != nil {
			return nil, err
		}
		s, err := getSteps(newComp, action)
		if err != nil {
			return nil, err
		}
		steps = append(steps, s...)
	}

	op.Steps = steps
	return op, nil
}

func (h *handler) parseActBackupSteps(c *v1.Cluster, b *v1.Backup, action v1.StepAction) ([]v1.Step, error) {
	steps := make([]v1.Step, 0)

	q := query.New()
	q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelClusterName, c.Name)

	bp, err := h.clusterOperator.GetBackupPointEx(context.TODO(), c.Labels[common.LabelBackupPoint], "0")
	if err != nil {
		return nil, err
	}
	pNode, err := h.clusterOperator.GetNodeEx(context.TODO(), b.PreferredNode, "0")
	if err != nil {
		return nil, err
	}
	actBackupStep, err := getActBackupStep(c, b, bp, pNode, action)
	if err != nil {
		return nil, err
	}

	steps = append(steps, actBackupStep...)

	return steps, nil
}

func getActBackupStep(c *v1.Cluster, b *v1.Backup, bp *v1.BackupPoint, pNode *v1.Node, action v1.StepAction) (steps []v1.Step, err error) {
	var actBackup *k8s.ActBackup
	meta := component.ExtraMetadata{
		ClusterName: c.Name,
	}
	meta.Masters = []component.Node{{
		ID:       b.PreferredNode, // the preferred node is used by default
		IPv4:     pNode.Status.Ipv4DefaultIP,
		Hostname: pNode.Status.NodeInfo.Hostname,
	}}
	ctx := component.WithExtraMetadata(context.TODO(), meta)

	switch bp.StorageType {
	case bs.S3Storage:
		actBackup = &k8s.ActBackup{
			StoreType:       bp.StorageType,
			BackupFileName:  b.Status.FileName,
			AccessKeyID:     bp.S3Config.AccessKeyID,
			AccessKeySecret: bp.S3Config.AccessKeySecret,
			Bucket:          bp.S3Config.Bucket,
			Endpoint:        bp.S3Config.Endpoint,
		}
	case bs.FSStorage:
		actBackup = &k8s.ActBackup{
			StoreType:          bp.StorageType,
			BackupFileName:     b.Status.FileName,
			BackupPointRootDir: bp.FsConfig.BackupRootDir,
		}
	}

	if err = actBackup.InitSteps(ctx); err != nil {
		return
	}

	return actBackup.GetStep(action), nil
}

func (h *handler) parseUpdateCertOperation(clu *v1.Cluster, extraMetadata *component.ExtraMetadata) (*v1.Operation, error) {
	cert := &k8s.Certification{}
	op := &v1.Operation{}
	nodes := utils.UnwrapNodeList(extraMetadata.Masters)
	op.Steps, _ = cert.InstallSteps(clu, nodes)
	return op, nil
}

func (h *handler) checkBackupPointInUse(backups *v1.BackupList, name string) bool {
	for _, item := range backups.Items {
		if item.BackupPointName == name {
			return true
		}
	}
	return false
}

func (h *handler) clusterNodeCheck(c *v1.Cluster, ipMap map[string]string, ips []string) error {
	nodeList, err := h.clusterOperator.ListNodes(context.TODO(), query.New())
	if err != nil {
		return err
	}
	for _, no := range nodeList.Items {
		if _, ok := ipMap[no.Status.Ipv4DefaultIP]; ok {
			return fmt.Errorf("node %s already exists", no.Status.Ipv4DefaultIP)
		}
		for _, addr := range no.Status.Addresses {
			if _, ok := ipMap[addr.Address]; ok {
				return fmt.Errorf("node %s already exists", no.Status.Ipv4DefaultIP)
			}
		}
	}
	if err := SudoCheck("sudo", c.Provider.SSHConfig.Convert(), ips); err != nil {
		return err
	}
	// check if the node is already added
	for _, nodeIP := range ips {
		if !preCheckKcAgent(c, nodeIP) {
			return fmt.Errorf("check node %s kc-agent service failed", nodeIP)
		}
	}

	// TODO: Currently considering only direct connection networks
	//return sudo.MultiNIC("ipDetect", o.deployConfig.SSHConfig, o.IOStreams, nodeIPs, c.ipDetect)
	return nil
}

func SudoCheck(name string, sshConfig *sshutils.SSH, allNodes []string) error {
	if sshConfig.User == "" {
		return fmt.Errorf("current ssh user is empty")
	}

	// need enter passwd/ssh-key to run sudo
	if sshConfig.Password == "" && sshConfig.PkFile == "" && sshConfig.PkDataEncode == "" {
		return fmt.Errorf("The current user(%s) ssh pk-file/pk/paasword are empty", sshConfig.User)
	}

	if sshConfig.User == "root" && name == "sudo" {
		return nil
	}
	// check sudo access
	err := sshutils.CmdBatchWithSudo(sshConfig, allNodes, "id -u", func(result sshutils.Result, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), "handshake failed: ssh: unable to authenticate, attempted methods [none password]") {
				return fmt.Errorf("passwd or user error while ssh '%s@%s',please try again", result.User, result.Host)
			}
			return err
		}
		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "is not in the sudoers file") {
				return fmt.Errorf("user '%s@%s' is not in the sudoers file,please config it", result.User, result.Host)
			}

			if strings.Contains(result.Stderr, "incorrect password attempt") {
				return fmt.Errorf("passwd error for '%s@%s',please try again", result.User, result.Host)
			}
			return fmt.Errorf("%s stderr:%s", result.Short(), result.Stderr)
		}
		return nil
	})

	return err
}

func preCheckKcAgent(c *v1.Cluster, ip string) bool {
	// TODO: check if the node is already in deploy config
	// TODO: In the future we should consider storing deploy config deployment parameters in etcd

	// check if kc-agent is running
	ret, err := sshutils.SSHCmdWithSudo(c.Provider.SSHConfig.Convert(), ip, "systemctl stop kc-agent")
	if err != nil {
		logger.Errorf("check node %s failed: %s", ip, err.Error())
		return false
	}
	if ret.ExitCode == 0 && ret.Stdout != "" {
		logger.Errorf("kc-agent.service already exist in node(%s), please go to node(%s) and run the command: sudo systemctl stop kc-agent && sudo rm -rf /usr/local/bin/kubeclipper-agent", ip, ip)
		return false
	}
	return true
}

func (h *handler) agentNodeFiles(c *v1.Cluster, nodeIPs []string, ipMap map[string]string) error {
	fmt.Println("qqqqqqqqqqq", nodeIPs)
	sshConf := c.Provider.SSHConfig.Convert()
	// send agent binary
	hook := fmt.Sprintf("cp -rf %s /usr/local/bin/ && chmod +x /usr/local/bin/kubeclipper-agent",
		filepath.Join("/tmp", "kubeclipper-agent"))
	err := SendPackageV2(sshConf,
		"/usr/local/bin/kubeclipper-agent", nodeIPs, "/tmp", nil, &hook)
	if err != nil {
		return errors.Wrap(err, "SendPackageV2")
	}
	hook = fmt.Sprintf("cp -rf %s /usr/local/bin/etcdctl && chmod +x /usr/local/bin/etcdctl",
		filepath.Join("/tmp", "etcdctl"))
	err = SendPackageV2(sshConf,
		"/usr/local/bin/etcdctl", nodeIPs, "/tmp", nil, &hook)
	if err != nil {
		return errors.Wrap(err, "SendPackageV2")
	}
	err = h.sendCerts(c, nodeIPs)
	if err != nil {
		return err
	}
	for ip, id := range ipMap {
		fmt.Println("@@@@@@@@@@", ip)
		agentConfig := h.getKcAgentConfigTemplateContent(c, id)
		cmdList := []string{
			sshutils.WrapEcho(KcAgentService, "/usr/lib/systemd/system/kc-agent.service"), // write systemd file
			"mkdir -pv /etc/kubeclipper-agent ",
			sshutils.WrapEcho(agentConfig, "/etc/kubeclipper-agent/kubeclipper-agent.yaml"), // write agent.yaml
		}
		for _, cmd := range cmdList {
			ret, err := sshutils.SSHCmdWithSudo(sshConf, ip, cmd)
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

func (h *handler) sendCerts(c *v1.Cluster, nodeIPs []string) error {
	if !h.config.MQOptions.Client.TLS {
		return nil
	}
	sshConf := c.Provider.SSHConfig.Convert()
	// download cert from server
	files := []string{
		h.config.MQOptions.Client.TLSCaPath,
		h.config.MQOptions.Client.TLSCertPath,
		h.config.MQOptions.Client.TLSKeyPath,
	}

	for _, file := range files {
		exist, err := sshutils.IsFileExist(file)
		if err != nil {
			return errors.WithMessage(err, "check file exist")
		}
		if !exist {
			if err = sshConf.DownloadSudo(h.config.StaticServerOptions.BindAddress, file, file); err != nil {
				return errors.WithMessage(err, "download cert from server")
			}
		}
	}

	destCa := filepath.Join(DefaultKcAgentConfigPath, DefaultCaPath)
	destCert := filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath)
	destKey := filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath)
	if h.config.MQOptions.External {
		destCa = filepath.Dir(h.config.MQOptions.Client.TLSCaPath)
		destCert = filepath.Dir(h.config.MQOptions.Client.TLSCertPath)
		destKey = filepath.Dir(h.config.MQOptions.Client.TLSKeyPath)
	}

	err := SendPackageV2(sshConf,
		h.config.MQOptions.Client.TLSCaPath, nodeIPs, destCa, nil, nil)
	if err != nil {
		return err
	}
	err = SendPackageV2(sshConf,
		h.config.MQOptions.Client.TLSCertPath, nodeIPs, destCert, nil, nil)
	if err != nil {
		return err
	}
	err = SendPackageV2(sshConf,
		h.config.MQOptions.Client.TLSKeyPath, nodeIPs, destKey, nil, nil)
	if err != nil {
		return err
	}
	logger.Info("sendCerts successfully")
	return nil
}

func (h *handler) getKcAgentConfigTemplateContent(c *v1.Cluster, id string) string {
	tmpl, err := template.New("text").Parse(KcAgentConfigTmpl)
	if err != nil {
		logger.Fatalf("template parse failed: %s", err.Error())
	}

	var data = make(map[string]interface{})
	data["Region"] = c.Provider.NodeRegion
	data["IPDetect"] = ""
	data["AgentID"] = id
	data["StaticServerAddress"] = fmt.Sprintf(
		"http://%s:%d", h.config.StaticServerOptions.BindAddress, h.config.StaticServerOptions.InsecurePort)
	data["LogLevel"] = h.config.LogOptions.Level
	data["MQServerEndpoints"] = h.config.MQOptions.Client.ServerAddress
	data["MQExternal"] = h.config.MQOptions.External
	data["MQUser"] = h.config.MQOptions.Auth.UserName
	data["MQAuthToken"] = h.config.MQOptions.Auth.Password
	data["MQTLS"] = h.config.MQOptions.Client.TLS
	if h.config.MQOptions.Client.TLS {
		if h.config.MQOptions.External {
			data["MQCaPath"] = h.config.MQOptions.Client.TLSCaPath
			data["MQClientCertPath"] = h.config.MQOptions.Client.TLSCertPath
			data["MQClientKeyPath"] = h.config.MQOptions.Client.TLSKeyPath
		} else {
			data["MQCaPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultCaPath, filepath.Base(h.config.MQOptions.Client.TLSCaPath))
			data["MQClientCertPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath, filepath.Base(h.config.MQOptions.Client.TLSCertPath))
			data["MQClientKeyPath"] = filepath.Join(DefaultKcAgentConfigPath, DefaultNatsPKIPath, filepath.Base(h.config.MQOptions.Client.TLSKeyPath))
		}
	}
	data["OpLogDir"] = h.config.LogOptions.LogFile
	data["OpLogThreshold"] = h.config.LogOptions.LogFileMaxSizeMB
	// TODO
	data["KcImageRepoMirror"] = ""
	var buffer bytes.Buffer
	if err = tmpl.Execute(&buffer, data); err != nil {
		logger.Fatalf("template execute failed: %s", err.Error())
	}
	return buffer.String()
}

func (h *handler) enableAgent(c *v1.Cluster, nodeIPs []string) error {
	fmt.Println("wwwwwwwwwwwww", nodeIPs)
	for _, nodeIP := range nodeIPs {
		// enable agent service
		ret, err := sshutils.SSHCmdWithSudo(c.Provider.SSHConfig.Convert(), nodeIP, "systemctl daemon-reload && systemctl enable kc-agent --now")
		if err != nil {
			return errors.Wrap(err, "enable kc agent")
		}
		if err = ret.Error(); err != nil {
			return errors.Wrap(err, "enable kc agent")
		}
		// TODO: update deploy-config.yaml
		//meta := options.Metadata{
		//	Region:  c.Provider.NodeRegion,
		//	FloatIP: "",
		//}
		//o.deployConfig.Agents.Add(nodeIP, meta)
		//err = o.deployConfig.Write()
		//if err != nil {
		//	return err
		//}
	}

	return nil
}

func (h *handler) clusterServiceAccount(c *v1.Cluster, action v1.StepAction) {
	switch action {
	case v1.ActionInstall:
		action = "create"
	case v1.ActionUninstall:
		action = "delete"
	}
	sa := []string{"kubectl", string(action), "sa", "kc-server", "-n", "kube-system"}
	bind := []string{"kubectl", string(action), "clusterrolebinding", "kc-server", "--clusterrole=cluster-admin", "--serviceaccount=kube-system:kc-server"}
	res, err := h.delivery.DeliverCmd(context.TODO(), c.Masters[0].ID, sa, 3*time.Second)
	if err != nil {
		logger.Warnf("create ServiceAccount failed: %v. result: %s", err, string(res))
		return
	}
	res, err = h.delivery.DeliverCmd(context.TODO(), c.Masters[0].ID, bind, 3*time.Second)
	if err != nil {
		logger.Warnf("create clusterrolebinding failed: %v. result: %s", err, string(res))
		return
	}
}

// SendPackageV2 scp file to remote host
func SendPackageV2(sshConfig *sshutils.SSH, location string, hosts []string, dstDir string, before, after *string) error {
	var md5 string
	// download pkg to /tmp/kc/
	location, md5, err := downloadFile(location)
	if err != nil {
		return errors.Wrap(err, "downloadFile")
	}
	pkg := path.Base(location)
	// scp to ~/kc/kc-bj-cd.tar.gz
	fullPath := fmt.Sprintf("%s/%s", dstDir, pkg)
	mkDstDir := fmt.Sprintf("mkdir -p %s || true", dstDir)
	var wg sync.WaitGroup
	var errCh = make(chan error, len(hosts))
	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			_, _ = sshutils.SSHCmd(sshConfig, host, mkDstDir)
			if before != nil {
				ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, *before)
				if err != nil {
					errCh <- errors.WithMessage(err, "run before hook")
					return
				}
				if err = ret.Error(); err != nil {
					errCh <- errors.WithMessage(err, "run before hook ret.Error()")
					return
				}
			}
			exists, err := sshConfig.IsFileExistV2(host, fullPath)
			if err != nil {
				errCh <- errors.WithMessage(err, "IsFileExistV2")
				return
			}
			if exists {
				validate, err := sshConfig.ValidateMd5sumLocalWithRemote(host, location, fullPath)
				if err != nil {
					errCh <- errors.WithMessage(err, "ValidateMd5sumLocalWithRemote")
					return
				}

				if validate {
					logger.Infof("[%s]SendPackage:  %s file is exist and ValidateMd5 success", host, fullPath)
				} else {
					// del then copy
					rm := fmt.Sprintf("rm -rf %s", fullPath)
					ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, rm)
					if err != nil {
						logger.Errorf("[%s]remove old file(%s) err %s", host, fullPath, err.Error())
						return
					}
					if err = ret.Error(); err != nil {
						logger.Errorf("[%s]remove old file(%s) err %s", host, fullPath, err.Error())
						return
					}
					ok, err := sshConfig.CopyForMD5V2(host, location, fullPath, md5)
					if err != nil {
						logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, location, err.Error())
						return
					}
					if ok {
						logger.Infof("[%s]copy file(%s) md5 validate success", host, location)
					} else {
						logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
					}
				}
			} else {
				ok, err := sshConfig.CopyForMD5V2(host, location, fullPath, md5)
				if err != nil {
					logger.Errorf("[%s]copy file(%s) md5 validate failed err %s", host, fullPath, err.Error())
					return
				}
				if ok {
					logger.Infof("[%s]copy file(%s) md5 validate success", host, location)
				} else {
					logger.Errorf("[%s]copy file(%s) md5 validate failed", host, location)
				}
			}

			if after != nil {
				ret, err := sshutils.SSHCmdWithSudo(sshConfig, host, *after)
				if err != nil {
					errCh <- errors.WithMessage(err, "run after hook")
					return
				}
				if err = ret.Error(); err != nil {
					errCh <- errors.WithMessage(err, "run after hook ret.Error()")
					return
				}
			}
		}(host)
	}
	var stopCh = make(chan struct{}, 1)
	go func() {
		defer func() {
			close(stopCh)
			close(errCh)
		}()
		wg.Wait()
		stopCh <- struct{}{}
	}()

	select {
	case err = <-errCh:
		return err
	case <-stopCh:
		return nil
	}
}

func downloadFile(location string) (filePATH, md5 string, err error) {
	if _, ok := httputil.IsURL(location); ok {
		absPATH := "/tmp/kc/" + path.Base(location)
		exist, err := sshutils.IsFileExist(absPATH)
		if err != nil {
			return "", "", err
		}
		if !exist {
			// generator download cmd
			dwnCmd := downloadCmd(location)
			// os exec download command
			sshutils.Cmd("/bin/sh", "-c", "mkdir -p /tmp/kc && cd /tmp/kc && "+dwnCmd)
		}
		location = absPATH
	}
	// file md5
	md5, err = sshutils.MD5FromLocal(location)
	return location, md5, errors.Wrap(err, "MD5FromLocal")
}

// downloadCmd build cmd from url.
func downloadCmd(url string) string {
	// only http
	u, isHTTP := httputil.IsURL(url)
	var c = ""
	if isHTTP {
		param := ""
		if u.Scheme == "https" {
			param = "--no-check-certificate"
		}
		c = fmt.Sprintf(" wget -c %s %s", param, url)
	}
	return c
}

const (
	DefaultCaPath            = "pki"
	DefaultNatsPKIPath       = "pki/nats"
	DefaultKcAgentConfigPath = "/etc/kubeclipper-agent"
)

func (h *handler) cleanAgentNodeFiles(c *v1.Cluster) error {
	cmd := "rm -rf /usr/local/bin/etcdctl && rm -rf /etc/kubeclipper-agent && rm -rf/usr/lib/systemd/system/kc-agent.service && systemctl disable kc-agent --now"
	nodes := append(c.Masters, c.Workers...)

	for _, no := range nodes {
		_, err := h.delivery.DeliverCmd(context.TODO(), no.ID, []string{"/bin/bash", "-c", cmd}, 3*time.Second)
		if err != nil {
			logger.Warnf("clean node(%s) kc-agent service failed: %v", no.ID, err)
			return err
		}
	}
	return nil
}
