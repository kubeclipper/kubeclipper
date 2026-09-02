/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package operationv2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

const (
	syncKey        = "tasks"
	workerLockMode = 0600
	// serverCallTimeout bounds each one-shot server round trip. The shared
	// rest config carries no client-wide timeout because http.Client.Timeout
	// would sever the informer's long-lived watch on every interval.
	serverCallTimeout = 30 * time.Second
)

type TaskClient interface {
	Get(context.Context, string, metav1.GetOptions) (*operations.OperationTask, error)
	List(context.Context, *metav1.ListOptions) (*operations.OperationTaskList, error)
	Watch(context.Context, *metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(context.Context, *operations.OperationTask) (*operations.OperationTask, error)
}

type TaskLog interface {
	CreateOperationDir(string) error
	CreateStepLogFile(string, string) (*os.File, error)
}

type WorkerOptions struct {
	AgentID  string
	NodeUID  types.UID
	Client   TaskClient
	Registry *Registry
	OpLog    TaskLog
	LockFile string
}

type Worker struct {
	agentID  string
	nodeUID  types.UID
	client   TaskClient
	registry *Registry
	oplog    TaskLog
	lockPath string
	runCtx   context.Context
	cancel   context.CancelFunc

	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
	lockFile *os.File
	close    sync.Once
}

func NewWorker(opts *WorkerOptions) (*Worker, error) {
	if opts == nil {
		return nil, fmt.Errorf("worker options are required")
	}
	if opts.AgentID == "" || opts.NodeUID == "" {
		return nil, fmt.Errorf("agent ID and Node UID are required")
	}
	if opts.Client == nil || opts.Registry == nil || opts.OpLog == nil {
		return nil, fmt.Errorf("task client, executor registry, and operation log are required")
	}
	if opts.LockFile == "" {
		opts.LockFile = "/run/kubeclipper-agent-operation-v2.lock"
	}
	runCtx, cancel := context.WithCancel(context.Background())
	w := &Worker{
		agentID:  opts.AgentID,
		nodeUID:  opts.NodeUID,
		client:   opts.Client,
		registry: opts.Registry,
		oplog:    opts.OpLog,
		lockPath: opts.LockFile,
		runCtx:   runCtx,
		cancel:   cancel,
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "operation-v2-agent"),
	}
	selector := fields.OneTermEqualSelector("spec.nodeRef.name", opts.AgentID).String()
	w.informer = cache.NewSharedIndexInformer(&cache.ListWatch{
		ListFunc: func(listOptions metav1.ListOptions) (runtime.Object, error) {
			listOptions.FieldSelector = selector
			return w.listTasks(&listOptions)
		},
		WatchFunc: func(listOptions metav1.ListOptions) (watch.Interface, error) {
			listOptions.FieldSelector = selector
			return w.watchTasks(&listOptions)
		},
	}, &operations.OperationTask{}, 0, cache.Indexers{})
	if _, err := w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { w.queue.Add(syncKey) },
		UpdateFunc: func(any, any) { w.queue.Add(syncKey) },
		DeleteFunc: func(any) { w.queue.Add(syncKey) },
	}); err != nil {
		return nil, fmt.Errorf("add task event handler: %w", err)
	}
	return w, nil
}

func (w *Worker) PrepareRun(<-chan struct{}) error {
	lockFile, err := os.OpenFile(w.lockPath, os.O_CREATE|os.O_RDWR, workerLockMode)
	if err != nil {
		return fmt.Errorf("open agent singleton lock: %w", err)
	}
	if err := unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lockFile.Close()
		return fmt.Errorf("another Operation v2 worker is active: %w", err)
	}
	w.lockFile = lockFile
	return nil
}

func (w *Worker) Run(stopCh <-chan struct{}) error {
	go w.informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, w.informer.HasSynced) {
		return fmt.Errorf("OperationTask informer stopped before initial List completed")
	}
	w.queue.Add(syncKey)
	go func() {
		for w.processNext(stopCh) {
		}
	}()
	return nil
}

func (w *Worker) Close() {
	w.close.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		w.queue.ShutDown()
		if w.lockFile != nil {
			if err := unix.Flock(int(w.lockFile.Fd()), unix.LOCK_UN); err != nil {
				logger.Errorf("unlock Operation v2 worker: %v", err)
			}
			_ = w.lockFile.Close()
		}
	})
}

