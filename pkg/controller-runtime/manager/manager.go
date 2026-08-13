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

package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeclipper/kubeclipper/pkg/controller-runtime/client"
	"github.com/kubeclipper/kubeclipper/pkg/server/registry"

	"github.com/kubeclipper/kubeclipper/pkg/client/clientset"
	"github.com/kubeclipper/kubeclipper/pkg/client/informers"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/uuid"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/kubeclipper/kubeclipper/pkg/leaderelect"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/models/lease"
	"github.com/kubeclipper/kubeclipper/pkg/service"
)

var _ service.Interface = (*ControllerManager)(nil)
var _ Manager = (*ControllerManager)(nil)

type Manager interface {
	// ClusterClientWriter() cluster.OperatorWriter
	// OperationClient() operation.Writer
	AddRunnable(run Runnable)
	AddWorkerLoop(fn LoopFunc, period time.Duration)
	GetLogger() logger.Logging
	GetClusterClientSet(cluster string) (client.Client, bool)
	AddClusterClientSet(cluster string, client client.Client)
	RemoveClusterClientSet(cluster string)
}

// Runnable allows a component to be started.
// It's very important that Start blocks until
// it's done running.
type Runnable interface {
	// Start starts running the component.  The component will stop running
	// when the context is closed. Start blocks until the context is closed or
	// an error occurs.
	Start(context.Context) error
}

// RunnableFunc implements Runnable using a function.
// It's very important that the given function block
// until it's done running.
type RunnableFunc func(context.Context) error

// Start implements Runnable.
func (r RunnableFunc) Start(ctx context.Context) error {
	return r(ctx)
}

const (
	resourceLockNS        = "default"
	resourceLockName      = "nodelifecycle-controller"
	readinessLeaseName    = "kc-controller-manager-ready"
	readinessLeasePeriod  = 5 * time.Second
	readinessLeaseTimeout = 15 * time.Second
)

type LoopFunc func()

type SetupFunc func(mgr Manager, informerFactory informers.SharedInformerFactory, storageFactory registry.SharedStorageFactory) error

type ControllerManager struct {
	storageFactory registry.SharedStorageFactory

	cs clientset.Interface

	setupFunc SetupFunc

	lock           resourcelock.Interface
	leaderElector  *leaderelection.LeaderElector
	leaderStopChan chan struct{}
	identity       string
	leaseOperator  lease.Operator

	defaultWorkerLoopPeriod time.Duration
	workerLoops             []workerLoop
	workerLoopLock          sync.Mutex

	runnables    []Runnable
	runnableLock sync.Mutex

	log logger.Logging

	// per cluster map storing cluster clientset
	clusterClientMap *clusterClientMap
}

type workerLoop struct {
	fn LoopFunc
	time.Duration
}

type clusterClientMap struct {
	sync.RWMutex
	clients map[string]client.Client
}

func newClusterClientMap() *clusterClientMap {
	return &clusterClientMap{
		clients: make(map[string]client.Client),
	}
}

func (c *clusterClientMap) Get(name string) (client.Client, bool) {
	c.RLock()
	defer c.RUnlock()
	cli, exist := c.clients[name]
	return cli, exist
}

func (c *clusterClientMap) AddClientSet(name string, client client.Client) {
	c.Lock()
	c.clients[name] = client
	c.Unlock()
}

func (c *clusterClientMap) removeClientSet(name string) {
	c.Lock()
	delete(c.clients, name)
	c.Unlock()
}

func NewControllerManager(rc *rest.Config, storageFactory registry.SharedStorageFactory, setupFunc SetupFunc) (*ControllerManager, error) {
	identity := uuid.New().String()
	leaseOperator := lease.NewLeaseOperator(storageFactory.Leases())
	s := &ControllerManager{
		leaderStopChan:          make(chan struct{}, 1),
		defaultWorkerLoopPeriod: time.Second,
		clusterClientMap:        newClusterClientMap(),
		storageFactory:          storageFactory,
		setupFunc:               setupFunc,
		identity:                identity,
		leaseOperator:           leaseOperator,
	}
	s.lock = leaderelect.NewLock(resourceLockNS, resourceLockName, leaseOperator, resourcelock.ResourceLockConfig{
		Identity:      identity,
		EventRecorder: nil,
	})
	cs, err := clientset.NewForConfig(rc)
	if err != nil {
		return nil, err
	}
	s.cs = cs
	s.log = logger.WithName("ctrl")
	return s, nil
}

func (s *ControllerManager) AddWorkerLoop(fn LoopFunc, period time.Duration) {
	s.workerLoopLock.Lock()
	defer s.workerLoopLock.Unlock()
	if period == 0 {
		period = s.defaultWorkerLoopPeriod
	}
	s.workerLoops = append(s.workerLoops, workerLoop{
		fn:       fn,
		Duration: period,
	})
}

func (s *ControllerManager) AddRunnable(run Runnable) {
	s.runnableLock.Lock()
	defer s.runnableLock.Unlock()
	s.runnables = append(s.runnables, run)
}

func (s *ControllerManager) GetLogger() logger.Logging {
	return s.log
}

func (s *ControllerManager) GetClusterClientSet(cluster string) (client.Client, bool) {
	return s.clusterClientMap.Get(cluster)
}

func (s *ControllerManager) AddClusterClientSet(cluster string, client client.Client) {
	s.clusterClientMap.AddClientSet(cluster, client)
}

func (s *ControllerManager) RemoveClusterClientSet(cluster string) {
	s.clusterClientMap.removeClientSet(cluster)
}

