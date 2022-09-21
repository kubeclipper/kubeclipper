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

package clustercontroller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/constatns"
	"github.com/kubeclipper/kubeclipper/pkg/controller/utils"
	"github.com/kubeclipper/kubeclipper/pkg/models/core"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"

	"github.com/kubeclipper/kubeclipper/pkg/service"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubeclipper/kubeclipper/pkg/models/operation"

	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type ClusterReconciler struct {
	CmdDelivery         service.CmdDelivery
	mgr                 manager.Manager
	ClusterLister       listerv1.ClusterLister
	NodeLister          listerv1.NodeLister
	NodeWriter          cluster.NodeWriter
	ClusterWriter       cluster.ClusterWriter
	OperationWriter     operation.Writer
	CronBackupWriter    cluster.CronBackupWriter
	ConfigMapOperator   core.Operator
	CloudProviderLister listerv1.CloudProviderLister
}

func (r *ClusterReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
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
	if err = c.Watch(source.NewKindWithCache(&v1.Node{}, cache), handler.EnqueueRequestsFromMapFunc(r.findObjectsForCluster)); err != nil {
		return err
	}

	r.mgr = mgr
	mgr.AddRunnable(c)
	return nil
}

func (r *ClusterReconciler) findObjectsForCluster(objNode client.Object) []reconcile.Request {
	// node deleted,ignore event
	if !objNode.GetDeletionTimestamp().IsZero() {
		return []reconcile.Request{}
	}

	node, err := r.NodeLister.Get(objNode.GetName())
	if err != nil {
		return []reconcile.Request{}
	}
	fip := node.Annotations[common.AnnotationMetadataFloatIP]
	role := node.Labels[common.LabelNodeRole]
	// if master node has fip, maybe we need update the cluster's kubeconfig.
	if fip != "" && role == string(common.NodeRoleMaster) {
		clusterName := node.Labels[common.LabelClusterName]
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				Name: clusterName,
			}},
		}
	}
	return []reconcile.Request{}
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	if clu.Status.Phase == v1.ClusterUnbinding || clu.Status.Phase == v1.ClusterUnbindFailed {
		err := r.unbindCluster(clu)
		if err != nil {
			logger.Errorf("unbind cluster(%s) failed: %v", clu.Name, err)
			return ctrl.Result{}, err
		}

		logger.Infof("update external cluster status to '%s' successfully", v1.ClusterImportFailed)
		return ctrl.Result{}, nil
	}

	if clu.ObjectMeta.DeletionTimestamp.IsZero() {
		if !sets.NewString(clu.ObjectMeta.Finalizers...).Has(v1.ClusterFinalizer) {
			clu.ObjectMeta.Finalizers = append(clu.ObjectMeta.Finalizers, v1.ClusterFinalizer)
			if clu, err = r.ClusterWriter.UpdateCluster(context.TODO(), clu); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		go func() {
			r.mgr.RemoveClusterClientSet(clu.Name)
		}()
		// The object is being deleted
		if sets.NewString(clu.ObjectMeta.Finalizers...).Has(v1.ClusterFinalizer) {
			err = r.updateClusterNode(ctx, clu, true)
			if err != nil && !errors.IsNotFound(err) {
				log.Error("Failed to update cluster node", zap.Error(err))
				return ctrl.Result{}, err
			}
			err = r.CronBackupWriter.DeleteCronBackupCollection(ctx, &query.Query{FieldSelector: fmt.Sprintf("spec.clusterName=%s", clu.Name)})
			if err != nil {
				log.Error("Failed to delete cronBackup", zap.Error(err))
				return ctrl.Result{}, err
			}
			err = r.OperationWriter.DeleteOperationCollection(ctx, &query.Query{LabelSelector: fmt.Sprintf("%s=%s", common.LabelClusterName, clu.Name)})
			if err != nil {
				log.Error("Failed to delete operation", zap.Error(err))
				return ctrl.Result{}, err
			}
			// remove our cluster finalizer
			finalizers := sets.NewString(clu.ObjectMeta.Finalizers...)
			finalizers.Delete(v1.ClusterFinalizer)
			clu.ObjectMeta.Finalizers = finalizers.List()
			if _, err = r.ClusterWriter.UpdateCluster(ctx, clu); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if err = r.updateClusterNode(ctx, clu, false); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.syncClusterClient(ctx, log, clu)
}

func (r *ClusterReconciler) updateClusterNode(ctx context.Context, c *v1.Cluster, del bool) error {
	for _, item := range c.Workers {
		if err := r.updateNodeRoleLabel(ctx, c.Name, item.ID, common.NodeRoleWorker, del); err != nil {
			return err
		}
	}
	for _, item := range c.Masters {
		if err := r.updateNodeRoleLabel(ctx, c.Name, item.ID, common.NodeRoleMaster, del); err != nil {
			return err
		}
	}
	return nil
}

func (r *ClusterReconciler) updateNodeRoleLabel(ctx context.Context, clusterName, nodeName string, role common.NodeRole, del bool) error {
	node, err := r.NodeLister.Get(nodeName)
	if err != nil {
		return err
	}
	if del {
		// check node role label exist.
		// if existed, delete label and update node.
		// if not return direct.
		if _, ok := node.Labels[common.LabelNodeRole]; !ok {
			return nil
		}
		if v, ok := node.Labels[common.LabelClusterName]; !ok || v != clusterName {
			return nil
		}
		delete(node.Labels, common.LabelNodeRole)
		delete(node.Labels, common.LabelClusterName)
	} else {
		// check node role label exist.
		// if existed, return direct
		// if not add label and update node.
		if _, ok := node.Labels[common.LabelNodeRole]; ok {
			return nil
		}
		node.Labels[common.LabelNodeRole] = string(role)
		node.Labels[common.LabelClusterName] = clusterName
	}

	if _, err = r.NodeWriter.UpdateNode(ctx, node); err != nil {
		return err
	}
	return nil
}

func (r *ClusterReconciler) syncClusterClient(ctx context.Context, log logger.Logging, c *v1.Cluster) error {
	if c.Status.Phase == v1.ClusterInstalling || c.Status.Phase == v1.ClusterInstallFailed {
		return nil
	}
	var (
		kubeconfig string
		err        error
	)
	// get kubeconfig from kc
	kubeconfig, err = r.getKubeConfig(ctx, c)
	if err != nil {
		log.Error("create cluster client config failed", zap.String("cluster", c.Name), zap.Error(err))
		return err
	}

	// needn't update
	if kubeconfig == "" {
		return nil
	}

	c.KubeConfig = []byte(kubeconfig)
	_, err = r.ClusterWriter.UpdateCluster(ctx, c)
	if err != nil {
		log.Error("update kube config failed", zap.String("cluster", c.Name), zap.Error(err))
		return err
	}
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		log.Error("create cluster client config failed", zap.String("cluster", c.Name), zap.Error(err))
		return err
	}
	clientcfg, err := clientConfig.ClientConfig()
	if err != nil {
		log.Error("get cluster kubeconfig client failed",
			zap.String("cluster", c.Name), zap.Error(err))
		return err
	}
	clientset, err := kubernetes.NewForConfig(clientcfg)
	if err != nil {
		log.Error("create cluster clientset failed", zap.String("cluster", c.Name), zap.Error(err))
		return err
	}
	r.mgr.AddClusterClientSet(c.Name, client.NewKubernetesClient(clientcfg, clientset))
	return nil
}

func (r *ClusterReconciler) getKubeConfig(ctx context.Context, c *v1.Cluster) (string, error) {
	log := logger.FromContext(ctx)

	node, err := r.NodeLister.Get(c.Masters[0].ID)
	if err != nil {
		log.Error("get master node error",
			zap.String("node", c.Masters[0].ID), zap.String("cluster", c.Name), zap.Error(err))
		return "", err
	}
	// there is 3 ip maybe used in kubeconfig,sort by priority: proxyServer > floatIP > defaultIP
	proxyAPIServer := node.Annotations[common.AnnotationMetadataProxyAPIServer]
	floatIP := node.Annotations[common.AnnotationMetadataFloatIP]
	apiServer := node.Status.Ipv4DefaultIP + ":6443"
	if floatIP != "" {
		apiServer = floatIP + ":6443"
	}
	if proxyAPIServer != "" {
		apiServer = proxyAPIServer
	}
	// kubeconfig already used same ip, do nothing.
	// else we need use current ip to generate a new kubeconfig.
	if _, exist := r.mgr.GetClusterClientSet(c.Name); exist && c.KubeConfig != nil && strings.Contains(string(c.KubeConfig), apiServer) {
		log.Debug("clientset has been init")
		return "", nil
	}

	token, err := r.CmdDelivery.DeliverCmd(ctx, c.Masters[0].ID,
		[]string{"/bin/bash", "-c", `kubectl get secret $(kubectl get sa kc-server -n kube-system -o jsonpath={.secrets[0].name}) -n kube-system -o jsonpath={.data.token} | base64 -d`}, 3*time.Second)
	if err != nil {
		log.Error("get cluster service account token error", zap.Error(err))
		return "", err
	}
	log.Debug("get cluster kc-server service account token", zap.String("token", string(token)))
	if string(token) == "" {
		return "", fmt.Errorf("get invalid token")
	}
	kubeconfig := getKubeConfig(c.Name, fmt.Sprintf("https://%s", apiServer), "kc-server", string(token))
	return kubeconfig, nil
}

var kubeconfigFormat = `
apiVersion: v1
clusters:
- cluster:
    insecure-skip-tls-verify: true
    server: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s-%s
current-context: %s-%s
kind: Config
preferences: {}
users:
- name: %s
  user:
    token: %s`

func getKubeConfig(clusterName string, address string, user string, token string) string {
	return fmt.Sprintf(kubeconfigFormat, address, clusterName, clusterName, user, user, clusterName, user, clusterName, user, token)
}

func (r *ClusterReconciler) unbindCluster(clu *v1.Cluster) error {
	for _, master := range clu.Masters {
		err := utils.ClusterServiceAccount(r.CmdDelivery, master.ID, v1.ActionUninstall)
		if err != nil {
			logger.Warnf("node(%s) delete clusterServiceAccount failed: %v", master.ID, err)
			continue
		}
		break
	}

	cmd := "rm -rf /usr/local/bin/etcdctl && rm -rf /etc/kubeclipper-agent && systemctl disable kc-agent --now"
	nodes := append(clu.Masters, clu.Workers...)
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
	go func() {
		<-time.After(time.Second * 3)
		defer cancelFunc()
	}()
	for i, _ := range nodes {
		go func(no v1.WorkerNode) {
			_, err := r.NodeLister.Get(no.ID)
			if errors.IsNotFound(err) {
				return
			}
			_, err = r.CmdDelivery.DeliverCmd(ctx, no.ID, []string{"/bin/bash", "-c", cmd}, 3*time.Second)
			if err != nil {
				logger.Warnf("clean node(%s) kc-agent service failed: %v", no.ID, err)
				return
			}
		}(nodes[i])
	}

	return r.deleteClusterData(clu)
}

func (r *ClusterReconciler) deleteClusterData(clu *v1.Cluster) error {
	nodeIPs := make([]string, 0)
	nodes := append(clu.Masters, clu.Workers...)
	for _, no := range nodes {
		node, err := r.NodeLister.Get(no.ID)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		nodeIPs = append(nodeIPs, node.Status.Ipv4DefaultIP)
	}

	deploy, err := r.ConfigMapOperator.GetConfigMapEx(context.TODO(), constatns.DeployConfigConfigMapName, "0")
	if err != nil {
		logger.Errorf("get deploy config failed: %v", err)
		return err
	}
	confString := deploy.Data[constatns.DeployConfigConfigMapKey]
	deployConf := &options.DeployConfig{}
	err = yaml.Unmarshal([]byte(confString), deployConf)
	if err != nil {
		logger.Errorf("deploy config unmarshal failed: %v", err)
		return err
	}
	for _, ip := range nodeIPs {
		deployConf.Agents.Delete(ip)
	}
	dcData, err := yaml.Marshal(deployConf)
	if err != nil {
		logger.Errorf("deploy config marshal failed: %v", err)
		return err
	}
	deploy.Data[constatns.DeployConfigConfigMapKey] = string(dcData)
	_, err = r.ConfigMapOperator.UpdateConfigMap(context.TODO(), deploy)
	if err != nil {
		logger.Errorf("update deploy config failed: %v", err)
		return err
	}

	if clu.Annotations[common.AnnotationConfigMap] != "" {
		err = r.ConfigMapOperator.DeleteConfigMap(context.TODO(), clu.Annotations[common.AnnotationConfigMap])
		if err != nil && !errors.IsNotFound(err) {
			logger.Errorf("delete import-cluster(%s) config failed: %v", clu.Name, err)
			return err
		}
	}

	for _, no := range nodes {
		err := r.NodeWriter.DeleteNode(context.TODO(), no.ID)
		if err != nil && !errors.IsNotFound(err) {
			logger.Warnf("delete unbundled-cluster node failed: %v", err)
			return nil
		}
	}

	clu.Status.Phase = v1.ClusterUnbundled
	clu, err = r.ClusterWriter.UpdateCluster(context.TODO(), clu)
	if err != nil {
		return fmt.Errorf("error updating an unbundled cluster: %v", err)
	}

	err = r.ClusterWriter.DeleteCluster(context.TODO(), clu.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error deleting an unbundled cluster: %v", err)
	}

	return nil
}