func (w *Worker) processNext(stopCh <-chan struct{}) bool {
	item, shutdown := w.queue.Get()
	if shutdown {
		return false
	}
	defer w.queue.Done(item)
	select {
	case <-stopCh:
		return false
	default:
	}
	if err := w.sync(w.runCtx); err != nil {
		logger.Errorf("Operation v2 agent sync failed: %v", err)
		w.queue.AddRateLimited(syncKey)
		return true
	}
	w.queue.Forget(item)
	return true
}

func (w *Worker) sync(ctx context.Context) error {
	selected, err := selectTask(w.eligibleTasks())
	if err != nil || selected == nil {
		return err
	}
	live, err := w.getLiveTask(ctx, selected)
	if err != nil || live == nil {
		return err
	}
	live, err = w.startPendingTask(ctx, live)
	if err != nil {
		return err
	}
	if live.Status.Phase != operations.TaskRunning {
		return fmt.Errorf("task %q has unsupported active phase %q", live.Name, live.Status.Phase)
	}
	return w.execute(ctx, live)
}

// listTasks serves the reflector's initial List. It is a one-shot round trip
// and carries its own deadline; a client-wide timeout must not be used because
// it would also sever the long-lived watch below.
func (w *Worker) listTasks(listOptions *metav1.ListOptions) (runtime.Object, error) {
	ctx, cancel := context.WithTimeout(w.runCtx, serverCallTimeout)
	defer cancel()
	return w.client.List(ctx, listOptions)
}

// watchTasks serves the reflector's long-lived Watch. It inherits only the
// worker's run context, so the stream lives until the worker stops and the
// reflector owns reconnects; bounding it here would kill the watch on every
// timeout and turn event delivery into polling.
func (w *Worker) watchTasks(listOptions *metav1.ListOptions) (watch.Interface, error) {
	return w.client.Watch(w.runCtx, listOptions)
}