func (s *ControllerManager) PrepareRun(stopCh <-chan struct{}) error {
	if s.lock == nil {
		return nil
	}
	var err error
	s.leaderElector, err = leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          s.lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: s.runManager,
			OnStoppedLeading: func() {
				s.log.Warn("leadership lost")
				s.leaderStopChan <- struct{}{}
			},
		},
		WatchDog:        nil,
		Name:            "kc-controller-manager",
		ReleaseOnCancel: true,
	})

	return err
}

func (s *ControllerManager) Run(stopCh <-chan struct{}) error {
	go s.runControllerManager(stopCh)
	return nil
}

func (s *ControllerManager) Close() {}

func (s *ControllerManager) runControllerManager(stopCh <-chan struct{}) {
	if s.lock == nil {
		return
	}
	reloadCh := make(chan struct{}, 1)
	defer close(reloadCh)
	go s.startLeaderElect(reloadCh)
	for {
		select {
		case <-stopCh:
			return
		case <-s.leaderStopChan:
			reloadCh <- struct{}{}
			go s.startLeaderElect(reloadCh)
		}
	}
}

func (s *ControllerManager) startLeaderElect(reload <-chan struct{}) {
	s.log.Debug("in start leader elect...")
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	go s.leaderElector.Run(ctx)
	<-reload
}

func (s *ControllerManager) runManager(ctx context.Context) {
	s.log.Debug("start run manager...")
	inFactory := informers.NewSharedInformerFactory(s.cs, 10*time.Hour)
	s.workerLoopLock.Lock()
	s.workerLoops = nil
	s.workerLoopLock.Unlock()
	s.runnableLock.Lock()
	s.runnables = nil
	s.runnableLock.Unlock()
	// TODO: error should be caught
	if err := s.setupFunc(s, inFactory, s.storageFactory); err != nil {
		s.log.Error("setup controller manager failed", zap.Error(err))
	}
	inFactory.Start(ctx.Done())
	s.runWorkerLoops(ctx)
	s.runRunnables(ctx)
	go s.runReadinessLease(ctx)
	<-ctx.Done()
	s.log.Debug("stop run manager...")
}

func (s *ControllerManager) Health(ctx context.Context) error {
	leader, err := s.leaseOperator.GetLeaseWithNamespaceEx(ctx, resourceLockName, resourceLockNS, "0")
	if err != nil {
		return fmt.Errorf("get leader lease: %w", err)
	}
	ready, err := s.leaseOperator.GetLeaseWithNamespaceEx(ctx, readinessLeaseName, resourceLockNS, "0")
	if err != nil {
		return fmt.Errorf("get readiness lease: %w", err)
	}
	now := time.Now()
	if err := validateLease(leader, now); err != nil {
		return fmt.Errorf("leader lease: %w", err)
	}
	if err := validateLease(ready, now); err != nil {
		return fmt.Errorf("readiness lease: %w", err)
	}
	if leader.Spec.HolderIdentity == nil || ready.Spec.HolderIdentity == nil || *leader.Spec.HolderIdentity != *ready.Spec.HolderIdentity {
		return fmt.Errorf("leader and readiness lease holders do not match")
	}
	return nil
}

func validateLease(item *coordinationv1.Lease, now time.Time) error {
	if item.Spec.RenewTime == nil || item.Spec.LeaseDurationSeconds == nil {
		return fmt.Errorf("lease timing is incomplete")
	}
	if !now.Before(item.Spec.RenewTime.Add(time.Duration(*item.Spec.LeaseDurationSeconds) * time.Second)) {
		return fmt.Errorf("lease expired")
	}
	return nil
}

func (s *ControllerManager) runReadinessLease(ctx context.Context) {
	ticker := time.NewTicker(readinessLeasePeriod)
	defer ticker.Stop()
	for {
		if err := s.renewReadinessLease(ctx); err != nil && ctx.Err() == nil {
			s.log.Error("renew controller manager readiness lease failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *ControllerManager) renewReadinessLease(ctx context.Context) error {
	now := metav1.NewMicroTime(time.Now())
	durationSeconds := int32(readinessLeaseTimeout / time.Second)
	item, err := s.leaseOperator.GetLeaseWithNamespaceEx(ctx, readinessLeaseName, resourceLockNS, "0")
	if apierrors.IsNotFound(err) {
		item = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: readinessLeaseName, Namespace: resourceLockNS},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       &s.identity,
				LeaseDurationSeconds: &durationSeconds,
				AcquireTime:          &now,
				RenewTime:            &now,
			},
		}
		_, err = s.leaseOperator.CreateLease(ctx, item)
		return err
	}
	if err != nil {
		return err
	}
	item.Spec.HolderIdentity = &s.identity
	item.Spec.LeaseDurationSeconds = &durationSeconds
	item.Spec.RenewTime = &now
	_, err = s.leaseOperator.UpdateLease(ctx, item)
	return err
}

func (s *ControllerManager) runWorkerLoops(ctx context.Context) {
	s.workerLoopLock.Lock()
	defer s.workerLoopLock.Unlock()
	for index := range s.workerLoops {
		go wait.Until(s.workerLoops[index].fn, s.workerLoops[index].Duration, ctx.Done())
	}
}

func (s *ControllerManager) runRunnables(ctx context.Context) {
	s.runnableLock.Lock()
	defer s.runnableLock.Unlock()
	for index := range s.runnables {
		go func(i int) {
			if err := s.runnables[i].Start(ctx); err != nil {
				s.log.Error("start runnalbes failed", zap.Error(err))
			}
		}(index)
	}
}
