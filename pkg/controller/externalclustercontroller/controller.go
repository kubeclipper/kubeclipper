package externalclustercontroller

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	agentconfig "github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/config"
	"github.com/kubeclipper/kubeclipper/pkg/cli/utils"
	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	utils2 "github.com/kubeclipper/kubeclipper/pkg/controller/utils"
	"github.com/kubeclipper/kubeclipper/pkg/externalcluster"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/utils/sshutils"
	errors2 "github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
)

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

const SyncCycle = time.Hour * 4

type ExternalClusterReconciler struct {
	CmdDelivery       service.CmdDelivery
	mgr               manager.Manager
	ClusterLister     listerv1.ClusterLister
	NodeLister        listerv1.NodeLister
	ClusterWriter     cluster.ClusterWriter
	NodeWriter        cluster.NodeWriter
	ConfigMapOperator core.Operator
}

func (r *ExternalClusterReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("cluster", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("cluster-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Cluster{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	r.mgr = mgr
	mgr.AddRunnable(c)
	return nil
}

func (r *ExternalClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	clu, err := r.ClusterLister.Get(req.Name)
	if err != nil {
		// cluster not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get cluster", zap.Error(err))
		return ctrl.Result{}, err
	}
	if clu.Annotations == nil {
		return ctrl.Result{}, nil
	}

	if _, ok := clu.Annotations[common.AnnotationConfigMap]; !ok {
		return ctrl.Result{}, nil
	}

	if clu.Status.Phase == v1.ClusterImporting || clu.Status.Phase == v1.ClusterImportFailed {
		err := r.integratedCluster(clu)
		if err != nil {
			logger.Errorf("integrated external cluster failed: %v", err)
			clu.Status.Phase = v1.ClusterImportFailed
			return ctrl.Result{}, err
		}

		clu.Status.Phase = v1.ClusterRunning

		_, err = r.ClusterWriter.UpdateCluster(ctx, clu)
		if err != nil && !errors.IsNotFound(err) {
			logger.Errorf("update external cluster status failed: %v", err)
			return ctrl.Result{}, err
		}
		logger.Info("update external cluster status successfully")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: SyncCycle}, nil
}

func (r *ExternalClusterReconciler) integratedCluster(clu *v1.Cluster) error {
	err := r.integratedClusterNode(clu)
	if err != nil {
		return fmt.Errorf("send agent failed: %v", err)
	}

	time.Sleep(5 * time.Second)

	err = utils2.ClusterServiceAccount(r.CmdDelivery, clu.Masters.GetNodeIDs()[0], v1.ActionInstall)
	if err != nil {
		logger.Errorf("create ServiceAccount ", err)
	}

	return nil
}

func (r *ExternalClusterReconciler) integratedClusterNode(clu *v1.Cluster) error {
	deployConfig, err := r.getDeployConfig()
	if err != nil {
		return fmt.Errorf("get deploy config failed: %v", err)
	}
	cm, err := r.ConfigMapOperator.GetConfigMapEx(context.TODO(), clu.Annotations[common.AnnotationConfigMap], "")
	if err != nil {
		return fmt.Errorf("get import cluster config failed: %v", err)
	}

	nodes := make([]*v1.WorkerNode, 0)
	for i, _ := range clu.Masters {
		nodes = append(nodes, &clu.Masters[i])
	}
	for i, _ := range clu.Workers {
		nodes = append(nodes, &clu.Workers[i])
	}

	// due to pointer passing, the IP of the cluster node will be modified to the ID
	return r.installAgent(nodes, cm, deployConfig)
}

func (r *ExternalClusterReconciler) getDeployConfig() (*options.DeployConfig, error) {
	deploy, err := r.ConfigMapOperator.GetConfigMapEx(context.TODO(), constatns.DeployConfigConfigMapName, "0")
	if err != nil {
		return nil, fmt.Errorf("get deploy config failed: %v", err)
	}
	confString := deploy.Data[constatns.DeployConfigConfigMapKey]
	conf := &options.DeployConfig{}
	err = yaml.Unmarshal([]byte(confString), conf)
	if err != nil {
		return nil, fmt.Errorf("deploy-config unmarshal failed: %v", err)
	}
	return conf, nil
}

func (r *ExternalClusterReconciler) updateDeployConfigAgents(ip string, meta *options.Metadata) error {
	deploy, err := r.ConfigMapOperator.GetConfigMapEx(context.TODO(), constatns.DeployConfigConfigMapName, "0")
	if err != nil {
		return fmt.Errorf("get deploy config failed: %v", err)
	}
	confString := deploy.Data[constatns.DeployConfigConfigMapKey]
	deployConfig := &options.DeployConfig{}
	err = yaml.Unmarshal([]byte(confString), deployConfig)
	if err != nil {
		return fmt.Errorf("deploy-config unmarshal failed: %v", err)
	}

	deployConfig.Agents.Add(ip, *meta)
	dcData, err := yaml.Marshal(deployConfig)
	if err != nil {
		return fmt.Errorf("deploy config marshal failed: %v", err)
	}
	deploy.Data[constatns.DeployConfigConfigMapKey] = string(dcData)
	_, err = r.ConfigMapOperator.UpdateConfigMap(context.TODO(), deploy)
	return err
}

func (r *ExternalClusterReconciler) agentStatus(ip string, sshConfig *sshutils.SSH) (id string, active bool) {
	// check if kc-agent is running
	ret, err := sshutils.SSHCmdWithSudo(sshConfig, ip,
		"systemctl --all --type service | grep kc-agent | grep running | wc -l")
	if err != nil {
		logger.Warnf("check node %s failed: %s", ip, err.Error())
		return "", false
	}
	if ret.StdoutToString("") == "0" {
		logger.Debugf("kc-agent service not exist on %s", ip)
		return "", false
	}

	ret, err = sshutils.SSHCmdWithSudo(sshConfig, ip,
		"cat /etc/kubeclipper-agent/kubeclipper-agent.yaml")
	if err != nil {
		logger.Warnf("check node %s failed: %s", ip, err.Error())
		return "", true
	}

	agentConf := &agentconfig.Config{}
	err = yaml.Unmarshal([]byte(ret.Stdout), agentConf)
	if err != nil {
		logger.Warnf("node(%s) agent agentConf unmarshal failed: %s", ip, err.Error())
		return "", true
	}

	return agentConf.AgentID, true
}

// due to pointer passing, the IP of the cluster node will be modified to the ID
func (r *ExternalClusterReconciler) installAgent(nodes []*v1.WorkerNode, cm *v1.ConfigMap, deployConfig *options.DeployConfig) error {
	ssh := externalcluster.SSH{}
	err := ssh.ReadEntity(cm)
	if err != nil {
		return err
	}
	sshConf := ssh.Convert()
	// the ID of the current node is replaced with an IP, which will be replaced with an ID later
	for i, _ := range nodes {
		ip := nodes[i].ID
		if !(net.ParseIP(nodes[i].ID) != nil && strings.Contains(nodes[i].ID, ".")) {
			node, err := r.NodeLister.Get(nodes[i].ID)
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			ip = node.Status.Ipv4DefaultIP
		}

		meta := options.Metadata{
			Region: cm.Data["nodeRegion"],
		}
		// due to pointer passing, the IP of the cluster node will be modified to the ID
		originalID, active := r.agentStatus(ip, sshConf)
		if originalID != "" {
			nodes[i].ID = originalID
		} else {
			nodes[i].ID = uuid.New().String()
		}
		if active {
			if !deployConfig.Agents.Exists(ip) {
				err = r.updateDeployConfigAgents(ip, &meta)
				if err != nil {
					logger.Errorf("add agent ip to deploy config failed: %v", err)
					return err
				}
			}
			logger.Warnf("update deploy-config agent failed: %v", err)
			continue
		}
		err = r.sendAgentFile(ip, nodes[i].ID, meta, deployConfig, sshConf)
		if err != nil {
			logger.Errorf("send agent file failed: %v", err)
			return err
		}
		err = r.sendCerts(ip, deployConfig, sshConf)
		if err != nil {
			logger.Errorf("send certs failed: %v", err)
			return err
		}
		err = r.enableAgent(ip, sshConf)
		if err != nil {
			logger.Errorf("start agent service failed: %v", err)
			return err
		}
		err = r.updateDeployConfigAgents(ip, &meta)
		if err != nil {
			return err
		}
	}

	logger.Info("install kc-agent service successfully")
	return nil
}

func (r *ExternalClusterReconciler) sendAgentFile(ip, id string, meta options.Metadata, deployConfig *options.DeployConfig, sshConfig *sshutils.SSH) error {
	// send agent binary
	hook := fmt.Sprintf("cp -rf %s /usr/local/bin/ && chmod +x /usr/local/bin/kubeclipper-agent",
		filepath.Join("/tmp", "kubeclipper-agent"))
	err := utils.SendPackageV2(sshConfig,
		"/usr/local/bin/kubeclipper-agent", []string{ip}, "/tmp", nil, &hook)
	if err != nil {
		return errors2.Wrap(err, "SendPackageV2")
	}
	hook = fmt.Sprintf("cp -rf %s /usr/local/bin/etcdctl && chmod +x /usr/local/bin/etcdctl",
		filepath.Join("/tmp", "etcdctl"))
	err = utils.SendPackageV2(sshConfig,
		"/usr/local/bin/etcdctl", []string{ip}, "/tmp", nil, &hook)
	if err != nil {
		return errors2.Wrap(err, "SendPackageV2")
	}

	agentConfigData := deployConfig.GetKcAgentConfig(meta)
	agentConfigData["AgentID"] = id
	agentConfig, err := deployConfig.KcAgentConfigString(agentConfigData)
	if err != nil {
		return fmt.Errorf("kc-agent config to string failed: %v", err)
	}
	cmdList := []string{
		sshutils.WrapEcho(config.KcAgentService, "/usr/lib/systemd/system/kc-agent.service"), // write systemd file
		"mkdir -pv /etc/kubeclipper-agent ",
		sshutils.WrapEcho(agentConfig, "/etc/kubeclipper-agent/kubeclipper-agent.yaml"), // write agent.yaml
	}
	for _, cmd := range cmdList {
		ret, err := sshutils.SSHCmdWithSudo(sshConfig, ip, cmd)
		if err != nil {
			return err
		}
		if err = ret.Error(); err != nil {
			return err
		}
	}
	logger.Debugf("send agent file successfully")
	return nil
}

func (r *ExternalClusterReconciler) sendCerts(ip string, deployConfig *options.DeployConfig, sshConfig *sshutils.SSH) error {
	destCa := filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultCaPath)
	destCert := filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultNatsPKIPath)
	destKey := filepath.Join(options.DefaultKcAgentConfigPath, options.DefaultNatsPKIPath)
	if deployConfig.MQ.External {
		destCa = filepath.Dir(deployConfig.MQ.CA)
		destCert = filepath.Dir(deployConfig.MQ.ClientCert)
		destKey = filepath.Dir(deployConfig.MQ.ClientKey)
	}

	err := utils.SendPackageV2(sshConfig,
		deployConfig.MQ.CA, []string{ip}, destCa, nil, nil)
	if err != nil {
		return err
	}
	err = utils.SendPackageV2(sshConfig,
		deployConfig.MQ.ClientCert, []string{ip}, destCert, nil, nil)
	if err != nil {
		return err
	}
	err = utils.SendPackageV2(sshConfig,
		deployConfig.MQ.ClientKey, []string{ip}, destKey, nil, nil)
	if err != nil {
		return err
	}

	logger.Debugf("send cert successfully")
	return nil
}

func (r *ExternalClusterReconciler) enableAgent(nodeIP string, sshConfig *sshutils.SSH) error {
	// enable agent service
	ret, err := sshutils.SSHCmdWithSudo(sshConfig, nodeIP, "systemctl daemon-reload && systemctl enable kc-agent --now")
	if err != nil {
		return errors2.Wrap(err, "enable kc agent")
	}
	if err = ret.Error(); err != nil {
		return errors2.Wrap(err, "enable kc agent result")
	}

	return nil
}
