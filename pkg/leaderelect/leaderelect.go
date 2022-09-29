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

package leaderelect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/lease"
)

var _ resourcelock.Interface = (*Lock)(nil)

type Lock struct {
	// LeaseMeta should contain a Name and a Namespace of a
	// LeaseMeta object that the LeaderElector will attempt to lead.
	LeaseMeta   metav1.ObjectMeta
	leaseClient lease.Operator
	lease       *coordinationv1.Lease
	config      resourcelock.ResourceLockConfig
}

func NewLock(ns string, name string, leaseOperator lease.Operator, rlc resourcelock.ResourceLockConfig) resourcelock.Interface {
	return &Lock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		leaseClient: leaseOperator,
		lease:       nil,
		config:      rlc,
	}
}

func (l *Lock) Get(ctx context.Context) (*resourcelock.LeaderElectionRecord, []byte, error) {
	var err error
	l.lease, err = l.leaseClient.GetLeaseWithNamespace(ctx, l.LeaseMeta.Name, l.LeaseMeta.Namespace)
	if err != nil {
		logger.Error("get lease failed", zap.Error(err))
		return nil, nil, err
	}
	record := leaseSpecToLeaderElectionRecord(&l.lease.Spec)
	recordByte, err := json.Marshal(*record)
	if err != nil {
		logger.Error("marshal lease record error", zap.Error(err))
		return nil, nil, err
	}
	return record, recordByte, nil
}

func (l *Lock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	var err error
	l.lease, err = l.leaseClient.CreateLease(ctx, &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l.LeaseMeta.Name,
			Namespace: l.LeaseMeta.Namespace,
		},
		Spec: leaderElectionRecordToLeaseSpec(&ler),
	})
	if err != nil {
		return err
	}
	return nil
}

func (l *Lock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	if l.lease == nil {
		return errors.New("lease not initialized, call get or create first")
	}
	l.lease.Spec = leaderElectionRecordToLeaseSpec(&ler)
	updateObj := l.lease.DeepCopy()
	var err error
	l.lease, err = l.leaseClient.UpdateLease(ctx, updateObj)
	if err != nil {
		return err
	}
	return nil
}

func (l *Lock) RecordEvent(s string) {
	if l.config.EventRecorder == nil {
		return
	}
	// TODO: always ignore record event
}

func (l *Lock) Identity() string {
	return l.config.Identity
}

func (l *Lock) Describe() string {
	return fmt.Sprintf("%v/%v", l.LeaseMeta.Namespace, l.LeaseMeta.Name)
}

func leaseSpecToLeaderElectionRecord(spec *coordinationv1.LeaseSpec) *resourcelock.LeaderElectionRecord {
	var r resourcelock.LeaderElectionRecord
	if spec.HolderIdentity != nil {
		r.HolderIdentity = *spec.HolderIdentity
	}
	if spec.LeaseDurationSeconds != nil {
		r.LeaseDurationSeconds = int(*spec.LeaseDurationSeconds)
	}
	if spec.LeaseTransitions != nil {
		r.LeaderTransitions = int(*spec.LeaseTransitions)
	}
	if spec.AcquireTime != nil {
		r.AcquireTime = metav1.Time{Time: spec.AcquireTime.Time}
	}
	if spec.RenewTime != nil {
		r.RenewTime = metav1.Time{Time: spec.RenewTime.Time}
	}
	return &r
}

func leaderElectionRecordToLeaseSpec(ler *resourcelock.LeaderElectionRecord) coordinationv1.LeaseSpec {
	leaseDurationSeconds := int32(ler.LeaseDurationSeconds)
	leaseTransitions := int32(ler.LeaderTransitions)
	return coordinationv1.LeaseSpec{
		HolderIdentity:       &ler.HolderIdentity,
		LeaseDurationSeconds: &leaseDurationSeconds,
		AcquireTime:          &metav1.MicroTime{Time: ler.AcquireTime.Time},
		RenewTime:            &metav1.MicroTime{Time: ler.RenewTime.Time},
		LeaseTransitions:     &leaseTransitions,
	}
}
