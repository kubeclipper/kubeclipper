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

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app"
	"github.com/kubeclipper/kubeclipper/pkg/cli/logger"
)

func main() {
	cmds := app.NewKubeClipperCommand(os.Stdin, os.Stdout, os.Stderr)
	if err := cmds.Execute(); err != nil {
		code := 1
		var exitError interface{ ExitCode() int }
		if errors.As(err, &exitError) {
			code = exitError.ExitCode()
		}
		if err.Error() != "" {
			_, _ = fmt.Fprintln(os.Stderr, logger.ColorizeError(err.Error()))
		}
		os.Exit(code)
	}
}
