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

package fileutil_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
	"github.com/kubeclipper/kubeclipper/pkg/utils/fileutil"
)

func TestWriteFileWithContextLogsResult(t *testing.T) {
	for _, tt := range []struct {
		name      string
		renderErr error
		missing   bool
	}{
		{name: "success"},
		{name: "render error", renderErr: errors.New("template execution failed")},
		{name: "open error", missing: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			logStore, err := oplog.NewOperationLog(&oplog.Options{Dir: t.TempDir(), OplogThreshold: oplog.DefaultThreshold})
			if err != nil {
				t.Fatal(err)
			}
			if err = logStore.CreateOperationDir("operation"); err != nil {
				t.Fatal(err)
			}
			ctx := component.WithOplog(context.Background(), logStore)
			ctx = component.WithOperationID(ctx, "operation")
			ctx = component.WithStepID(ctx, "step")
			path := filepath.Join(t.TempDir(), "config.toml")
			if tt.missing {
				path = filepath.Join(path, "missing", "config.toml")
			}
			err = fileutil.WriteFileWithContext(ctx, path, os.O_CREATE|os.O_WRONLY, 0600, func(w io.Writer) error {
				if tt.renderErr != nil {
					return tt.renderErr
				}
				_, writeErr := io.WriteString(w, "version = 3\n")
				return writeErr
			}, false)
			if (err != nil) != (tt.missing || tt.renderErr != nil) {
				t.Fatalf("unexpected write error: %v", err)
			}
			if tt.renderErr != nil && !errors.Is(err, tt.renderErr) {
				t.Fatalf("render error lost: %v", err)
			}
			content, _, _, readErr := logStore.GetStepLogContent("operation", "step", 0, -1)
			if readErr != nil {
				t.Fatal(readErr)
			}
			want := "+ rendered " + path
			if err != nil {
				want = "ERROR rendering " + path + ": " + err.Error()
			}
			if !strings.Contains(string(content), want) {
				t.Fatalf("log = %q, want %q", content, want)
			}
		})
	}
}
