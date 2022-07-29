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

package dnscontroller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/utils/sliceutil"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/hashutil"
)

type DNSReconciler struct {
	DomainLister  listerv1.DomainLister
	DomainWriter  cluster.DNSWriter
	ClusterLister listerv1.ClusterLister
	mgr           manager.Manager
}

func (r *DNSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	domain, err := r.DomainLister.Get(req.Name)
	if err != nil {
		// domain not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get domain with name", zap.Error(err))
		return ctrl.Result{}, err
	}

	if domain.ObjectMeta.DeletionTimestamp.IsZero() {
		// 更新 count 字段
		if domain.Status.Count != int64(len(domain.Spec.Records)) {
			domain.Status.Count = int64(len(domain.Spec.Records))
			if _, err = r.DomainWriter.UpdateDomain(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err = r.syncDomainCluster(ctx, domain); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.syncConfigMap(ctx, log) // sync coredns configmap
}

func (r *DNSReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("dns", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("dns-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Domain{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Cluster{}, cache), handler.EnqueueRequestsFromMapFunc(r.findObjectsForCluster)); err != nil {
		return err
	}
	r.mgr = mgr
	mgr.AddRunnable(c)
	return nil
}

func (r *DNSReconciler) syncDomainCluster(ctx context.Context, domain *v1.Domain) error {
	removed := sets.NewString()
	for _, cluName := range domain.Spec.SyncCluster {
		clu, err := r.ClusterLister.Get(cluName)
		if err != nil {
			return err
		}
		if errors.IsNotFound(err) || !clu.ObjectMeta.DeletionTimestamp.IsZero() {
			// cluster is deleted or deleting
			removed.Insert(cluName)
		}
	}
	if removed.Len() == 0 {
		return nil
	}
	domain.Spec.SyncCluster = sliceutil.RemoveString(domain.Spec.SyncCluster, func(item string) bool {
		return removed.Has(item)
	})
	_, err := r.DomainWriter.UpdateDomain(ctx, domain)
	return err
}

func (r *DNSReconciler) syncConfigMap(ctx context.Context, log logger.Logging) error {
	// sync dns records to coredns configmap
	originDomains, err := r.DomainLister.List(labels.Everything())
	if err != nil {
		log.Error("failed to list domains", zap.Error(err))
		return err
	}
	m := buildDomain(originDomains)

	clusters, err := r.ClusterLister.List(labels.Everything())
	if err != nil {
		log.Error("failed to list clusters", zap.Error(err))
		return err
	}
	for _, clu := range clusters {
		cc, ok := r.mgr.GetClusterClientSet(clu.Name)
		if !ok {
			return fmt.Errorf("get cluster client failed")
		}
		cli := cc.Kubernetes()
		svc, err := cli.CoreV1().Services("kube-system").Get(context.TODO(), "kube-dns", metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get coredns svc failed: %s", err.Error())
		}
		domains := m[clu.Name]
		corefile, err := renderCorefile(domains, svc.Spec.ClusterIP, clu.Networking.DNSDomain)
		if err != nil {
			return fmt.Errorf("failed to render corefile: %s", err.Error())
		}
		log.Debugf("cluster name:%s new corefile:%v", clu.Name, corefile)

		configmap, err := cli.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "coredns", metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get coredns configmap failed: %s", err.Error())
		}

		if equal(configmap.Data["Corefile"], corefile) {
			log.Debugf("cluster name:%s corefile not changed,skip update", clu.Name)
			continue
		}
		configmap.Data["Corefile"] = corefile
		_, err = cli.CoreV1().ConfigMaps("kube-system").Update(context.TODO(), configmap, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("update coredns configmap failed: %s", err.Error())
		}

		addonPort := addonPorts(svc.Spec.Ports)
		if len(addonPort) != len(svc.Spec.Ports) {
			svc.Spec.Ports = addonPort
			log.Debugf("update coredns svc cluster:%s svc ports:%+v", clu.Name, svc.Spec.Ports)
			_, err = cli.CoreV1().Services("kube-system").Update(context.TODO(), svc, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("update coredns svc failed: %s", err.Error())
			}
		}
	}
	return nil
}

func (r *DNSReconciler) findObjectsForCluster(clu client.Object) []reconcile.Request {
	if clu.GetDeletionTimestamp() == nil || clu.GetDeletionTimestamp().IsZero() {
		return []reconcile.Request{}
	}
	domains, err := r.DomainLister.List(labels.Everything())
	if err != nil {
		return []reconcile.Request{}
	}
	var requests []reconcile.Request
	for _, domain := range domains {
		if sliceutil.HasString(domain.Spec.SyncCluster, clu.GetName()) {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: domain.Name,
				},
			})
		}
	}
	return requests
}

var (
	UDP5300 = corev1.ServicePort{
		Name:        "5300udp",
		Protocol:    corev1.ProtocolUDP,
		AppProtocol: nil,
		Port:        5300,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 5300,
		},
	}
	TCP5300 = corev1.ServicePort{
		Name:        "5300tcp",
		Protocol:    corev1.ProtocolTCP,
		AppProtocol: nil,
		Port:        5300,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 5300,
		},
	}
	ports = []corev1.ServicePort{UDP5300, TCP5300}
)

// addonPorts fills in the dns ports if they are not explicitly configured.
func addonPorts(data []corev1.ServicePort) []corev1.ServicePort {
	m := make(map[string]struct{}, len(data))
	for _, v := range data {
		m[v.Name] = struct{}{}
	}
	for _, port := range ports {
		if _, ok := m[port.Name]; !ok {
			data = append(data, port)
		}
	}
	return data
}

func buildDomain(domains []*v1.Domain) map[string][]*v1.Domain {
	// group by cluster name
	m := make(map[string][]*v1.Domain)
	for _, domain := range domains {
		for _, clusterName := range domain.Spec.SyncCluster {
			m[clusterName] = append(m[clusterName], domain)
		}
	}
	return m
}

func equal(old, new string) bool {
	return hashutil.MD5(old) == hashutil.MD5(new)
}
