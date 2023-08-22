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

package utils

import (
	"context"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

// LoadImage decompress and load image
func LoadImage(ctx context.Context, dryRun bool, file, criType string) error {
	//_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "gzip", "-df", file)
	//if err != nil {
	//	return err
	//}
	//
	//file = strings.ReplaceAll(file, ".gz", "")

	switch criType {
	case "containerd":
		// ctr --namespace k8s.io image import --all-platforms xxx/images.tar
		// Why need set `--all-platforms` flag?
		// This prevents unexpected errors due to the incorrect schema type declaration of the mirroring system that cannot be imported normally.
		_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "nerdctl", "-n", "k8s.io", "load", "-i", file)
		if err != nil {
			return err
		}
	case "docker":
		// docker load -i xxx/images.tar
		_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "docker", "load", "-i", file)
		if err != nil {
			return err
		}
	}

	_, err := cmdutil.RunCmdWithContext(ctx, dryRun, "rm", "-rf", file)

	return err
}

func RetryFunc(ctx context.Context, opts component.Options, intervalTime time.Duration, funcName string, fn func(ctx context.Context, opts component.Options) error) error {
	for {
		select {
		case <-ctx.Done():
			logger.Warnf("retry function '%s' timeout...", funcName)
			return ctx.Err()
		case <-time.After(intervalTime):
			err := fn(ctx, opts)
			if err == nil {
				return nil
			}
			logger.Warnf("function '%s' running error: %s. about to enter retry", funcName, err.Error())
		}
	}
}

func UnwrapNodeList(nl component.NodeList) (nodes []v1.StepNode) {
	for _, v := range nl {
		nodes = append(nodes, v1.StepNode{
			ID:       v.ID,
			IPv4:     v.IPv4,
			NodeIPv4: v.NodeIPv4,
			Hostname: v.Hostname,
		})
	}
	return nodes
}
