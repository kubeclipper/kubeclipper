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

package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/controller"

	"github.com/kubeclipper/kubeclipper/pkg/oplog"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"

	"github.com/kubeclipper/kubeclipper/pkg/component"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/cluster"
	"github.com/kubeclipper/kubeclipper/pkg/models/lease"
	"github.com/kubeclipper/kubeclipper/pkg/models/operation"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/natsio"
)

var _ service.Interface = (*Service)(nil)
var _ service.IDelivery = (*Service)(nil)

const (
	updateOperationStatusRetry = 10
)

type stepStatus struct {
	OperationIdentity  string
	OperationCondition v1.OperationCondition
	DryRun             bool
}

type Service struct {
	external          bool
	nodeReportSubject string
	queueGroup        string
	subjectSuffix     string
	client            natsio.Interface
	clusterOperator   cluster.Operator
	leaseOperator     lease.Operator
	opOperator        operation.Operator
	stepStatusChan    chan stepStatus
	terminationChan   *chan struct{}
}

func NewService(opts *natsio.NatsOptions, clusterOperator cluster.Operator, leaseOperator lease.Operator, opOperator operation.Operator, terminationChan *chan struct{}) *Service {
	s := &Service{
		external:          opts.External,
		client:            natsio.NewNats(opts),
		subjectSuffix:     opts.Client.SubjectSuffix,
		nodeReportSubject: opts.Client.NodeReportSubject,
		queueGroup:        opts.Client.QueueGroupName,
		clusterOperator:   clusterOperator,
		leaseOperator:     leaseOperator,
		opOperator:        opOperator,
		stepStatusChan:    make(chan stepStatus, 256),
		terminationChan:   terminationChan,
	}
	s.client.SetReconnectHandler(s.defaultMQReconnectHandler)
	s.client.SetDisconnectErrHandler(s.defaultMQDisconnectHandler)
	s.client.SetErrorHandler(s.defaultMQErrorHandler)
	s.client.SetClosedHandler(s.defaultMQClosedHandler)
	return s
}

func (s *Service) defaultMQReconnectHandler(conn *nats.Conn) {
	logger.Debug("message queue reconnecting...", zap.Uint64("reconnect", conn.Reconnects))
}

func (s *Service) defaultMQDisconnectHandler(conn *nats.Conn, err error) {
	logger.Error("message queue disconnect with error", zap.Error(err))
}

func (s *Service) defaultMQErrorHandler(conn *nats.Conn, subscription *nats.Subscription, err error) {
	logger.Error("message queue handler error", zap.Error(err),
		zap.String("subj", subscription.Subject), zap.String("queue_group", subscription.Queue))
}

func (s *Service) defaultMQClosedHandler(conn *nats.Conn) {
	logger.Debug("message queue closed ...")
}

func (s *Service) Run(stopCh <-chan struct{}) error {
	if !s.external {
		if err := s.client.RunServer(stopCh); err != nil {
			return err
		}
	}
	// wait server start
	// TODO: add method checkServerRunning
	time.Sleep(1 * time.Second)
	if err := s.client.InitConn(stopCh); err != nil {
		return err
	}
	if err := s.client.QueueSubscribe(s.nodeReportSubject, s.queueGroup, s.nodeStateReportInHandler); err != nil {
		return err
	}
	go s.stepStatusChannelController()
	return nil
}

func (s *Service) PrepareRun(stopCh <-chan struct{}) error {
	return nil
}

func (s *Service) Close() {
	s.client.Close()
}

func initPayload(operationIdentity string, operation service.Operation, step *v1.Step, lastStepReply []byte, cmds []string, dryRun, retry bool) ([]byte, error) {
	payload := service.MsgPayload{
		Op:                operation,
		OperationIdentity: operationIdentity,
		DryRun:            dryRun,
		Retry:             retry,
		Cmds:              cmds,
	}
	if step != nil {
		payload.Step = *step
	}
	if lastStepReply != nil {
		payload.LastTaskReply = lastStepReply
	}
	return json.Marshal(payload)
}

