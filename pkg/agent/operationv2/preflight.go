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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

const NodePreflightExecutorName = "NodePreflight/v1"

const maxPreflightChecks = 64

type NodePreflightPayload struct {
	RequiredExecutables []string `json:"requiredExecutables,omitempty"`
	RequiredPaths       []string `json:"requiredPaths,omitempty"`
}

// NodePreflight is intentionally read-only. It is the first real executor used
// to prove Task List/Watch, restart, timeout, status, output, and log behavior.
type NodePreflight struct{}

func (NodePreflight) Reconcile(ctx context.Context, task *operations.OperationTask, log io.Writer) (operations.TaskResult, error) {
	var payload NodePreflightPayload
	decoder := json.NewDecoder(bytes.NewReader(task.Spec.Payload.Raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return operations.TaskResult{}, fmt.Errorf("decode NodePreflight/v1 payload: %w", err)
	}
	if len(payload.RequiredExecutables)+len(payload.RequiredPaths) > maxPreflightChecks {
		return operations.TaskResult{}, fmt.Errorf("preflight supports at most %d checks", maxPreflightChecks)
	}

	missing := make([]string, 0)
	for _, name := range payload.RequiredExecutables {
		if err := ctx.Err(); err != nil {
			return operations.TaskResult{}, err
		}
		if name == "" || strings.ContainsRune(name, os.PathSeparator) {
			return operations.TaskResult{}, fmt.Errorf("invalid executable name %q", name)
		}
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, "executable:"+name)
		}
	}
	for _, path := range payload.RequiredPaths {
		if err := ctx.Err(); err != nil {
			return operations.TaskResult{}, err
		}
		if path == "" {
			return operations.TaskResult{}, fmt.Errorf("required path must not be empty")
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, "path:"+path)
				continue
			}
			return operations.TaskResult{}, fmt.Errorf("stat %q: %w", path, err)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		_, _ = fmt.Fprintf(log, "preflight missing: %s\n", strings.Join(missing, ", "))
		return operations.TaskResult{}, fmt.Errorf("node preflight failed: %s", strings.Join(missing, ", "))
	}

	hostname, err := os.Hostname()
	if err != nil {
		return operations.TaskResult{}, fmt.Errorf("read hostname: %w", err)
	}
	_, _ = fmt.Fprintln(log, "node preflight passed")
	return operations.TaskResult{Outputs: map[string]string{
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
	}}, nil
}
