/*
 *
 *  * Copyright 2024 KubeClipper Authors.
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

package controller

import (
	"context"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/auditing/option"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
)

var (
	auditStatusMonitorPeriod = 30 * time.Minute
)

type AuditStatusMon struct {
	AuditOperator platform.Operator
	mgr           manager.Manager
	logger        logger.Logging
	AuditOptions  *option.AuditOptions
}

func (s *AuditStatusMon) SetupWithManager(mgr manager.Manager) {
	s.mgr = mgr
	s.logger = mgr.GetLogger().WithName("audit-status-monitor")
	if s.AuditOptions.RetentionPeriod > auditStatusMonitorPeriod {
		auditStatusMonitorPeriod = s.AuditOptions.RetentionPeriod / 2
	}
	mgr.AddWorkerLoop(s.monitorAuditStatus, auditStatusMonitorPeriod)
}

func (s *AuditStatusMon) monitorAuditStatus() {
	timestamp := time.Now().Add(-s.AuditOptions.RetentionPeriod)
	// clean audit operation / login record
	s.clean("type=", timestamp)
	s.clean("type!=", timestamp)
}

func (s *AuditStatusMon) clean(fieldSelector string, timestamp time.Time) {
	// clean operation audit event
	q := &query.Query{
		FieldSelector: fieldSelector,
		Reverse:       true,
		Pagination:    query.NoPagination(),
	}

	eventList, err := s.AuditOperator.ListEventsEx(context.TODO(), q)
	if err != nil {
		logger.Errorf("list event log with time error: %s", err.Error())
		return
	}

	deviation := len(eventList.Items) - s.AuditOptions.MaximumEntries
	if deviation > 0 {
		for _, item := range eventList.Items[:deviation] {
			event := item.(*v1.Event)
			if err = s.AuditOperator.DeleteEvent(context.TODO(), event.Name); err != nil {
				logger.Errorf(err.Error())
			}
		}

		for _, item := range eventList.Items[deviation:] {
			event := item.(*v1.Event)
			if !timestamp.After(event.RequestReceivedTimestamp.Time) {
				break
			}
			if err = s.AuditOperator.DeleteEvent(context.TODO(), event.Name); err != nil {
				logger.Errorf(err.Error())
			}
		}

	}
}
