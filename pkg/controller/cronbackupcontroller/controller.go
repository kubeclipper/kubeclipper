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

package cronbackupcontroller

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/types"

	"github.com/google/uuid"
	"go.uber.org/zap"
	apimachineryErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/kubeclipper/kubeclipper/pkg/client/informers"
	listerv1 "github.com/kubeclipper/kubeclipper/pkg/client/lister/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/component"
	ctrl "github.com/kubeclipper/kubeclipper/pkg/controller-runtime"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/controller"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/handler"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/source"
	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1/k8s"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	bs "github.com/kubeclipper/kubeclipper/pkg/simple/backupstore"
)

type CronBackupReconciler struct {
	ClusterWriter     cluster.ClusterWriter
	ClusterLister     listerv1.ClusterLister
	NodeLister        listerv1.NodeLister
	BackupLister      listerv1.BackupLister
	OperationWriter   operation.Writer
	BackupWriter      cluster.BackupWriter
	BackupPointLister listerv1.BackupPointLister
	CronBackupLister  listerv1.CronBackupLister
	CronBackupWriter  cluster.CronBackupWriter
	CmdDelivery       service.CmdDelivery
	Now               func() time.Time
}

func (r *CronBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logger.FromContext(ctx)
	cronBackup, err := r.CronBackupLister.Get(req.Name)
	if err != nil {
		// cronBackup not found, possibly been deleted
		// need to do the cleanup
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error("Failed to get cronBackup with name", zap.Error(err))
		return ctrl.Result{}, err
	}

	_, cErr := r.ClusterLister.Get(cronBackup.Spec.ClusterName)
	if cErr != nil && !errors.IsNotFound(cErr) {
		log.Error("unexpected error, cronBackups require a cluster to be specified")
		return ctrl.Result{}, cErr
	}

	if _, ok := cronBackup.Labels[common.LabelCronBackupDisable]; ok {
		return ctrl.Result{}, nil
	}

	now := metav1.NewTime(r.Now())
	if cronBackup.Spec.RunAt != nil {
		// first created cron backup, update next schedule time
		if cronBackup.Status.NextScheduleTime == nil {
			cronBackup.Status.NextScheduleTime = cronBackup.Spec.RunAt
			_, err = r.CronBackupWriter.UpdateCronBackup(ctx, cronBackup)
			if err != nil {
				log.Error("Failed to update cronBackup", zap.Error(err))
			}
			return ctrl.Result{}, err
		}
		if cronBackup.Status.LastSuccessfulTime == nil {
			if now.After(cronBackup.Spec.RunAt.Time) {
				// delivery the create backup operation
				err = r.createBackup(log, cronBackup)
				if err != nil {
					log.Error("Failed to delivery operation to create backup", zap.Error(err))
					return ctrl.Result{}, err
				}
				cronBackup.Status.LastScheduleTime = cronBackup.Status.NextScheduleTime
				cronBackup.Status.LastSuccessfulTime = cronBackup.Status.LastScheduleTime
				_, err = r.CronBackupWriter.UpdateCronBackup(ctx, cronBackup)
				if err != nil {
					log.Error("Failed to update cronBackup", zap.Error(err))
				}
				return ctrl.Result{}, nil
			}
			sub := cronBackup.Spec.RunAt.Sub(now.Time)
			return ctrl.Result{
				RequeueAfter: sub,
			}, nil
		}
	}

	if cronBackup.Spec.Schedule != "" {
		// first created cron backup, update next schedule time
		if cronBackup.Status.NextScheduleTime == nil {
			schedule := ParseSchedule(cronBackup.Spec.Schedule, r.Now())
			s, _ := cron.NewParser(4 | 8 | 16 | 32 | 64).Parse(schedule)
			// update the next schedule time
			nextRunAt := metav1.NewTime(s.Next(time.Now()))
			cronBackup.Status.NextScheduleTime = &nextRunAt
			_, err = r.CronBackupWriter.UpdateCronBackup(ctx, cronBackup)
			if err != nil {
				log.Error("Failed to update cronBackup", zap.Error(err))
			}
			return ctrl.Result{}, err
		}
		// time to create backup
		if after := now.After(cronBackup.Status.NextScheduleTime.Time); after {
			// delivery the create backup operation
			err = r.createBackup(log, cronBackup)
			if err != nil {
				cronBackup.Status.LastScheduleTime = &now
				log.Error("Failed to delivery operation to create backup", zap.Error(err))
				// update cronBackup
				_, err = r.CronBackupWriter.UpdateCronBackup(ctx, cronBackup)
				if err != nil {
					log.Error("Failed to update cronBackup", zap.Error(err))
				}
				return ctrl.Result{}, err
			}
			cronBackup.Status.LastScheduleTime = &now
			cronBackup.Status.LastSuccessfulTime = cronBackup.Status.LastScheduleTime

			schedule := ParseSchedule(cronBackup.Spec.Schedule, r.Now())
			s, _ := cron.NewParser(4 | 8 | 16 | 32 | 64).Parse(schedule)
			// update the next schedule time
			nextRunAt := metav1.NewTime(s.Next(now.Time))
			cronBackup.Status.NextScheduleTime = &nextRunAt
			// update cronBackup
			_, err = r.CronBackupWriter.UpdateCronBackup(ctx, cronBackup)
			if err != nil {
				log.Error("Failed to update cronBackup", zap.Error(err))
			}

			// keep the backup num within the maxNum
			backupList, err := r.BackupLister.List(labels.Everything())
			if err != nil {
				log.Error("Failed to list backups of the cronBackup", zap.Error(err))
				return ctrl.Result{}, err
			}
			backupsByCb := GroupBackupsByParent(backupList)
			if len(backupsByCb[cronBackup.UID]) > cronBackup.Spec.MaxBackupNum {
				// sort the backups by creation time
				sort.Slice(backupsByCb[cronBackup.UID], func(i, j int) bool {
					return backupsByCb[cronBackup.UID][i].CreationTimestamp.Time.Before(backupsByCb[cronBackup.UID][j].CreationTimestamp.Time)
				})
				sub := len(backupsByCb[cronBackup.UID]) - cronBackup.Spec.MaxBackupNum
				for del := 0; del < sub; del++ {
					if backupsByCb[cronBackup.UID][del].Status.ClusterBackupStatus != v1.ClusterBackupAvailable {
						continue
					}
					err = r.BackupWriter.DeleteBackup(ctx, backupsByCb[cronBackup.UID][del].Name)
					if err != nil {
						log.Error("Failed to delete backup", zap.Error(err))
						return ctrl.Result{}, err
					}
					// delivery the create backup operation
					err = r.deleteBackup(log, backupsByCb[cronBackup.UID][del].Labels[common.LabelClusterName], backupsByCb[cronBackup.UID][del])
					if err != nil {
						log.Error("Failed to delivery operation to delete backup", zap.Error(err))
						return ctrl.Result{}, err
					}
				}
			}
		}
		sub := cronBackup.Status.NextScheduleTime.Sub(now.Time)
		return ctrl.Result{
			RequeueAfter: sub,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *CronBackupReconciler) SetupWithManager(mgr manager.Manager, cache informers.InformerCache) error {
	c, err := controller.NewUnmanaged("cronbackup", controller.Options{
		MaxConcurrentReconciles: 2,
		Reconciler:              r,
		Log:                     mgr.GetLogger().WithName("cronbackup-controller"),
		RecoverPanic:            true,
	})
	if err != nil {
		return err
	}
	if err = c.Watch(source.NewKindWithCache(&v1.CronBackup{}, cache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	mgr.AddRunnable(c)
	return nil
}

func (r *CronBackupReconciler) createBackup(log logger.Logging, cronBackup *v1.CronBackup) error {
	trueVar := true
	rfs := []metav1.OwnerReference{
		{
			APIVersion: "core.kubeclipper.io/v1",
			Kind:       "CronBackup",
			Name:       cronBackup.Name,
			UID:        cronBackup.UID,
			Controller: &trueVar,
		},
	}
	backup := &v1.Backup{}
	backup.SetOwnerReferences(rfs)
	c, err := r.ClusterLister.Get(cronBackup.Spec.ClusterName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("Cluster is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get cluster", zap.Error(err))
		return err
	}

	if c.Status.Phase != v1.ClusterRunning {
		log.Warnf("the cluster is %v, create backup in next reconcile", c.Status.Phase)
		return err
	}

	randNum := rand.String(6)
	backup.Name = fmt.Sprintf("%s-%s-%s", c.Name, cronBackup.Name, randNum)
	b, err := r.BackupLister.Get(backup.Name)
	if err != nil && !apimachineryErrors.IsNotFound(err) {
		log.Error("Failed to get backup", zap.Error(err))
		return err
	}
	if b != nil {
		log.Errorf("backup %s already exists", backup.Name)
		return fmt.Errorf("backup %s already exists", backup.Name)
	}

	// check the backup point exist or not
	if c.Labels[common.LabelBackupPoint] == "" {
		log.Error("no backup point specified")
		return fmt.Errorf("no backup point specified")
	}

	bp, err := r.BackupPointLister.Get(c.Labels[common.LabelBackupPoint])
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("backupPoint is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get backupPoint", zap.Error(err))
		return err
	}

	backup.Status.KubernetesVersion = c.KubernetesVersion
	backup.Status.FileName = backup.Name
	backup.BackupPointName = c.Labels[common.LabelBackupPoint]
	// check preferred node in cluster
	if backup.PreferredNode == "" {
		backup.PreferredNode = c.Masters[0].ID
	}
	if len(c.Masters.Intersect(v1.WorkerNode{
		ID: backup.PreferredNode,
	})) == 0 {
		log.Errorf("the node %s not a master node", backup.PreferredNode)
		return fmt.Errorf("the node %s not a master node", backup.PreferredNode)
	}

	requirement, err := labels.NewRequirement(common.LabelClusterName, selection.Equals, []string{c.Name})
	if err != nil {
		log.Error("Failed to create a requirement", zap.Error(err))
		return err
	}
	nodeList, err := r.NodeLister.List(labels.NewSelector().Add(*requirement))
	if err != nil {
		log.Error("Failed to list nodes in the cluster", zap.Error(err))
		return err
	}

	if backup.PreferredNode != "" {
		_, err := r.NodeLister.Get(backup.PreferredNode)
		if err != nil {
			log.Error("Failed to get node by name", zap.Error(err))
			return err
		}
	}

	// create operation
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = make(map[string]string)
	op.Labels[common.LabelOperationAction] = v1.OperationBackupCluster
	op.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	op.Labels[common.LabelClusterName] = c.Name
	op.Labels[common.LabelBackupName] = backup.Name
	op.Labels[common.LabelTopologyRegion] = c.Masters[0].Labels[common.LabelTopologyRegion]
	op.Status.Status = v1.OperationStatusRunning
	// add backup
	backup.Labels = make(map[string]string)
	backup.Labels[common.LabelClusterName] = c.Name
	backup.Labels[common.LabelOperationName] = op.Name
	backup.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	backup.ClusterNodes = make(map[string]string)

	for _, node := range nodeList {
		backup.ClusterNodes[node.Status.Ipv4DefaultIP] = node.Status.NodeInfo.Hostname
	}

	steps := make([]v1.Step, 0)
	var actBackup *k8s.ActBackup
	var masters component.NodeList
	for _, no := range nodeList {
		masters = append(masters, component.Node{
			ID:       no.Name,
			IPv4:     no.Status.Ipv4DefaultIP,
			NodeIPv4: no.Status.NodeIpv4DefaultIP,
			Region:   no.Labels[common.LabelTopologyRegion],
			Hostname: no.Labels[common.LabelHostname],
			Role:     no.Labels[common.LabelNodeRole],
			//Disable:  no.Labels[common.LabelNodeDisable]
		})
	}
	meta := component.ExtraMetadata{
		ClusterName: c.Name,
		Masters:     masters,
	}
	meta.Masters = append(meta.Masters, component.Node{ID: backup.PreferredNode})
	ctx := component.WithExtraMetadata(context.TODO(), meta)

	switch bp.StorageType {
	case bs.S3Storage:
		actBackup = &k8s.ActBackup{
			StoreType:       bp.StorageType,
			BackupFileName:  backup.Status.FileName,
			AccessKeyID:     bp.S3Config.AccessKeyID,
			AccessKeySecret: bp.S3Config.AccessKeySecret,
			Bucket:          bp.S3Config.Bucket,
			Endpoint:        bp.S3Config.Endpoint,
		}
	case bs.FSStorage:
		actBackup = &k8s.ActBackup{
			StoreType:          bp.StorageType,
			BackupFileName:     backup.Status.FileName,
			BackupPointRootDir: bp.FsConfig.BackupRootDir,
		}
	}

	if err = actBackup.InitSteps(ctx); err != nil {
		log.Error("Failed to init steps", zap.Error(err))
		return err
	}

	nc, err := r.ClusterLister.Get(cronBackup.Spec.ClusterName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("Cluster is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get cluster", zap.Error(err))
		return err
	}

	if nc.Status.Phase != v1.ClusterRunning {
		log.Warnf("the cluster is %v, create backup in next reconcile", c.Status.Phase)
		return err
	}

	// update cluster status to backing_up
	c.Status.Phase = v1.ClusterBackingUp

	actBackupStep := actBackup.GetStep(v1.ActionInstall)
	op.Steps = append(steps, actBackupStep...)
	op, err = r.OperationWriter.CreateOperation(ctx, op)
	if err != nil {
		log.Error("Failed to create operation", zap.Error(err))
		return err
	}
	_, err = r.BackupWriter.CreateBackup(ctx, backup)
	if err != nil {
		log.Error("Failed to create backup", zap.Error(err))
		return err
	}
	_, err = r.ClusterWriter.UpdateCluster(context.TODO(), c)
	if err != nil {
		log.Error("Failed to update cluster", zap.Error(err))
		return err
	}
	// delivery the create backup operation
	go func() {
		err = r.CmdDelivery.DeliverTaskOperation(ctx, op, &service.Options{DryRun: false})
		log.Error("Failed to delivery operation", zap.Error(err))
	}()
	return nil
}

func GroupBackupsByParent(bs []*v1.Backup) map[types.UID][]*v1.Backup {
	BackupsByCb := make(map[types.UID][]*v1.Backup)
	for _, backup := range bs {
		parentUID, found := GetParentUIDFromBackup(backup)
		if !found {
			logger.Errorf("Unable to get parent uid from backup %s in namespace %s", backup.Name, backup.Namespace)
			continue
		}
		BackupsByCb[parentUID] = append(BackupsByCb[parentUID], backup)
	}
	return BackupsByCb
}

// GetParentUIDFromBackup extracts UID of job's parent and whether it was found
func GetParentUIDFromBackup(b *v1.Backup) (types.UID, bool) {
	controllerRef := metav1.GetControllerOf(b)

	if controllerRef == nil {
		return types.UID(""), false
	}

	if controllerRef.Kind != "CronBackup" {
		logger.Errorf("Unable to get parent uid from backup %s in namespace %s", b.Name, b.Namespace)
		return types.UID(""), false
	}

	return controllerRef.UID, true
}

func (r *CronBackupReconciler) deleteBackup(log logger.Logging, clusterName string, backup *v1.Backup) error {
	c, err := r.ClusterLister.Get(clusterName)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("Cluster is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get cluster", zap.Error(err))
		return err
	}

	bp, err := r.BackupPointLister.Get(c.Labels[common.LabelBackupPoint])
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("backupPoint is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get backupPoint", zap.Error(err))
		return err
	}

	b, err := r.BackupLister.Get(backup.Name)
	if err != nil {
		if apimachineryErrors.IsNotFound(err) {
			log.Error("Backup is not found", zap.Error(err))
			return err
		}
		log.Error("Failed to get Backup", zap.Error(err))
		return err
	}

	if backup.PreferredNode != "" {
		_, err := r.NodeLister.Get(backup.PreferredNode)
		if err != nil {
			log.Error("Failed to get node by name", zap.Error(err))
			return err
		}
	}

	// create operation
	op := &v1.Operation{}
	op.Name = uuid.New().String()
	op.Labels = make(map[string]string)
	op.Labels[common.LabelOperationAction] = v1.OperationDeleteBackup
	op.Labels[common.LabelTimeoutSeconds] = strconv.Itoa(v1.DefaultBackupTimeoutSec)
	op.Labels[common.LabelClusterName] = c.Name
	op.Labels[common.LabelBackupName] = b.Name
	op.Labels[common.LabelTopologyRegion] = c.Masters[0].Labels[common.LabelTopologyRegion]
	op.Status.Status = v1.OperationStatusRunning

	if b.Status.ClusterBackupStatus == v1.ClusterBackupRestoring || b.Status.ClusterBackupStatus == v1.ClusterBackupCreating {
		return fmt.Errorf("backup is %s now, can't delete", b.Status.ClusterBackupStatus)
	}

	steps := make([]v1.Step, 0)
	var actBackup *k8s.ActBackup
	meta := component.ExtraMetadata{
		ClusterName: c.Name,
	}
	meta.Masters = append(meta.Masters, component.Node{ID: backup.PreferredNode})
	ctx := component.WithExtraMetadata(context.TODO(), meta)

	switch bp.StorageType {
	case bs.S3Storage:
		actBackup = &k8s.ActBackup{
			StoreType:       bp.StorageType,
			BackupFileName:  backup.Status.FileName,
			AccessKeyID:     bp.S3Config.AccessKeyID,
			AccessKeySecret: bp.S3Config.AccessKeySecret,
			Bucket:          bp.S3Config.Bucket,
			Endpoint:        bp.S3Config.Endpoint,
		}
	case bs.FSStorage:
		actBackup = &k8s.ActBackup{
			StoreType:          bp.StorageType,
			BackupFileName:     backup.Status.FileName,
			BackupPointRootDir: bp.FsConfig.BackupRootDir,
		}
	}

	if err = actBackup.InitSteps(ctx); err != nil {
		log.Error("Failed to init steps", zap.Error(err))
		return err
	}

	actBackupStep := actBackup.GetStep(v1.ActionUninstall)
	op.Steps = append(steps, actBackupStep...)
	op, err = r.OperationWriter.CreateOperation(ctx, op)
	if err != nil {
		log.Error("Failed to create operation", zap.Error(err))
		return err
	}

	// delivery the delete backup operation
	go func() {
		err = r.CmdDelivery.DeliverTaskOperation(ctx, op, &service.Options{DryRun: false})
		log.Error("Failed to delivery operation", zap.Error(err))
	}()
	return nil
}

func ParseSchedule(schedule string, now time.Time) string {
	arr := strings.Split(schedule, " ")
	if arr[2] == "L" {
		first := now.AddDate(0, 0, -now.Day()+1)
		next := first.AddDate(0, 1, -1)
		lastDay := strconv.Itoa(next.Day())
		return strings.Replace(schedule, "L", lastDay, 1)
	}
	return schedule
}
