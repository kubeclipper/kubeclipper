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

package service

import (
	"context"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/oplog"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type Runnable interface {
	PrepareRun(stopCh <-chan struct{}) error
	Run(stopCh <-chan struct{}) error
	Close()
}

type Interface interface {
	Runnable
}

type IDelivery interface {
	DeliverLogRequest(ctx context.Context, operation *LogOperation) (oplog.LogContentResponse, error) // request & response synchronously.
	CmdDelivery
}

type CmdDelivery interface {
	DeliverTaskOperation(ctx context.Context, operation *v1.Operation, opts *Options) error
	DeliverStep(ctx context.Context, operation *v1.Step, opts *Options) error
	DeliverCmd(ctx context.Context, toNode string, cmds []string, timeout time.Duration) ([]byte, error)
}

func HandlerCrash() {
	if r := recover(); r != nil {
		logger.Error("handler crash", zap.Any("recover_for", r))
	}
}
