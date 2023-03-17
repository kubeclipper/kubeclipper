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
	"crypto/x509"
	"encoding/json"
	buildinerrors "errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	pkgerr "github.com/pkg/errors"
	"k8s.io/client-go/util/cert"

	"github.com/kubeclipper/kubeclipper/pkg/component/utils"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"

	"github.com/kubeclipper/kubeclipper/pkg/clusteroperation"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/clustermanage"
	"github.com/kubeclipper/kubeclipper/pkg/clustermanage/kubeadm"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/cri"
	"github.com/kubeclipper/kubeclipper/pkg/service"

	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
)

type ClusterReconciler struct {
	CmdDelivery         service.CmdDelivery
	mgr                 manager.Manager
	ClusterLister       listerv1.ClusterLister
	ClusterWriter       cluster.ClusterWriter
	ClusterOperator     cluster.Operator
	RegistryLister      listerv1.RegistryLister
	NodeLister          listerv1.NodeLister
	NodeWriter          cluster.NodeWriter
	OperationOperator   operation.Operator
	OperationWriter     operation.Writer
	CronBackupWriter    cluster.CronBackupWriter
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
	if err = c.Watch(source.NewKindWithCache(&v1.Node{}, cache),
		handler.EnqueueRequestsFromMapFunc(r.findObjectsForCluster)); err != nil {
		return err
	}

	if err = c.Watch(source.NewKindWithCache(&v1.Registry{}, cache),
		handler.EnqueueRequestsFromMapFunc(r.findRegistryCluster)); err != nil {
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

func (r *ClusterReconciler) findRegistryCluster(obj client.Object) []reconcile.Request {
	reg, ok := obj.(*v1.Registry)
	if !ok {
		return nil
	}
	clusters, err := r.ClusterLister.List(labels.Everything())
	if err != nil {
		return nil
	}
	var res []reconcile.Request
	for _, cluster := range clusters {
		for _, cReg := range cluster.ContainerRuntime.Registries {
			ref := cReg.RegistryRef
			if ref != nil && *ref == reg.Name {
				res = append(res, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: cluster.Namespace,
						Name:      cluster.Name,
					},
				})
				break
			}
		}
	}
	return res
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
	if err = r.syncClusterClient(ctx, log, clu); err != nil {
		return ctrl.Result{}, nil
	}
	if err = r.processPendingOperations(ctx, log, clu); err != nil {
		log.Error("process pending operation error", zap.Error(err))
		return ctrl.Result{}, nil
	}
	if err = r.updateAPIServerCerts(ctx, clu); err != nil {
		log.Error("update apiServer cert error", zap.Error(err))
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.updateCRIRegistries(ctx, clu)
}

func (r *ClusterReconciler) updateClusterNode(ctx context.Context, c *v1.Cluster, del bool) error {
	for _, item := range c.Workers {
		if err := r.updateNodeRoleLabel(ctx, c.Name, item.ID, common.NodeRoleWorker, del); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	for _, item := range c.Masters {
		if err := r.updateNodeRoleLabel(ctx, c.Name, item.ID, common.NodeRoleMaster, del); err != nil && !errors.IsNotFound(err) {
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

	provider, ok := c.Labels[common.LabelClusterProviderName]
	if ok && provider != kubeadm.ProviderKubeadm {
		kubeconfig, err = r.getKubeconfigFromProvider(c)
		if err != nil {
			log.Error("create cluster client config from provider failed", zap.String("cluster", c.Name), zap.Error(err))
			return err
		}
	} else {
		// get kubeconfig from kc
		kubeconfig, err = r.getKubeConfig(ctx, c)
		if err != nil {
			log.Error("create cluster client config failed", zap.String("cluster", c.Name), zap.Error(err))
			return err
		}
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

func (r *ClusterReconciler) getKubeconfigFromProvider(c *v1.Cluster) (string, error) {
	// clientset has been init
	if _, exist := r.mgr.GetClusterClientSet(c.Name); exist {
		return "", nil
	}

	// get kubeconfig from provider
	providerName := c.Labels[common.LabelClusterProviderName]
	provider, err := r.CloudProviderLister.Get(providerName)
	if err != nil {
		return "", err
	}

	cp, err := clustermanage.GetProvider(clustermanage.Operator{}, *provider)
	if err != nil {
		return "", err
	}
	kubeconfig, err := cp.GetKubeConfig(context.TODO(), c.Name)
	if err != nil {
		return "", err
	}
	return kubeconfig, nil

}

// TODO: need redesign
// clientset need local lb
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
	apiServer := node.Status.NodeIpv4DefaultIP + ":6443"
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

func (r *ClusterReconciler) updateAPIServerCerts(ctx context.Context, c *v1.Cluster) error {
	log := logger.FromContext(ctx)

	if c.Status.Phase != v1.ClusterRunning {
		return nil
	}
	nodeList, err := r.canUpdate(c) // if some master node is unavailable,can't update cert.
	if err != nil {
		return err
	}
	needUpdate, err := r.needUpdate(ctx, c)
	if err != nil {
		return err
	}
	if !needUpdate {
		log.Infof("cluster %s api server cert included all sans %v,skip update", c.Name, c.GetAllCertSANs())
		return nil
	}
	return r.doUpdateAPIServerCerts(ctx, c, nodeList)
}

func (r *ClusterReconciler) doUpdateAPIServerCerts(ctx context.Context, c *v1.Cluster, nl component.NodeList) error {
	sanOperation := generateUpdateSANOperation(c, nl)

	_, err := r.OperationOperator.CreateOperation(ctx, sanOperation)
	if err != nil {
		return pkgerr.WithMessage(err, "create san update operation failed")
	}

	c.Status.Phase = v1.ClusterUpdating
	_, err = r.ClusterOperator.UpdateCluster(ctx, c)
	if err != nil {
		return pkgerr.WithMessage(err, "create san update operation,update cluster status failed")
	}

	return nil
}

func generateUpdateSANOperation(c *v1.Cluster, nodeList component.NodeList) *v1.Operation {
	san := &k8s.SAN{}
	op := &v1.Operation{
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
			Labels: map[string]string{
				common.LabelClusterName:     c.Name,
				common.LabelTimeoutSeconds:  v1.DefaultOperationTimeoutSecs,
				common.LabelOperationAction: v1.OperationUpdateAPIServerCertification,
			}},
		Steps: nil,
		Status: v1.OperationStatus{
			Status:     v1.OperationStatusPending, // operator controller will deliver it
			Conditions: nil,
		},
	}
	nodes := utils.UnwrapNodeList(nodeList)
	op.Steps, _ = san.InstallSteps(nodes, c.GetAllCertSANs())
	return op
}

func (r *ClusterReconciler) needUpdate(ctx context.Context, c *v1.Cluster) (bool, error) {
	for _, nodeID := range c.Masters.GetNodeIDs() {
		certificate, err := getCert(ctx, r.CmdDelivery, nodeID)
		if err != nil {
			return false, err
		}
		// if current cert not included all san，need update it
		if !includeAllSan(c, certificate) {
			return true, nil
		}
	}
	return false, nil
}

func (r *ClusterReconciler) canUpdate(c *v1.Cluster) (component.NodeList, error) {
	nl := make(component.NodeList, 0, len(c.Masters))
	for _, nodeID := range c.Masters.GetNodeIDs() {
		node, err := r.NodeLister.Get(nodeID)
		if err != nil {
			return nil, err
		}
		nl = append(nl, component.Node{
			ID:       node.Name,
			IPv4:     node.Status.Ipv4DefaultIP,
			NodeIPv4: node.Status.NodeIpv4DefaultIP,
			Region:   node.Labels[common.LabelTopologyRegion],
			Hostname: node.Status.NodeInfo.Hostname,
			Role:     node.Labels[common.LabelNodeRole],
			Disable:  false,
		})
	}
	masters, err := nl.AvailableKubeMasters()
	if err != nil {
		return nil, err
	}
	if len(masters) != len(nl) {
		return nil, buildinerrors.New("there is an unavailable master node in the cluster, please check the cluster master node status")
	}
	return nl, nil
}

func getCert(ctx context.Context, cmdDelivery service.CmdDelivery, nodeID string) (*x509.Certificate, error) {
	content, err := cmdDelivery.DeliverCmd(ctx, nodeID, []string{"cat", "/etc/kubernetes/pki/apiserver.crt"}, time.Second*30)
	if err != nil {
		return nil, pkgerr.WithMessagef(err, "cat /etc/kubernetes/pki/apiserver.crt on %s failed", nodeID)
	}
	certificates, err := cert.ParseCertsPEM(content)
	if err != nil {
		return nil, pkgerr.WithMessagef(err, "parse /etc/kubernetes/pki/apiserver.crt from %s,data:%s failed", nodeID, content)
	}
	if len(certificates) != 1 {
		return nil, fmt.Errorf("invlaid apiserver cert")
	}
	return certificates[0], nil
}

// includeAllSan check is cert included all sans
func includeAllSan(c *v1.Cluster, cert *x509.Certificate) bool {
	set := sets.NewString(cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		set.Insert(ip.String())
	}
	sans := c.GetAllCertSANs()
	for _, san := range sans {
		if !set.Has(san) {
			return false
		}
	}
	return true
}

func (r *ClusterReconciler) updateCRIRegistries(ctx context.Context, c *v1.Cluster) error {
	registries, err := r.getClusterCRIRegistries(c)
	if err != nil {
		return err
	}
	switch c.Status.Phase {
	case v1.ClusterInstalling:
		if !registriesEqual(c.Status.Registries, registries) {
			newSpec, err := r.ClusterWriter.UpdateCluster(ctx, c)
			if err != nil {
				return fmt.Errorf("update cluster:%w", err)
			}
			*c = *newSpec
		}
		return nil
	case v1.ClusterRunning:
		break
	default:
		return nil
	}

	if !registriesEqual(c.Status.Registries, registries) {
		c.Status.Registries = registries

		clusterSelector, err := labels.NewRequirement(common.LabelClusterName, selection.Equals, []string{c.Name})
		if err != nil {
			return fmt.Errorf("new cluster node selector requirement:%w", err)
		}
		nodes, err := r.NodeLister.List(labels.NewSelector().Add(*clusterSelector))
		if err != nil {
			return fmt.Errorf("list")
		}

		step, err := criRegistryUpdateStep(c, registries, nodes)
		if err != nil {
			return fmt.Errorf("criRegistryUpdateOperation:%w", err)
		}
		err = r.CmdDelivery.DeliverStep(ctx, step, &service.Options{DryRun: false})
		if err != nil {
			return fmt.Errorf("DeliverTaskOperation:%w", err)
		}
		newSpec, err := r.ClusterWriter.UpdateCluster(ctx, c)
		if err != nil {
			return fmt.Errorf("update cluster:%w", err)
		}
		*c = *newSpec
	}
	return nil
}

func (r *ClusterReconciler) getClusterCRIRegistries(c *v1.Cluster) ([]v1.RegistrySpec, error) {
	type mirror struct {
		ImageRepoMirror string `json:"imageRepoMirror"`
	}
	insecureRegistry := append([]string{}, c.ContainerRuntime.InsecureRegistry...)
	sort.Strings(insecureRegistry)
	// add addons mirror registry
	for _, a := range c.Addons {
		var m mirror
		err := json.Unmarshal(a.Config.Raw, &m)
		if err != nil {
			continue
		}
		if m.ImageRepoMirror != "" {
			idx, ok := sort.Find(len(insecureRegistry), func(i int) int {
				return strings.Compare(m.ImageRepoMirror, insecureRegistry[i])
			})
			if !ok {
				if idx == len(insecureRegistry) {
					insecureRegistry = append(insecureRegistry, m.ImageRepoMirror)
				} else {
					insecureRegistry = append(insecureRegistry[:idx+1], insecureRegistry[idx:]...)
					insecureRegistry[idx] = m.ImageRepoMirror
				}
			}
		}
	}

	registries := make([]v1.RegistrySpec, 0, len(insecureRegistry)*2+len(c.ContainerRuntime.Registries))
	// insecure registry
	for _, host := range insecureRegistry {
		registries = appendUniqueRegistry(registries,
			v1.RegistrySpec{Scheme: "http", Host: host},
			v1.RegistrySpec{Scheme: "https", Host: host, SkipVerify: true})
	}

	validRegistries := c.ContainerRuntime.Registries[:0]
	for _, reg := range c.ContainerRuntime.Registries {
		if reg.RegistryRef == nil || *reg.RegistryRef == "" {
			// fix reg.RegistryRef=""
			reg.RegistryRef = nil
			registries = appendUniqueRegistry(registries,
				v1.RegistrySpec{Scheme: "http", Host: reg.InsecureRegistry},
				v1.RegistrySpec{Scheme: "https", Host: reg.InsecureRegistry, SkipVerify: true})
			validRegistries = append(validRegistries, reg)
			continue
		}
		registry, err := r.RegistryLister.Get(*reg.RegistryRef)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get registry %s:%w", *reg.RegistryRef, err)
		}
		registries = appendUniqueRegistry(registries, registry.RegistrySpec)
		validRegistries = append(validRegistries, reg)
	}
	c.ContainerRuntime.Registries = validRegistries
	return registries, nil
}

func criRegistryUpdateStep(cluster *v1.Cluster, registries []v1.RegistrySpec, nodes []*v1.Node) (*v1.Step, error) {
	var (
		step     component.StepRunnable
		identity string
	)
	switch cluster.ContainerRuntime.Type {
	case v1.CRIDocker:
		identity = cri.DockerInsecureRegistryConfigureIdentity
		step = &cri.DockerInsecureRegistryConfigure{
			InsecureRegistry: cri.ToDockerInsecureRegistry(registries),
		}
	case v1.CRIContainerd:
		identity = cri.ContainerdRegistryConfigureIdentity
		step = &cri.ContainerdRegistryConfigure{
			Registries: cri.ToContainerdRegistryConfig(registries),
			// TODO: get from config
			ConfigDir: cri.ContainerdDefaultRegistryConfigDir,
		}
	default:
		return nil, fmt.Errorf("unknown CRI type:%s", cluster.ContainerRuntime.Type)
	}
	stepData, err := json.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("step marshal:%w", err)
	}
	var allNodes []v1.StepNode
	for _, node := range nodes {
		allNodes = append(allNodes, v1.StepNode{
			ID:       node.Name,
			IPv4:     node.Status.Ipv4DefaultIP,
			NodeIPv4: node.Status.NodeIpv4DefaultIP,
			Hostname: node.Labels[common.LabelHostname],
		})
	}
	return &v1.Step{
		Name:   "update-cri-registry-config",
		Nodes:  allNodes,
		Action: v1.ActionInstall,
		Timeout: metav1.Duration{
			Duration: time.Second * 30,
		},
		Commands: []v1.Command{
			{
				Type:          v1.CommandCustom,
				Identity:      identity,
				CustomCommand: stepData,
			},
		},
	}, nil
}

// Use scheme and host as unique key

func appendUniqueRegistry(s []v1.RegistrySpec, items ...v1.RegistrySpec) []v1.RegistrySpec {
	for _, r := range items {
		key := r.Scheme + r.Host
		idx, ok := sort.Find(len(s), func(i int) int {
			return strings.Compare(key, s[i].Scheme+s[i].Host)
		})
		if !ok {
			if idx == len(s) {
				s = append(s, r)
			} else {
				s = append(s[:idx+1], s[idx:]...)
				s[idx] = r
			}
		}
	}
	return s
}

func registriesEqual(a, b []v1.RegistrySpec) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func (r *ClusterReconciler) processPendingOperations(ctx context.Context, log logger.Logging, c *v1.Cluster) error {
	log.Debug("process pending operations")

	// all pending operations are processed
	clean := true

	for _, pendingOperation := range c.PendingOperations {
		log.Debugf("pending operation info is %+v, extraData is %+v", pendingOperation, string(pendingOperation.ExtraData))
		// the operation has been processed ?
		op, err := r.OperationOperator.GetOperationEx(ctx, pendingOperation.OperationID, "0")
		// ignore IsNotFound error
		if err != nil && !errors.IsNotFound(err) {
			// if an error occurs, the deletion operation will not be performed
			clean = false
			continue
		}
		// if no, the operation will be created
		if op == nil {
			clean = false

			// get cluster information about the specified resource version when the operation is performed
			clu, err := r.ClusterOperator.GetClusterEx(ctx, c.Name, pendingOperation.ClusterResourceVersion)
			if err != nil {
				log.Error(fmt.Sprintf("get resource-version-%s cluster info failed", pendingOperation.ClusterResourceVersion),
					zap.String("cluster", c.Name), zap.String("operation-id", pendingOperation.OperationID), zap.Error(err))
				continue
			}

			// restore and assemble the cluster metadata, passing it through
			extraMeta, err := r.assembleClusterExtraMetadata(ctx, clu)
			if err != nil {
				log.Error("get cluster extra metadata failed", zap.String("cluster", c.Name), zap.String("operation-id", pendingOperation.OperationID), zap.Error(err))
				continue
			}

			// pass-through operationID
			extraMeta.OperationID = pendingOperation.OperationID

			// build the operation structure based on the type of operation
			newOperation, err := clusteroperation.BuildOperationAdapter(clu, pendingOperation, extraMeta, nil)
			if err != nil {
				log.Error("create operation struct failed", zap.String("cluster", c.Name), zap.String("operation-id", pendingOperation.OperationID), zap.Error(err))
				continue
			}

			log.Debugf("create operation struct successful", zap.String("cluster", c.Name), zap.String("operation-id", pendingOperation.OperationID))
			_, err = r.OperationWriter.CreateOperation(ctx, newOperation)
			if err != nil {
				log.Error("create operation failed", zap.String("cluster", c.Name), zap.String("operation-id", pendingOperation.OperationID), zap.Error(err))
				continue
			}
		}
	}

	if clean && len(c.PendingOperations) > 0 {
		log.Debugf("clean pending operation in %s cluster", c.Name)
		// clean pending operations
		c.PendingOperations = nil
		if _, err := r.ClusterWriter.UpdateCluster(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
