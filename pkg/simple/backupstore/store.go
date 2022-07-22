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

package backupstore

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
)

const (
	FSStorage = "fs"
	S3Storage = "s3"
)

var providerFactories = make(map[string]ProviderFactory)

func RegisterProvider(factory ProviderFactory) {
	providerFactories[factory.Type()] = factory
}

type ProviderFactory interface {
	Type() string
	Create() (BackupStore, error)
}

type BackupStore interface {
	Save(ctx context.Context, r io.Reader, fileName string) error
	Delete(ctx context.Context, fileName string) error
	Download(ctx context.Context, fileName string, w io.Writer) error
}

func GetProviderFactoryType() []string {
	var types []string
	for key := range providerFactories {
		types = append(types, key)
	}
	return types
}

// logProbe are operation log probe needs to be invoked proactively by services
func logProbe(ctx context.Context, cmd string, err error) {
	var errStr string
	if err != nil {
		errStr = fmt.Sprintf("\n an error occurred: %+v", err)
	}
	ln := fmt.Sprintf("[%s] + %s %s\n\n", time.Now().Format(time.RFC3339), cmd, errStr)
	if check, sErr := cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(ln)); sErr != nil {
		// detect context content and distinguish errors
		if check {
			logger.Error("get operation step log file failed: "+sErr.Error(),
				zap.String("operation", component.GetOperationID(ctx)),
				zap.String("step", component.GetStepID(ctx)),
				zap.String("cmd", cmd),
			)
		} else {
			// commands do not need to be logged
			logger.Debug("this command does not need to be logged", zap.String("cmd", cmd))
		}
	}
}
