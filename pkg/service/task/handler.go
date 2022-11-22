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

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/oplog"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/errors"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

func (s *Service) runTaskStep(ctx context.Context, payload *service.MsgPayload, subject string) ([]byte, *errors.StatusError) {
	// stepKey to distinguish which step the log file belongs to
	stepKey := fmt.Sprintf("%s-%s", payload.Step.ID, payload.Step.Name)
	ctx = component.WithOperationID(ctx, payload.OperationIdentity) // put operation ID into context
	ctx = component.WithStepID(ctx, stepKey)                        // put step ID into context
	ctx = component.WithOplog(ctx, s.oplog)                         // put operation log object into context
	ctx = component.WithRepoMirror(ctx, s.repoMirror)

	var entry string
	// truncate step log file
	if payload.Retry {
		entry = fmt.Sprintf("\n\n--------------------> steps retry %s <--------------------\n\n", time.Now().Format(time.RFC3339))
	}
	if err := s.oplog.CreateStepLogFileAndAppend(payload.OperationIdentity, stepKey, []byte(entry)); err != nil {
		// log errors do not affect the execution of main processes
		logger.Error("create operation step log file failed: "+err.Error(),
			zap.String("operation", payload.OperationIdentity),
			zap.String("step", stepKey),
			zap.String("cmd", "run task step"),
			zap.Bool("retry", payload.Retry),
		)
	}
	if payload.Step.AutomaticRetry {
		// the automatic retry step clears the log file
		if err := s.oplog.TruncateStepLogFile(payload.OperationIdentity, stepKey); err != nil {
			logger.Error("truncate step log file failed: "+err.Error(),
				zap.String("operation", payload.OperationIdentity),
				zap.String("step", stepKey),
				zap.String("cmd", "run task step"),
				zap.Bool("retry", payload.Retry),
			)
		}
	}

	cmds := make([]v1.Command, len(payload.Step.BeforeRunCommands)+len(payload.Step.Commands)+len(payload.Step.AfterRunCommands))
	cmds = append(cmds, payload.Step.BeforeRunCommands...)
	cmds = append(cmds, payload.Step.Commands...)
	cmds = append(cmds, payload.Step.AfterRunCommands...)
	var replyData []byte
	for _, c := range cmds {
		switch c.Type {
		case v1.CommandShell:
			logger.Debug("run shell command", zap.Strings("cmd", c.ShellCommand))
			if err := runShellCommand(ctx, c.ShellCommand, payload.DryRun); err != nil {
				errMsg := "run shell command error"
				return nil, doStatusError(errMsg, errMsg, errors.ShellCommand, 500, err)
			}
		case v1.CommandCustom:
			var statusError *errors.StatusError
			if payload.LastTaskReply != nil {
				// Put join command into context
				ctx = component.WithExtraData(ctx, payload.LastTaskReply)
			}
			if replyData, statusError = runCustomCommand(ctx, &payload.Step, c.Identity, c.CustomCommand, payload.DryRun); statusError != nil {
				return nil, statusError
			}
		case v1.CommandTemplateRender:
			if statusError := runTemplateRenderCommand(ctx, c.Template, payload.DryRun); statusError != nil {
				return nil, statusError
			}
		}
	}
	return replyData, nil
}

func (s *Service) runStep(ctx context.Context, payload *service.MsgPayload, subject string) ([]byte, *errors.StatusError) {
	// stepKey to distinguish which step the log file belongs to
	stepKey := fmt.Sprintf("%s-%s", payload.Step.ID, payload.Step.Name)
	ctx = component.WithStepID(ctx, stepKey) // put step ID into context
	ctx = component.WithOplog(ctx, s.oplog)  // put operation log object into context
	ctx = component.WithRepoMirror(ctx, s.repoMirror)

	cmds := make([]v1.Command, len(payload.Step.BeforeRunCommands)+len(payload.Step.Commands)+len(payload.Step.AfterRunCommands))
	cmds = append(cmds, payload.Step.BeforeRunCommands...)
	cmds = append(cmds, payload.Step.Commands...)
	cmds = append(cmds, payload.Step.AfterRunCommands...)
	var replyData []byte
	for _, c := range cmds {
		switch c.Type {
		case v1.CommandShell:
			logger.Debug("run shell command", zap.Strings("cmd", c.ShellCommand))
			if err := runShellCommand(ctx, c.ShellCommand, payload.DryRun); err != nil {
				errMsg := "run shell command error"
				return nil, doStatusError(errMsg, errMsg, errors.ShellCommand, 500, err)
			}
		case v1.CommandCustom:
			var statusError *errors.StatusError
			if payload.LastTaskReply != nil {
				// Put join command into context
				ctx = component.WithExtraData(ctx, payload.LastTaskReply)
			}
			if replyData, statusError = runCustomCommand(ctx, &payload.Step, c.Identity, c.CustomCommand, payload.DryRun); statusError != nil {
				return nil, statusError
			}
		case v1.CommandTemplateRender:
			if statusError := runTemplateRenderCommand(ctx, c.Template, payload.DryRun); statusError != nil {
				return nil, statusError
			}
		}
	}
	return replyData, nil
}

