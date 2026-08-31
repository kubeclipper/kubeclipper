package operationv2

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

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
		return ctrl.Result{}, fmt.Errorf("operation %q target Cluster UID changed", op.Name)
	}
	if op.Spec.Action == corev1.OperationDeleteCluster && op.Status.Phase == operations.OperationSucceeded {
		return ctrl.Result{}, r.Clusters.DeleteCluster(ctx, clusterObject.Name)
	}
	desired := failedClusterPhase(op.Spec.Action)
	if op.Status.Phase == operations.OperationSucceeded {
		desired = corev1.ClusterRunning
	}
	if clusterObject.Status.Phase == desired &&
		(op.Spec.Action != corev1.OperationUpgradeCluster || op.Status.Phase != operations.OperationSucceeded) {
		return ctrl.Result{}, nil
	}
	if err := r.updateCluster(ctx, op, desired); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// updateCluster retries the complete read/modify/write cycle. A retry must
// re-read the object because the previous resourceVersion is no longer valid
// after a concurrent controller update.
func (r *BusinessReconciler) updateCluster(ctx context.Context, op *operations.Operation, desired corev1.ClusterPhase) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		clusterObject, err := r.Clusters.GetClusterEx(ctx, op.Spec.TargetRef.Name, "")
		if err != nil {
			return err
		}
		if clusterObject.UID != op.Spec.TargetRef.UID {
			return fmt.Errorf("operation %q target Cluster UID changed", op.Name)
		}
		if clusterObject.Status.Phase == desired &&
			(op.Spec.Action != corev1.OperationUpgradeCluster || op.Status.Phase != operations.OperationSucceeded) {
			return nil
		}
		clusterObject = clusterObject.DeepCopy()
		clusterObject.Status.Phase = desired
		if op.Spec.Action == corev1.OperationUpgradeCluster && op.Status.Phase == operations.OperationSucceeded {
			clusterObject.KubernetesVersion = op.Labels[common.LabelUpgradeVersion]
		}
		_, err = r.Clusters.UpdateCluster(ctx, clusterObject)
		return err
	})
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
	case corev1.OperationAddNodes, corev1.OperationRemoveNodes:
		// Node membership changes can leave a partially applied cluster state.
		// Keep the cluster usable and expose the failure on the Operation itself;
		// forcing UpdateFailed would unnecessarily block unrelated workflows.
		return corev1.ClusterRunning
	default:
		return corev1.ClusterUpdateFailed
	}
}

func (r *BusinessReconciler) SetupWithManager(mgr manager.Manager, factory informers.SharedInformerFactory) error {
	informer := factory.Operations().V1alpha1().Operations()
	informer.Informer()
	managedController, err := controller.NewUnmanaged("operation-v2-business", controller.Options{
		MaxConcurrentReconciles: 2, Reconciler: r,
		Log: mgr.GetLogger().WithName("operation-v2-business-controller"), RecoverPanic: true,
	})
	if err != nil {
		return err
	}
	operationSource := source.NewKindWithCache(&operations.Operation{}, factory)
	if err := managedController.Watch(operationSource, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(managedController)
	return nil
}
