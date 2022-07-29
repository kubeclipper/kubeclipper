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

package backupcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/reconcile"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
)

type BackupReconciler struct {
	ClusterLister   listerv1.ClusterLister
	BackupLister    listerv1.BackupLister
	OperationLister listerv1.OperationLister
	BackupWriter    cluster.BackupWriter
}

func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)

	b, err := r.BackupLister.Get(req.Name)
	if err != nil {
		// backup not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get backup with name", zap.Error(err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.updateBackupStatus(ctx, log, b)
}

func (r *BackupReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("backup", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("backup-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Backup{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.Operation{}, cache), handler.EnqueueRequestsFromMapFunc(r.findObjectsForOperation)); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}

func (r *BackupReconciler) updateBackupStatus(ctx context.Context, log logger.Logging, b *v1.Backup) error {
	c, cErr := r.ClusterLister.Get(b.Labels[common.LabelClusterName])
	if cErr != nil && !errors.IsNotFound(cErr) {
		log.Warn("unexpected error, backup should always has a cluster name label")
		return cErr
	}

	o, oErr := r.OperationLister.Get(b.Labels[common.LabelOperationName])
	if oErr != nil && !errors.IsNotFound(oErr) {
		log.Warn("unexpected error, backup should always has a operation name label", zap.String("operation err", oErr.Error()))
		return oErr
	}

	// if cluster not exist, backup will be deleted, operation-informer will delete this operation
	if errors.IsNotFound(cErr) {
		err := r.BackupWriter.DeleteBackup(context.TODO(), b.Name)
		if err != nil {
			log.Warnf("backup(%s) delete failed: %s", b.Name, err.Error())
		}
		return err
	}

	// if operation not exist, backup change to error
	if oErr != nil && errors.IsNotFound(oErr) {
		b.Status.ClusterBackupStatus = v1.ClusterBackupError
		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync status failed: %s", b.Name, err.Error())
			return err
		}
		return nil
	}

	// when the action of creating backup is timed out, set the cluster status to running, and the backup status to error
	if b.Status.ClusterBackupStatus == v1.ClusterBackupCreating {
		timeout := checkBackupTimeout(log, b)
		if timeout {
			b.Status.ClusterBackupStatus = v1.ClusterBackupError
			_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
			if err != nil {
				log.Warnf("backup(%s) sync status failed: %s", b.Name, err.Error())
			}
		}
	}

	// when the operation status is running and the action is backup cluster, set the backup status to creating
	if o != nil && o.Status.Status == v1.OperationStatusRunning && o.Labels[common.LabelOperationAction] == v1.OperationBackupCluster {
		b.Status.ClusterBackupStatus = v1.ClusterBackupCreating
		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync operation(%s) status failed: %s", b.Name, o.Name, err.Error())
		}
	}

	// when the operation status is running and the action is recovery cluster, set the backup status to restoring
	if o != nil && o.Status.Status == v1.OperationStatusRunning && o.Labels[common.LabelOperationAction] == v1.OperationRecoverCluster {
		b.Status.ClusterBackupStatus = v1.ClusterBackupRestoring
		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync operation(%s) status failed: %s", b.Name, o.Name, err.Error())
		}
	}

	// when the backup is creating and operation is successful, set the backup status to available
	if b.Status.ClusterBackupStatus == v1.ClusterBackupCreating && o != nil && o.Status.Status == v1.OperationStatusSuccessful && o.Labels[common.LabelOperationAction] == v1.OperationBackupCluster {
		checkFile := k8s.CheckFile{}
		if o.Status.Conditions != nil {
			if o.Status.Conditions[0].Status != nil {
				err := json.Unmarshal(o.Status.Conditions[0].Status[0].Response, &checkFile)
				if err != nil {
					log.Errorf("get backup file size and md5 value failed: %s", err.Error())
				}
			}
		}
		if checkFile.BackupFileSize != int64(0) && checkFile.BackupFileMD5 != "" {
			b.Status.BackupFileSize = checkFile.BackupFileSize
			b.Status.BackupFileMD5 = checkFile.BackupFileMD5
		} else {
			log.Warnf("backup file size is %s, and backup md5 is %s, reconcile again", checkFile.BackupFileSize, checkFile.BackupFileMD5)
			return fmt.Errorf("backup file size is %d, and backup md5 is %s", checkFile.BackupFileSize, checkFile.BackupFileMD5)
		}
		b.Status.ClusterBackupStatus = v1.ClusterBackupAvailable
		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync status failed: %s", b.Name, err.Error())
		}
	}

	// when the backup is restoring and operation is successful, set the backup status to available
	if b.Status.ClusterBackupStatus == v1.ClusterBackupRestoring && o != nil && o.Status.Status == v1.OperationStatusSuccessful && o.Labels[common.LabelOperationAction] == v1.OperationRecoverCluster {
		b.Status.ClusterBackupStatus = v1.ClusterBackupAvailable
		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync status failed: %s", b.Name, err.Error())
		}
	}

	// when the operation status is failed, set the backup status to error
	if o != nil && o.Status.Status == v1.OperationStatusFailed {
		if c.Status.Phase == v1.ClusterRestoreFailed {
			b.Status.ClusterBackupStatus = v1.ClusterBackupAvailable
		} else {
			b.Status.ClusterBackupStatus = v1.ClusterBackupError
		}

		_, err := r.BackupWriter.UpdateBackup(context.TODO(), b)
		if err != nil {
			log.Warnf("backup(%s) sync status failed: %s", b.Name, err.Error())
		}
	}

	return nil
}

func (r *BackupReconciler) findObjectsForOperation(clu client.Object) []reconcile.Request {
	backups, err := r.BackupLister.List(labels.Everything())
	if err != nil {
		return []reconcile.Request{}
	}
	var requests []reconcile.Request
	for _, backup := range backups {
		if backup.Labels[common.LabelOperationName] == clu.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: backup.Name,
				},
			})
		}
	}
	return requests
}

func checkBackupTimeout(log logger.Logging, b *v1.Backup) bool {
	if b.Labels[common.LabelTimeoutSeconds] == "" {
		log.Warn("unexpected error, backup should always has a timeout label. will be considered as timeout")
		return true
	}

	createTime := b.CreationTimestamp
	nowTime := time.Now()
	subTime := nowTime.Sub(createTime.Time)

	log.Debugf("timeout value: %s, create time: %s, current time: %s, sub time is %s",
		b.Labels[common.LabelTimeoutSeconds], createTime.String(), nowTime.String(), subTime.String())

	return subTime.Seconds() > float64(v1.DefaultBackupTimeoutSec)
}