func (s *Service) msgHandler(msg *nats.Msg) {
	go s.taskHandler(msg)
}

func (s *Service) taskHandler(msg *nats.Msg) {
	// TODO: recovery from panic
	logger.Debugf("Got incoming msg subject %s", msg.Subject)
	logger.Debugf("Got incoming msg content %s", string(msg.Data))
	payload := &service.MsgPayload{}
	if err := json.Unmarshal(msg.Data, payload); err != nil {
		logger.Error("unmarshal task payload error", zap.Error(err))
		return
	}
	logger.Debug("in coming task payload", zap.Int("operation", int(payload.Op)),
		zap.String("step", payload.Step.Name), zap.ByteString("lastResponse", payload.LastTaskReply), zap.Duration("timeout", payload.Step.Timeout.Duration))
	ctx, cancel := context.WithTimeout(context.TODO(), payload.Step.Timeout.Duration)
	defer cancel()
	var statusError *errors.StatusError

	switch payload.Op {
	case service.OperationRunCmd:
		var replyData []byte
		logger.Debug("run shell command", zap.Strings("cmd", payload.Cmds))
		ec, err := cmdutil.RunCmdWithContext(ctx, payload.DryRun, payload.Cmds[0], payload.Cmds[1:]...)
		if err != nil {
			errMsg := "run shell command error"
			statusError = doStatusError(errMsg, errMsg, errors.ShellCommand, 500, err)
		}
		replyData = []byte(ec.StdOut())
		responseMessage(msg, replyData, statusError)
	case service.OperationStepLog:
		var replyData []byte
		defer func() {
			logger.Debugf("reply data: %s", string(replyData))
			responseMessage(msg, replyData, statusError)
		}()
		errMsg := "handle step log error"
		req, err := s.parseStepLogOperationID(payload.OperationIdentity)
		if err != nil {
			logger.Error("parse step log message error", zap.Error(err))
			statusError = doStatusError(errMsg, "parse step log message error", errors.StepLog, 500, err)
			return
		}
		content, deliverySize, logSize, err := s.oplog.GetStepLogContent(req.OpID, req.StepID, req.Offset, req.Length)
		if err != nil {
			logger.Error("read log file error", zap.Error(err))
			statusError = doStatusError(errMsg, "read log file error", errors.StepLog, 500, err)
			return
		}
		replyData, err = json.Marshal(oplog.LogContentResponse{
			Content:      string(content),
			LogSize:      logSize,
			DeliverySize: deliverySize,
		})
		if err != nil {
			logger.Error("marshal log content response error", zap.Error(err))
			statusError = doStatusError(errMsg, "marshal log content response error", errors.StepLog, 500, err)
			return
		}
	case service.OperationRunTask:
		var replyData []byte
		for i := 0; i <= int(payload.Step.RetryTimes); i++ {
			// reset retry field
			if i > 0 {
				payload.Retry = true
			}
			replyData, statusError = s.runTaskStep(ctx, payload, msg.Subject)
			if statusError == nil {
				break
			}
			logger.Debug("run task step failed", zap.String("step", payload.Step.Name), zap.Int("retry", i), zap.Int32("maxRetry", payload.Step.RetryTimes))
		}
		responseMessage(msg, replyData, statusError)
	case service.OperationRunStep:
		var replyData []byte
		for i := 0; i <= int(payload.Step.RetryTimes); i++ {
			// reset retry field
			if i > 0 {
				payload.Retry = true
			}
			replyData, statusError = s.runStep(ctx, payload, msg.Subject)
			if statusError == nil {
				break
			}
			logger.Debug("run step failed", zap.String("step", payload.Step.Name), zap.Int("retry", i), zap.Int32("maxRetry", payload.Step.RetryTimes))
		}
		responseMessage(msg, replyData, statusError)
	default:
		responseMessage(msg, nil, &errors.StatusError{
			Message: "unknown operation",
			Reason:  errors.StatusReason(fmt.Sprintf("operation: %d", payload.Op)),
			Code:    500,
		})
	}
}