func (s *Service) stepStatusChannelController() {
	for status := range s.stepStatusChan {
		if status.DryRun {
			logger.Debug("dry run update step status", zap.String("op", status.OperationIdentity),
				zap.String("step", status.OperationCondition.StepID), zap.Any("step_status", status.OperationCondition))
			return
		}
		// TODO: 简化更新,允许强制更新?
		for i := 0; i < updateOperationStatusRetry; i++ {
			o, err := s.opOperator.GetOperation(context.TODO(), status.OperationIdentity)
			if err != nil {
				logger.Error("get operation failed", zap.String("op", status.OperationIdentity),
					zap.String("step", status.OperationCondition.StepID), zap.Any("step_status", status.OperationCondition), zap.Error(err))
				continue
			}

			stepLen := len(o.Status.Conditions)
			if stepLen > 0 && status.OperationCondition.StepID == o.Status.Conditions[stepLen-1].StepID {
				o.Status.Conditions[stepLen-1].Status = append(o.Status.Conditions[stepLen-1].Status, status.OperationCondition.Status...)
			} else {
				o.Status.Conditions = append(o.Status.Conditions, status.OperationCondition)
			}

			if _, err := s.opOperator.UpdateOperation(context.TODO(), o); err != nil {
				logger.Error("update operation step condition failed", zap.String("op", status.OperationIdentity),
					zap.String("step", status.OperationCondition.StepID), zap.Any("step_status", status.OperationCondition), zap.Error(err))
				continue
			}
			break
		}
	}
}

func (s *Service) updateOperationStatus(op string, status v1.OperationStatusType, dryRun bool) {
	if dryRun {
		logger.Debug("dry run update operation status", zap.String("status", string(status)))
		return
	}
	// TODO: 简化更新,允许强制更新?
	for i := 0; i < updateOperationStatusRetry; i++ {
		logger.Debug("update operation status", zap.String("status", string(status)))
		o, err := s.opOperator.GetOperation(context.TODO(), op)
		if err != nil {
			logger.Error("update operation status type failed", zap.String("op", op), zap.String("status", string(status)), zap.Error(err))
			continue
		}
		o.Status.Status = status
		if o, err = s.opOperator.UpdateOperation(context.TODO(), o); err != nil {
			logger.Error("update operation status type failed", zap.String("op", op), zap.String("status", string(status)), zap.Error(err))
			continue
		}
		go s.SyncClusterCondition(o)
		return
	}
}

func (s *Service) SyncClusterCondition(op *v1.Operation) {
	defer service.HandlerCrash()
	for i := 0; i < updateOperationStatusRetry; i++ {
		clu, err := s.clusterOperator.GetClusterEx(context.TODO(), op.Labels[common.LabelClusterName], "0")
		if err != nil {
			logger.Error("get cluster failed", zap.String("name", clu.Name), zap.String("operation", op.Name), zap.Error(err))
			continue
		}
		if err = s.syncClusterCondition(op, clu); err != nil {
			continue
		}
		return
	}
}

func (s *Service) syncClusterCondition(op *v1.Operation, clu *v1.Cluster) error {
	switch v := op.Labels[common.LabelOperationAction]; v {
	case v1.OperationCreateCluster:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
		} else {
			clu.Status.Phase = v1.ClusterInstallFailed
		}
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationAddNodes:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
		} else {
			clu.Status.Phase = v1.ClusterUpdateFailed
		}
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationDeleteCluster:
		if op.Status.Status == v1.OperationStatusSuccessful {
			return s.clusterOperator.DeleteCluster(context.TODO(), clu.Name)
		}
		clu.Status.Phase = v1.ClusterTerminateFailed
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationRemoveNodes:
		if op.Status.Status == v1.OperationStatusSuccessful {
			var nodes []string
			for _, step := range op.Steps {
				// TODO: 后续考虑其他方式判断需要删除的节点
				if step.Name == "kubeadmReset" {
					var ids []string
					for _, v := range step.Nodes {
						ids = append(ids, v.ID)
					}
					nodes = ids
					break
				}
			}
			removed := make([]v1.WorkerNode, len(nodes))
			for i, n := range nodes {
				removed[i] = v1.WorkerNode{
					ID:     n,
					Labels: nil,
					Taints: nil,
				}
			}
			removed = clu.Workers.Intersect(removed...)
			if len(removed) > 0 {
				clu.Workers = clu.Workers.Complement(removed...)
			}
			clu.Status.Phase = v1.ClusterRunning
			if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
				return err
			}
			for _, n := range nodes {
				if err := s.updateNodeRoleLabel(clu.Name, n, common.NodeRoleWorker, true); err != nil {
					return err
				}
				logger.Debug("update node role label when remove nodes successful", zap.String("cluster", clu.Name),
					zap.String("op", op.Name), zap.String("node_id", n))
			}
			return nil
		}
		clu.Status.Phase = v1.ClusterUpdateFailed
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationUpgradeCluster:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
			clu.KubernetesVersion = op.Labels[common.LabelUpgradeVersion]
		} else {
			clu.Status.Phase = v1.ClusterUpgradeFailed
		}
		_, err := s.clusterOperator.UpdateCluster(context.TODO(), clu)
		return err
	case v1.OperationBackupCluster:
		clu.Status.Phase = v1.ClusterRunning
		_, err := s.clusterOperator.UpdateCluster(context.TODO(), clu)
		return err
	case v1.OperationRecoverCluster:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
		} else {
			clu.Status.Phase = v1.ClusterRestoreFailed
		}
		_, err := s.clusterOperator.UpdateCluster(context.TODO(), clu)
		return err
	case v1.OperationInstallComponents, v1.OperationUninstallComponents:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
		} else {
			clu.Status.Phase = v1.ClusterUpdateFailed
		}
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationUpdateCertification:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
			var err error
			clu.Status.Certifications, err = (&controller.ClusterStatusMon{
				CmdDelivery: s,
			}).GetCertificationFromKC(clu)
			if err != nil {
				logger.Errorf("update expiration error after update certs: %s", err.Error())
			}
		} else {
			clu.Status.Phase = v1.ClusterUpdateFailed
		}
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	case v1.OperationUpdateAPIServerCertification:
		if op.Status.Status == v1.OperationStatusSuccessful {
			clu.Status.Phase = v1.ClusterRunning
		} else {
			clu.Status.Phase = v1.ClusterUpdateFailed
		}
		if _, err := s.clusterOperator.UpdateCluster(context.TODO(), clu); err != nil {
			return err
		}
		return nil
	default:
		logger.Error("unsupported operation action", zap.String("operation", op.Name),
			zap.String("cluster", clu.Name), zap.String("action", v))
		return nil
	}
}

