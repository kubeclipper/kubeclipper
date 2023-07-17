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

package k8s

import (
	"context"
	"strings"
	"syscall"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/utils/initsystem"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/strutil"
)

func getJoinCmdFromStdOut(output string, cutBegin string) string {
	outputSlice := strings.Split(output, "\n")
	controlPlaneJoinStartIndex := 0
	for index, val := range outputSlice {
		if val == cutBegin {
			controlPlaneJoinStartIndex = index
		}
	}
	result := strings.Join([]string{strings.TrimSpace(outputSlice[controlPlaneJoinStartIndex+2]), strings.TrimSpace(outputSlice[controlPlaneJoinStartIndex+3]), strings.TrimSpace(outputSlice[controlPlaneJoinStartIndex+4])}, "\n")
	return strings.Replace(result, "\\\n", "", -1)
}

func generateKubeConfig(ctx context.Context) error {
	if err := fileutil.CreateDirIfNotExists("/root/.kube", 0755); err != nil {
		return err
	}
	_, err := cmdutil.RunCmdWithContext(ctx, false, "cp", "-rf", "/etc/kubernetes/admin.conf", "/root/.kube/config")
	if err != nil {
		return err
	}
	return nil
}

func doCommandRemoveStep(name string, nodes []v1.StepNode, dirs ...string) v1.Step {
	return v1.Step{
		ID:         strutil.GetUUID(),
		Name:       name,
		Timeout:    metav1.Duration{Duration: 5 * time.Second},
		ErrIgnore:  true,
		Nodes:      nodes,
		RetryTimes: 1,
		Action:     v1.ActionUninstall,
		Commands: []v1.Command{
			{
				Type:         v1.CommandShell,
				ShellCommand: append([]string{"rm", "-rf"}, dirs...),
			},
		},
	}
}

func deleteContainer(namespace string) error {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), namespace)

	ctrs, err := client.Containers(ctx)
	if err != nil {
		return err
	}

	logger.Infof("current namespace task num is %d. task %v", len(ctrs), ctrs)

	for _, ctr := range ctrs {
		logger.Debugf("Attempt to kill and delete task belong to container %s", ctr.ID())
		if task, err := ctr.Task(ctx, nil); err != nil {
			logger.Warnf("Failed to get task of container %s , it may has not task at all, let move on", ctr.ID())
		} else {
			logger.Debugf("Attempt to kill task %s", task.ID())
			exitStatusC, err := task.Wait(ctx)
			if err != nil {
				logger.Errorf("Failed to get status chan due to error %s", err)
				return err
			}
			if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
				if errdefs.IsNotFound(err) {
					logger.Errorf("Task kill %s error: %s", task.ID(), err.Error())
					continue
				}
				logger.Errorf("Failed to kill task due to error %s", err)
				return err
			}
			logger.Debugf("Containerd task %s killed", task.ID())
			logger.Debugf("Wait for task %s exit signal", task.ID())
			status := <-exitStatusC
			logger.Debugf("Got signal from task %s", task.ID())
			code, _, err := status.Result()
			if err != nil {
				logger.Errorf("Failed to get task result due to error %s", err)
				return err
			}
			logger.Debugf("Got task exit signal %v", code)
			logger.Debugf("Attempt to delete task %s", task.ID())
			if statusCode, err := task.Delete(ctx); err != nil {
				logger.Errorf("(ignore) Failed to delete task %s due to error: %s since task already been killed it`s ok to leave it alone", task.ID(), err)
			} else {
				logger.Debugf("Task %s deleted and got exit code %v", task.ID(), statusCode.ExitCode())
			}
		}
		logger.Debugf("Attempt to delete container %s", ctr.ID())
		if err := ctr.Delete(ctx); err != nil {
			logger.Errorf("(ignored) Failed to delete container %s due to error: %s but container without task can be consider harmless let`s move on", ctr.ID(), err.Error())
			continue
		}
	}

	return nil
}

// isServiceActive checks whether the given service exists and is running
func isServiceActive(name string) (bool, error) {
	initSystem, err := initsystem.GetInitSystem()
	if err != nil {
		return false, err
	}
	return initSystem.ServiceIsActive(name), nil
}