func (w *Worker) eligibleTasks() []*operations.OperationTask {
	objects := w.informer.GetStore().List()
	tasks := make([]*operations.OperationTask, 0, len(objects))
	for _, object := range objects {
		task, ok := object.(*operations.OperationTask)
		if !ok || task.Spec.NodeRef.Name != w.agentID || task.Spec.NodeRef.UID != w.nodeUID || task.Status.Phase.IsTerminal() {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (w *Worker) getLiveTask(ctx context.Context, selected *operations.OperationTask) (*operations.OperationTask, error) {
	getCtx, cancel := context.WithTimeout(ctx, serverCallTimeout)
	defer cancel()
	live, err := w.client.Get(getCtx, selected.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if live.UID != selected.UID || live.Spec.NodeRef.Name != w.agentID || live.Spec.NodeRef.UID != w.nodeUID {
		return nil, fmt.Errorf("task %q identity changed during dispatch", selected.Name)
	}
	if live.Status.Phase.IsTerminal() {
		w.queue.Add(syncKey)
		return nil, nil
	}
	return live, nil
}

func (w *Worker) startPendingTask(ctx context.Context, live *operations.OperationTask) (*operations.OperationTask, error) {
	if live.Status.Phase == operations.TaskPending {
		runningTask := live.DeepCopy()
		runningTask.Status = operations.OperationTaskStatus{Phase: operations.TaskRunning}
		putCtx, cancel := context.WithTimeout(ctx, serverCallTimeout)
		defer cancel()
		return w.client.UpdateStatus(putCtx, runningTask)
	}
	return live, nil
}

func selectTask(tasks []*operations.OperationTask) (*operations.OperationTask, error) {
	running := make([]*operations.OperationTask, 0, 1)
	pending := make([]*operations.OperationTask, 0, len(tasks))
	for _, task := range tasks {
		switch task.Status.Phase {
		case operations.TaskRunning:
			running = append(running, task)
		case "", operations.TaskPending:
			pending = append(pending, task)
		}
	}
	if len(running) > 1 {
		return nil, fmt.Errorf("found %d Running tasks for one agent; refusing concurrent execution", len(running))
	}
	if len(running) == 1 {
		return running[0], nil
	}
	if len(pending) == 0 {
		return nil, nil
	}
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].CreationTimestamp.Equal(&pending[j].CreationTimestamp) {
			// Second granularity is not enough for a deterministic order:
			// the monotonic resourceVersion is the intended tiebreaker.
			ri, errI := strconv.ParseUint(pending[i].ResourceVersion, 10, 64)
			rj, errJ := strconv.ParseUint(pending[j].ResourceVersion, 10, 64)
			if errI == nil && errJ == nil && ri != rj {
				return ri < rj
			}
			return string(pending[i].UID) < string(pending[j].UID)
		}
		return pending[i].CreationTimestamp.Before(&pending[j].CreationTimestamp)
	})
	return pending[0], nil
}

func (w *Worker) execute(parent context.Context, task *operations.OperationTask) error {
	executor, ok := w.registry.Get(task.Spec.Executor)
	if !ok {
		return w.finish(parent, task, operations.TaskFailed, operations.TaskResult{
			Reason:  operations.TaskReasonExecutionFailed,
			Message: fmt.Sprintf("executor %q is not registered", task.Spec.Executor),
		})
	}
	if err := w.oplog.CreateOperationDir(string(task.UID)); err != nil {
		logger.Errorf("create Task log directory for %s: %v", task.Name, err)
	}
	logWriter, logErr := w.oplog.CreateStepLogFile(string(task.UID), "task")
	if logErr != nil {
		logger.Errorf("open Task log for %s: %v", task.Name, logErr)
		logWriter = nil
	}
	if logWriter != nil {
		defer logWriter.Close()
		_, _ = fmt.Fprintf(logWriter, "\n--- reconcile %s ---\n", time.Now().UTC().Format(time.RFC3339))
	}
	writer := io.Discard
	if logWriter != nil {
		writer = logWriter
	}

	if task.Spec.Deadline.Time.IsZero() {
		// A zero deadline would expire the context immediately and kill the
		// command before it ran; fail the task with an explicit reason.
		return w.finish(parent, task, operations.TaskFailed, operations.TaskResult{
			Reason:  operations.TaskReasonExecutionFailed,
			Message: taskResultMessage("task has no deadline", logErr),
		})
	}
	ctx, cancel := context.WithDeadline(parent, task.Spec.Deadline.Time)
	defer cancel()
	result, reconcileErr := executor.Reconcile(ctx, task.DeepCopy(), writer)
	if parent.Err() != nil {
		return parent.Err()
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(reconcileErr, context.DeadlineExceeded) {
		return w.finish(context.Background(), task, operations.TaskTimedOut, operations.TaskResult{
			Reason:  operations.TaskReasonDeadlineExceeded,
			Message: taskResultMessage("task deadline exceeded", logErr),
		})
	}
	if reconcileErr != nil {
		return w.finish(parent, task, operations.TaskFailed, operations.TaskResult{
			Reason:  operations.TaskReasonExecutionFailed,
			Message: taskResultMessage(reconcileErr.Error(), logErr),
		})
	}
	result.Reason = ""
	// The server-side log read would 404 without an explanation; surface the
	// reason through the task result regardless of executor outcome.
	result.Message = taskResultMessage(result.Message, logErr)
	return w.finish(parent, task, operations.TaskSucceeded, result)
}

func (w *Worker) finish(
	ctx context.Context,
	task *operations.OperationTask,
	phase operations.TaskPhase,
	result operations.TaskResult,
) error {
	updatedTask := task.DeepCopy()
	updatedTask.Status.Phase = phase
	updatedTask.Status.Result = &result
	putCtx, cancel := context.WithTimeout(ctx, serverCallTimeout)
	defer cancel()
	_, err := w.client.UpdateStatus(putCtx, updatedTask)
	if err != nil {
		getCtx, getCancel := context.WithTimeout(ctx, serverCallTimeout)
		latest, getErr := w.client.Get(getCtx, task.Name, metav1.GetOptions{})
		getCancel()
		if getErr == nil && latest.UID == task.UID && latest.Status.Phase.IsTerminal() {
			w.queue.Add(syncKey)
			return nil
		}
		return err
	}
	w.queue.Add(syncKey)
	return nil
}

func boundedMessage(message string) string {
	if len(message) <= operations.MaxMessageSize {
		return message
	}
	truncated := message[:operations.MaxMessageSize]
	// Do not split a rune: trim the invalid trailing bytes so the message
	// stays valid UTF-8 after truncation.
	for truncated != "" {
		r, size := utf8.DecodeLastRuneInString(truncated)
		if r != utf8.RuneError || size != 1 {
			break
		}
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func taskResultMessage(message string, logErr error) string {
	if logErr != nil {
		message += "; agent task log unavailable: " + logErr.Error()
	}
	return boundedMessage(message)
}

var _ interface {
	PrepareRun(<-chan struct{}) error
	Run(<-chan struct{}) error
	Close()
} = (*Worker)(nil)
