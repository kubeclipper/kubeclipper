package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/authentication/options"
	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/manager"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/iam"
	"github.com/kubeclipper/kubeclipper/pkg/query"
	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/iam/v1"
)

var (
	iamStatusMonitorPeriod = 30 * time.Minute
)

type IAMStatusMon struct {
	IAMOperator        iam.Operator
	AuthenticationOpts *options.AuthenticationOptions
	logger             logger.Logging
}

func (a *IAMStatusMon) SetupWithManager(mgr manager.Manager) {
	a.logger = mgr.GetLogger().WithName("iam-status-monitor")
	if a.AuthenticationOpts.LoginHistoryRetentionPeriod > iamStatusMonitorPeriod {
		iamStatusMonitorPeriod = a.AuthenticationOpts.LoginHistoryRetentionPeriod / 2
	}
	mgr.AddWorkerLoop(a.monitorIAMStatus, iamStatusMonitorPeriod)
}

func (a *IAMStatusMon) monitorIAMStatus() {
	timestamp := time.Now().Add(-a.AuthenticationOpts.LoginHistoryRetentionPeriod)
	q := &query.Query{
		Pagination: query.NoPagination(),
		Reverse:    true,
	}
	userList, err := a.IAMOperator.ListUserEx(context.TODO(), q, true, true)
	if err != nil {
		logger.Errorf("list user error: %s", err.Error())
		return
	}
	for _, item := range userList.Items {
		user := item.(*v1.User)
		q.LabelSelector = fmt.Sprintf("%s=%s", common.LabelUserReference, user.Name)
		loginLogs, err := a.IAMOperator.ListLoginRecordEx(context.TODO(), q)
		if err != nil {
			logger.Errorf("list login record error: %s", err.Error())
			return
		}

		deviation := len(loginLogs.Items) - a.AuthenticationOpts.LoginHistoryMaximumEntries
		if deviation > 0 {
			for _, item := range loginLogs.Items[:deviation] {
				record := item.(*v1.LoginRecord)
				if err = a.IAMOperator.DeleteLoginRecord(context.TODO(), record.Name); err != nil {
					logger.Errorf("delete [%s] login record error: %s", user.Name, err.Error())
					return
				}
			}
			for _, item := range loginLogs.Items[deviation:] {
				record := item.(*v1.LoginRecord)
				if !timestamp.After(record.CreationTimestamp.Time) {
					break
				}
				if err = a.IAMOperator.DeleteLoginRecord(context.TODO(), record.Name); err != nil {
					logger.Errorf("delete [%s] login record error: %s", user.Name, err.Error())
					return
				}
			}
		}

	}
}
