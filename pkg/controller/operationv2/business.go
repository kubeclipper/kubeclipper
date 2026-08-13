package operationv2

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	operationslister "github.com/kubeclipper/kubeclipper/pkg/client/lister/operations/v1alpha1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

type BusinessReconciler struct {
	Operations operationslister.OperationLister
	Clusters   cluster.Operator
}

func (r *BusinessReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	op, err := r.Operations.Get(request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if !op.Status.Phase.IsTerminal() {
		return ctrl.Result{}, nil
	}
	clusterObject, err := r.Clusters.GetClusterEx(ctx, op.Spec.TargetRef.Name, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if clusterObject.UID != op.Spec.TargetRef.UID {
		return ctrl.Result{}, fmt.Errorf("Operation %q target Cluster UID changed", op.Name)
	}
	if op.Spec.Action == corev1.OperationDeleteCluster && op.Status.Phase == operations.OperationSucceeded {
		return ctrl.Result{}, r.Clusters.DeleteCluster(ctx, clusterObject.Name)
	}
	desired := failedClusterPhase(op.Spec.Action)
	if op.Status.Phase == operations.OperationSucceeded {
		desired = corev1.ClusterRunning
	}
	if clusterObject.Status.Phase == desired && !(op.Spec.Action == corev1.OperationUpgradeCluster && op.Status.Phase == operations.OperationSucceeded) {
		return ctrl.Result{}, nil
	}
	clusterObject = clusterObject.DeepCopy()
	clusterObject.Status.Phase = desired
	if op.Spec.Action == corev1.OperationUpgradeCluster && op.Status.Phase == operations.OperationSucceeded {
		clusterObject.KubernetesVersion = op.Labels[common.LabelUpgradeVersion]
	}
	_, err = r.Clusters.UpdateCluster(ctx, clusterObject)
	return ctrl.Result{}, err
}

func failedClusterPhase(action string) corev1.ClusterPhase {
	switch action {
	case corev1.OperationCreateCluster:
		return corev1.ClusterInstallFailed
	case corev1.OperationDeleteCluster:
		return corev1.ClusterTerminateFailed
	case corev1.OperationUpgradeCluster:
		return corev1.ClusterUpgradeFailed
	case corev1.OperationRecoverCluster:
		return corev1.ClusterRestoreFailed
	default:
		return corev1.ClusterUpdateFailed
	}
}

func (r *BusinessReconciler) SetupWithManager(mgr manager.Manager, factory informers.SharedInformerFactory) error {
	informer := factory.Operations().V1alpha1().Operations()
	informer.Informer()
	controller, err := controller.NewUnmanaged("operation-v2-business", controller.Options{
		MaxConcurrentReconciles: 2, Reconciler: r,
		Log: mgr.GetLogger().WithName("operation-v2-business-controller"), RecoverPanic: true,
	})
	if err != nil {
		return err
	}
	if err := controller.Watch(source.NewKindWithCache(&operations.Operation{}, factory), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(controller)
	return nil
}
