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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"

	"github.com/kubeclipper/kubeclipper/pkg/service"

	"github.com/kubeclipper/kubeclipper/pkg/models/operation"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"

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
	CmdDelivery      service.CmdDelivery
	mgr              manager.Manager
	ClusterLister    listerv1.ClusterLister
	NodeLister       listerv1.NodeLister
	NodeWriter       cluster.NodeWriter
	ClusterWriter    cluster.ClusterWriter
	OperationWriter  operation.Writer
	CronBackupWriter cluster.CronBackupWriter
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
			if err != nil {
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

	node, err := r.NodeLister.Get(c.Masters[0].ID)
	if err != nil {
		log.Error("get master node error",
			zap.String("node", c.Masters[0].ID), zap.String("cluster", c.Name))
		return err
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
		return nil
	}

	token, err := r.CmdDelivery.DeliverCmd(ctx, c.Masters[0].ID,
		[]string{"/bin/bash", "-c", `kubectl get secret $(kubectl get sa kc-server -n kube-system -o jsonpath={.secrets[0].name}) -n kube-system -o jsonpath={.data.token} | base64 -d`}, 3*time.Second)
	if err != nil {
		log.Error("get cluster service account token error", zap.Error(err))
		return err
	}
	log.Debug("get cluster kc-server service account token", zap.String("token", string(token)))
	if string(token) == "" {
		return fmt.Errorf("get invalid token")
	}
	// cacrt, err := s.cmdDelivery.DeliverCmd(context.TODO(), clu.Masters[0].ID, []string{"kubectl", "config", "view", "--raw", "-o", "jsonpath={.clusters[0].cluster..certificate-authority-data}"}, 3*time.Second)
	// if err != nil {
	//	logger.Error("get cluster service account token error", zap.String("cluster", name), zap.Error(err))
	//	return err
	// }
	kubeconfig := getKubeConfig(c.Name, fmt.Sprintf("https://%s", apiServer), "kc-server", string(token))
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