func (s *Service) updateNodeRoleLabel(clusterName, nodeName string, role common.NodeRole, del bool) error {
	node, err := s.clusterOperator.GetNodeEx(context.TODO(), nodeName, "0")
	if err != nil {
		return err
	}
	if del {
		// check node role label exist.
		// if existed, delete label and update node.
		// if not return direct.
		if _, ok := node.Labels[common.LabelNodeRole]; !ok {
			return nil
		}
		if v, ok := node.Labels[common.LabelClusterName]; !ok || v != clusterName {
			return nil
		}
		delete(node.Labels, common.LabelNodeRole)
		delete(node.Labels, common.LabelClusterName)
	} else {
		// check node role label exist.
		// if existed, return direct
		// if not add label and update node.
		if _, ok := node.Labels[common.LabelNodeRole]; ok {
			return nil
		}
		node.Labels[common.LabelNodeRole] = string(role)
		node.Labels[common.LabelClusterName] = clusterName
	}

	if _, err = s.clusterOperator.UpdateNode(context.TODO(), node); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeliverTaskOperation(ctx context.Context, operation *v1.Operation, opts *service.Options) error {
	if opts == nil {
		opts = &service.Options{DryRun: false}
	}
	timeoutSecs := operation.Labels[common.LabelTimeoutSeconds]
	secs, _ := strconv.Atoi(timeoutSecs)
	ctx, cancelFn := context.WithTimeout(ctx, time.Duration(secs)*time.Second)
	defer cancelFn()
	// new empty context, pass retry value
	stepCtx, stepCtxCancel := context.WithCancel(component.WithRetry(context.TODO(), component.GetRetry(ctx)))
	defer stepCtxCancel()
	doneChan := make(chan struct{}, 1)
	defer close(doneChan)
	errChan := make(chan error, 1)
	defer close(errChan)
	operation.Status.Conditions = make([]v1.OperationCondition, len(operation.Steps))
	var termination bool
	go func() {
		for {
			select {
			case <-ctx.Done():
				// operation timeout
				stepCtxCancel()
				go s.updateOperationStatus(operation.Name, v1.OperationStatusFailed, opts.DryRun)
				return
			case <-*s.terminationChan:
				if operation.Status.Status == v1.OperationStatusRunning {
					// get latest edition operation
					op, err := s.opOperator.GetOperationEx(context.TODO(), operation.Name, "0")
					if err != nil {
						logger.Error("get latest edition operation failed", zap.String("op", op.Name), zap.Error(err))
						continue
					}
					logger.Debugf("get latest operation: %v, label intent: %v", operation.Name, op.Labels[common.LabelOperationIntent])
					if op.Labels[common.LabelOperationIntent] == common.OperationIntent {
						// termination operation
						stepCtxCancel()
						// termination step flag
						termination = true

						go s.updateOperationStatus(operation.Name, v1.OperationStatusTermination, opts.DryRun)
						return
					}
				}
			case <-doneChan:
				// all step done, set operation status successful
				go s.updateOperationStatus(operation.Name, v1.OperationStatusSuccessful, opts.DryRun)
				return
			case <-errChan:
				// step run error and step ignoreError flag is false
				go s.updateOperationStatus(operation.Name, v1.OperationStatusFailed, opts.DryRun)
				return
			}
		}
	}()
	var err error
	for i, step := range operation.Steps {
		if termination {
			logger.Debug("termination delivery task step", zap.String("operation", operation.Name), zap.String("step", step.Name))
			break
		}
		// TODO: add retry steps
		// TODO: refactor
		// Notice: 目前只针对 CUSTOM 命令有用，下一步骤依赖上一步骤的输出，比如 K8S 安装时初始化一个 K8S 控制节点后得到 kubeadm join 命令，需要传给其他节点进行执行
		// len(steps) > 0
		if i-1 > 0 {
			// Steps will not be run when nodes field is empty,
			// so there is no running status.
			// May be out of list range here.
			if len(operation.Status.Conditions[i-1].Status) < 1 {
				if !opts.ForceSkipError {
					return errors.New("unexpected error, steps node field must be valid")
				}
				err = s.deliveryTaskStep(stepCtx, operation.Name, &operation.Steps[i],
					nil, &operation.Status.Conditions[i], opts.DryRun)
			} else {
				logger.Info("last response", zap.ByteString("response", operation.Status.Conditions[i-1].Status[0].Response))
				err = s.deliveryTaskStep(stepCtx, operation.Name, &operation.Steps[i],
					operation.Status.Conditions[i-1].Status[0].Response, &operation.Status.Conditions[i], opts.DryRun)
			}
		} else {
			err = s.deliveryTaskStep(stepCtx, operation.Name, &operation.Steps[i],
				component.GetExtraData(ctx), &operation.Status.Conditions[i], opts.DryRun)
		}
		logger.Debug("after delivery task step", zap.Error(err))
		if err != nil {
			logger.Error("delivery task step error", zap.Error(err), zap.String("step", step.Name))
			if step.ErrIgnore || opts.ForceSkipError {
				logger.Debug("delivery task step, ignore the error", zap.Error(err), zap.String("step", step.Name))
				// reset error
				err = nil
				continue
			}
			break
		}
	}
	if err != nil {
		errChan <- err
	} else {
		doneChan <- struct{}{}
	}
	return nil
}

func (s *Service) DeliverLogRequest(ctx context.Context, operation *service.LogOperation) (opResp oplog.LogContentResponse, err error) {
	pb, err := initPayload(operation.OperationIdentity, operation.Op, nil, nil, nil, false, component.GetRetry(ctx))
	if err != nil {
		return
	}
	msg := &natsio.Msg{
		Subject: fmt.Sprintf(service.MsgSubjectFormat, operation.To, s.subjectSuffix),
		Data:    pb,
	}
	// may be blocked and can be cancelled ny context
	data, err := s.client.RequestWithContext(ctx, msg)
	if err != nil {
		return
	}
	resp := &service.CommonReply{}
	if err = json.Unmarshal(data, resp); err != nil {
		logger.Error("unmarshal agent reply error", zap.Error(err))
		return
	}
	if err = json.Unmarshal(resp.Data, &opResp); err != nil {
		logger.Error("unmarshal operation log response error", zap.Error(err))
		return
	}
	return
}

func (s *Service) DeliverStep(ctx context.Context, step *v1.Step, opts *service.Options) error {
	if opts == nil {
		opts = &service.Options{DryRun: false}
	}
	errChan := make(chan error, len(step.Nodes))
	defer close(errChan)
	wg := &sync.WaitGroup{}
	status := new(v1.StepStatus)
	payloadBytes, err := initPayload("", service.OperationRunStep, step, nil, nil, opts.DryRun, component.GetRetry(ctx))

	for _, node := range step.Nodes {
		wg.Add(1)
		go s.deliveryStepToNode(wg, node.ID, payloadBytes, step.Timeout.Duration+2*time.Second, status, errChan)
	}

	wg.Wait()

	logger.Debug("after delivery task step", zap.Error(err))
	if err != nil {
		logger.Error("delivery task step error", zap.Error(err), zap.String("step", step.Name))
		return err
	}

	return nil
}

func (s *Service) DeliverCmd(ctx context.Context, toNode string, cmds []string, timeout time.Duration) ([]byte, error) {
	payload, err := initPayload("", service.OperationRunCmd, &v1.Step{Timeout: metav1.Duration{Duration: timeout}}, nil, cmds, false, component.GetRetry(ctx))
	if err != nil {
		return nil, err
	}
	msg := &natsio.Msg{
		Subject: fmt.Sprintf(service.MsgSubjectFormat, toNode, s.subjectSuffix),
		Data:    payload,
	}
	// may be blocked and can be cancelled ny context
	data, err := s.client.RequestWithContext(ctx, msg)
	if err != nil {
		return nil, err
	}
	resp := &service.CommonReply{}
	if err := json.Unmarshal(data, resp); err != nil {
		logger.Error("unmarshal agent reply error", zap.Error(err))
		return nil, err
	}
	return resp.Data, nil
}

func (s *Service) deliveryTaskStep(ctx context.Context, opName string, step *v1.Step, lastStepReply []byte, cond *v1.OperationCondition, dryRun bool) error {
	payloadBytes, err := initPayload(opName, service.OperationRunTask, step, lastStepReply, nil, dryRun, component.GetRetry(ctx))
	if err != nil {
		return err
	}

	doneChan := make(chan struct{}, 1)
	defer close(doneChan)
	status := make([]v1.StepStatus, len(step.Nodes))
	cond.Status = status
	cond.StepID = step.ID
	// opCond := v1.OperationCondition{
	//	StepID: step.ID,
	//	Status: status,
	// }
	go func(op string, cond *v1.OperationCondition) {
		for {
			select {
			case <-ctx.Done():
				// operation timeout
				s.sendStepStatusToChannel(stepStatus{
					OperationIdentity:  op,
					OperationCondition: *cond,
					DryRun:             dryRun,
				})
				return
			case <-doneChan:
				// step done
				logger.Debug("in step done cond", zap.Any("condition", *cond))
				s.sendStepStatusToChannel(stepStatus{
					OperationIdentity:  op,
					OperationCondition: *cond,
					DryRun:             dryRun,
				})
				return
			}
		}
	}(opName, cond)

	wg := sync.WaitGroup{}
	// NOTE: per node can send one error only.
	errChan := make(chan error, len(step.Nodes))
	defer close(errChan)

	for i, node := range step.Nodes {
		wg.Add(1)
		// notice: make sure step timeout less than operation timeout
		// TODO: add step retry
		go s.deliveryStepToNode(&wg, node.ID, payloadBytes, step.Timeout.Duration+2*time.Second, &status[i], errChan)
	}

	wg.Wait()

	if len(errChan) > 0 {
		logger.Debug("err chan has value...")
		return <-errChan
	}
	doneChan <- struct{}{}
	logger.Debug("deliveryTaskStep method end...")
	return nil
}

func (s *Service) deliveryStepToNode(wg *sync.WaitGroup, node string, payload []byte, timeout time.Duration, stepStatus *v1.StepStatus, errChan chan error) {
	defer wg.Done()

	now := time.Now()
	stepStatus.StartAt = metav1.NewTime(now)
	stepStatus.Node = node

	msg := &natsio.Msg{
		Subject: fmt.Sprintf(service.MsgSubjectFormat, node, s.subjectSuffix),
		From:    "",
		To:      "",
		Step:    "",
		Timeout: timeout,
		Data:    payload,
	}
	data, err := s.client.Request(msg, func(msg *natsio.Msg) error {
		setStepStatus(stepStatus, v1.StepStatusFailed, "run step timeout", "server wait for agent reply timeout", nil)
		return nil
	})
	if err != nil {
		setStepStatus(stepStatus, v1.StepStatusFailed, err.Error(), "internal server error for send request to agent", nil)
		errChan <- err
		return
	}
	resp := &service.CommonReply{}
	if err = json.Unmarshal(data, resp); err != nil {
		logger.Error("unmarshal agent reply error", zap.Error(err))
		setStepStatus(stepStatus, v1.StepStatusFailed, "unmarshal agent reply error", err.Error(), nil)
		errChan <- err
		return
	}
	if resp.Error != nil {
		setStepStatus(stepStatus, v1.StepStatusFailed, resp.Error.Message, resp.Error.Error(), nil)
		errChan <- resp.Error
		return
	}
	setStepStatus(stepStatus, v1.StepStatusSuccessful, "run step successfully", "run step successfully", resp.Data)
}

func setStepStatus(status *v1.StepStatus, statusType v1.StepStatusType, message, reason string, response []byte) {
	status.Status = statusType
	status.Message = message
	status.Reason = reason
	status.Response = response
	status.EndAt = metav1.NewTime(time.Now())
}

func (s *Service) sendStepStatusToChannel(status stepStatus) {
	s.stepStatusChan <- status
}