func runShellCommand(ctx context.Context, cmds []string, dryRun bool) error {
	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, cmds[0], cmds[1:]...)
	return err
}

func runTemplateRenderCommand(ctx context.Context, cmd *v1.TemplateCommand, dryRun bool) *errors.StatusError {
	errMsg := "render template error"
	tmplRender, ok := component.LoadTemplate(cmd.Identity)
	if !ok {
		logger.Errorf("unsupported custom agent step: %s", cmd.Identity)
		return &errors.StatusError{
			Message: errMsg,
			Reason:  "template not found",
			Code:    500,
		}
	}
	implMeta := tmplRender.NewInstance()
	newImpl, _ := implMeta.(component.TemplateRender)
	if err := json.Unmarshal(cmd.Data, newImpl); err != nil {
		logger.Error("unmarshal error", zap.Error(err))
		return doStatusError(errMsg, "unmarshal template renderer error", errors.Unmarshal, 500, err)
	}
	if err := newImpl.Render(ctx, component.Options{DryRun: dryRun}); err != nil {
		logger.Error("render template failed", zap.Error(err))
		return &errors.StatusError{
			Message: "run step commands error",
			Reason:  "render template error",
			Code:    500,
		}
	}
	return nil
}

func runCustomCommand(ctx context.Context, step *v1.Step, identity string, customData []byte, dryRun bool) ([]byte, *errors.StatusError) {
	agentStep, ok := component.LoadAgentStep(identity)
	if !ok {
		reason := fmt.Sprintf("unsupported custom agent step: %s", identity)
		logger.Errorf(reason)
		return nil, &errors.StatusError{
			Message: "run step commands error",
			Reason:  errors.StatusReason(reason),
			Details: nil,
			Code:    500,
		}
	}
	agentStepMeta := agentStep.NewInstance()
	if err := json.Unmarshal(customData, agentStepMeta); err != nil {
		logger.Error("unmarshal error", zap.Error(err))
		return nil, doStatusError("run step commands error", "unmarshal custom agent step error",
			errors.Unmarshal, 500, err)
	}
	newImpl, _ := agentStepMeta.(component.StepRunnable)
	var (
		data   []byte
		err    error
		errMsg = "run custom command error"
	)
	if step.Action == v1.ActionInstall {
		if data, err = newImpl.Install(ctx, component.Options{DryRun: dryRun}); err != nil {
			logger.Error("custom step run error", zap.Error(err))
			return nil, doStatusError(errMsg, "run custom command for installation error",
				errors.AgentStepInstall, 500, err)
		}
	} else {
		if data, err = newImpl.Uninstall(ctx, component.Options{DryRun: dryRun}); err != nil {
			logger.Error("custom step run error", zap.Error(err))
			return nil, doStatusError(errMsg, "run custom command for uninstallation error",
				errors.AgentStepUninstall, 500, err)
		}
	}
	return data, nil
}

func responseMessage(msg *nats.Msg, data []byte, error *errors.StatusError) {
	reply := service.CommonReply{
		Error: error,
		Data:  data,
	}
	replyBytes, err := json.Marshal(reply)
	if err != nil {
		logger.Error("marshal response message failed", zap.Error(err))
		return
	}
	if err := msg.Respond(replyBytes); err != nil {
		logger.Error("respond message failed", zap.Error(err))
	}
}

func doStatusError(errMsg, errReason string, errType errors.CauseType, errCode int32, err error) *errors.StatusError {
	return &errors.StatusError{
		Message: errMsg,
		Reason:  errors.StatusReason(errReason),
		Details: &errors.StatusDetails{
			Causes: []errors.StatusCause{
				{
					Type:    errType,
					Message: err.Error(),
				},
			},
		},
		Code: errCode,
	}
}
