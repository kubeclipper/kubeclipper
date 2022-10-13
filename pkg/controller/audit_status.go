package controller

import (
	"context"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/auditing/option"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/platform"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

const (
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
	mgr.AddWorkerLoop(s.monitorAuditStatus, auditStatusMonitorPeriod)
}

func (s *AuditStatusMon) monitorAuditStatus() {
	timestamp := time.Now().Add(-s.AuditOptions.EventLogHistoryRetentionPeriod).Format(time.RFC3339)
	s.cleanLogs("type", timestamp)
	s.cleanLogs("type!", timestamp)
}

func (s *AuditStatusMon) cleanLogs(fieldSelector, timestamp string) {
	q := &query.Query{
		FieldSelector: fieldSelector,
		Reverse:       false,
		Limit:         -1,
		Pagination: &query.Pagination{
			Limit:  -1,
			Offset: 0,
		},
	}
	loginLogs, err := s.AuditOperator.ListEvents(context.TODO(), q)
	if err != nil {
		logger.Errorf("list event log error: %s", err.Error())
		return
	}

	if len(loginLogs.Items) >= s.AuditOptions.EventLogHistoryMaximumEntries {
		q.FuzzySearch = map[string]string{
			"time": timestamp,
		}
		eventList, err := s.AuditOperator.ListEventsWithTimeEx(context.TODO(), q)
		if err != nil {
			logger.Errorf("list event log with time error: %s", err.Error())
			return
		}
		for _, item := range eventList.Items {
			event := item.(*v1.Event)
			if err = s.AuditOperator.DeleteEvent(context.TODO(), event.Name); err != nil {
				logger.Errorf(err.Error())
			}

		}
	}
}
