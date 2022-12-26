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

package agent

import (
	"github.com/pkg/errors"
	"github.com/txn2/txeh"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	"github.com/kubeclipper/kubeclipper/pkg/agent/config"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/service"
	"github.com/kubeclipper/kubeclipper/pkg/service/task"
)

type Server struct {
	taskService service.Interface
	Config      *config.Config
}

func (s *Server) PrepareRun(stopCh <-chan struct{}) error {
	opLog, err := oplog.NewOperationLog(s.Config.OpLogOptions)
	if err != nil {
		return err
	}
	err = configHosts(s.Config.Metadata.ProxyServer)
	if err != nil {
		return errors.WithMessage(err, "config hosts")
	}
	s.taskService = task.NewService(s.Config.AgentID, s.Config.Metadata.Region, s.Config.IPDetect, s.Config.NodeIPDetect, s.Config.RegisterNode, s.Config.MQOptions,
		task.WithNodeStatusUpdateFrequency(s.Config.NodeStatusUpdateFrequency),
		task.WithLeaseDurationSeconds(240),
		task.WithOplog(opLog),
		task.WithRepoMirror(s.Config.ImageProxyOptions.KcImageRepoMirror),
	)
	return s.taskService.PrepareRun(stopCh)
}

// config hosts if proxyServer specified.
func configHosts(proxyServer string) error {
	hosts, err := txeh.NewHostsDefault()
	if err != nil {
		return err
	}
	if proxyServer != "" {
		hosts.AddHost(proxyServer, options.NatsAltNameProxy)
	} else {
		hosts.RemoveHost(options.NatsAltNameProxy)
	}
	return hosts.Save()
}

func (s *Server) Run(stopCh <-chan struct{}) error {
	if err := s.taskService.Run(stopCh); err != nil {
		return err
	}
	<-stopCh
	logger.Debugf("get stopCh signal, exit...")
	s.taskService.Close()
	return nil
}
