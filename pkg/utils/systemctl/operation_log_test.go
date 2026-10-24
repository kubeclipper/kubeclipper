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

package systemctl

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kubeclipper/kubeclipper/pkg/component"
	"github.com/kubeclipper/kubeclipper/pkg/oplog"
)

func TestOperationLogKeepsCauseAndCollectsUnitDiagnostics(t *testing.T) {
	for _, tt := range []struct {
		name       string
		unit       string
		statusExit string
		canceled   bool
	}{
		{name: "kubelet failure", unit: "kubelet.service", statusExit: "0"},
		{name: "docker failure", unit: "docker.service", statusExit: "0"},
		{name: "diagnostics fail", unit: "demo.service", statusExit: "1"},
		{name: "canceled context", unit: "demo.service", statusExit: "0", canceled: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			binDir := t.TempDir()
			for command, script := range map[string]string{
				"systemctl":  "#!/bin/sh\nprintf 'ActiveState=failed\\nExecMainStatus=1\\n'\nexit " + tt.statusExit + "\n",
				"journalctl": "#!/bin/sh\nprintf 'service: failed to load configuration\\n'\n",
			} {
				writeDiagnosticCommand(t, filepath.Join(binDir, command), script)
			}
			t.Setenv("PATH", binDir)
			ctx, logStore := operationLogContext(t)
			if tt.canceled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			var operationErr error
			finish := logOperation(ctx, "restart", tt.unit, &operationErr)
			cause := errors.New("D-Bus job failed")
			operationErr = cause
			finish()
			if !errors.Is(operationErr, cause) || !strings.Contains(operationErr.Error(), "systemctl restart "+tt.unit) {
				t.Fatalf("original service error lost: %v", operationErr)
			}
			content, _, _, err := logStore.GetStepLogContent("operation", "step", 0, -1)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), "ActiveState=failed") || !strings.Contains(string(content), "service: failed to load configuration") {
				t.Fatalf("diagnostic output missing from step log, including after cancellation: %s", content)
			}
			if !strings.Contains(string(content), "+ systemctl restart "+tt.unit+" (via D-Bus)") || !strings.Contains(string(content), "ERROR: "+operationErr.Error()) {
				t.Fatalf("service action or failure missing from step log: %s", content)
			}
			if tt.statusExit == "1" && !strings.Contains(string(content), "WARNING: collect diagnostics for "+tt.unit+" (systemctl): exit status 1") {
				t.Fatalf("diagnostic error missing from step log: %s", content)
			}
		})
	}
}

func TestOperationLogSuccessDoesNotCollectDiagnostics(t *testing.T) {
	ctx, logStore := operationLogContext(t)
	var operationErr error
	finish := logOperation(ctx, "daemon-reload", "", &operationErr)
	content, _, _, err := logStore.GetStepLogContent("operation", "step", 0, -1)
	if err != nil || !strings.Contains(string(content), "+ systemctl daemon-reload (via D-Bus)") {
		t.Fatalf("action was not logged before execution: %s, %v", content, err)
	}
	finish()
	content, _, _, err = logStore.GetStepLogContent("operation", "step", 0, -1)
	if err != nil || operationErr != nil || !strings.Contains(string(content), "+ completed systemctl daemon-reload") || strings.Contains(string(content), "journalctl") {
		t.Fatalf("unexpected success log: %s, %v", content, err)
	}
}

func operationLogContext(t *testing.T) (context.Context, component.OperationLogFile) {
	t.Helper()
	logStore, err := oplog.NewOperationLog(&oplog.Options{Dir: t.TempDir(), OplogThreshold: oplog.DefaultThreshold})
	if err != nil {
		t.Fatal(err)
	}
	if err := logStore.CreateOperationDir("operation"); err != nil {
		t.Fatal(err)
	}
	ctx := component.WithOplog(context.Background(), logStore)
	ctx = component.WithOperationID(ctx, "operation")
	return component.WithStepID(ctx, "step"), logStore
}

func writeDiagnosticCommand(t *testing.T, path, script string) {
	t.Helper()
	//nolint:gosec // Test probes require owner execute permission in their test-owned temporary directory.
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
}

func TestWaitJobResult(t *testing.T) {
	for _, result := range []string{"done", "failed", "canceled", "timeout"} {
		results := make(chan string, 1)
		results <- result
		err := waitJobResult(context.Background(), results)
		if (err == nil) != (result == "done") {
			t.Fatalf("result %q: %v", result, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := make(chan string, 1)
	if err := waitJobResult(ctx, results); !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cancellation is not confirmed") {
		t.Fatalf("cause or cancellation semantics lost: %v", err)
	}
	// go-systemd must be able to deliver the late result after we stop waiting.
	results <- "done"
}

func TestDiagnosticsBoundsAndSanitizesOutput(t *testing.T) {
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	writeDiagnosticCommand(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\nprintf 'LoadState=not-found\\n'\n")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsFile + "'\ni=0; while [ $i -lt 60 ]; do printf 'old entry %s\\n' $i; i=$((i+1)); done\n" +
		"printf '\\033[31m password=keep-secret token=hidden-token Authorization: Bearer secret-token amqp://user:secret@host\\033[0m\\n'\n" +
		"i=0; while [ $i -lt 24000 ]; do printf '中文'; i=$((i+1)); done\nprintf '\\n'\n"
	writeDiagnosticCommand(t, filepath.Join(bin, "journalctl"), script)
	t.Setenv("PATH", bin)
	ctx, logs := operationLogContext(t)
	collectUnitDiagnostics(ctx, "example.service", time.Now())
	content, _, _, err := logs.GetStepLogContent("operation", "step", 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !utf8.Valid(content) || !strings.Contains(text, "truncated") || !strings.Contains(text, "LoadState=not-found") {
		t.Fatalf("missing safe bounds or status: %s", text[:min(len(text), 1000)])
	}
	for _, forbidden := range []string{"old entry 0", "keep-secret", "hidden-token", "secret-token", "user:secret", "\x1b"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unbounded or raw diagnostics contain %q", forbidden)
		}
	}
	if len(content) > diagnosticOutputLimit+3000 {
		t.Fatalf("output exceeded bound: %d", len(content))
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--boot=0", "--since=", " UTC", "--until=", "--unit=example.service"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("missing journal window %q: %s", want, args)
		}
	}
}

func TestDiagnosticsMissingAndPermissionErrorsKeepCause(t *testing.T) {
	for _, mode := range []string{"missing", "denied", "empty"} {
		t.Run(mode, func(t *testing.T) {
			bin := t.TempDir()
			if mode != "missing" {
				script := "#!/bin/sh\n"
				if mode == "denied" {
					script += "printf 'permission denied\\n' >&2; exit 1\n"
				}
				for _, name := range []string{"systemctl", "journalctl"} {
					writeDiagnosticCommand(t, filepath.Join(bin, name), script)
				}
			}
			t.Setenv("PATH", bin)
			ctx, logs := operationLogContext(t)
			cause := errors.New("original job failure")
			operationErr := cause
			finish := logOperation(ctx, "restart", "example.service", &operationErr)
			finish()
			if !errors.Is(operationErr, cause) {
				t.Fatal(operationErr)
			}
			data, _, _, readErr := logs.GetStepLogContent("operation", "step", 0, -1)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if mode == "empty" {
				if !strings.Contains(string(data), "No diagnostic output available") {
					t.Fatal(string(data))
				}
			} else if !strings.Contains(string(data), "WARNING: collect diagnostics") {
				t.Fatal(string(data))
			}
		})
	}
}

func TestDiagnosticBudgetIncludesIgnoredTermCleanup(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"systemctl", "journalctl"} {
		writeDiagnosticCommand(t, filepath.Join(bin, name), "#!/bin/sh\ntrap '' TERM\nwhile :; do :; done\n")
	}
	t.Setenv("PATH", bin)
	ctx, logs := operationLogContext(t)
	start := time.Now()
	collectUnitDiagnostics(ctx, "example.service", start)
	if elapsed := time.Since(start); elapsed > 6*time.Second {
		t.Fatalf("diagnostics took %s", elapsed)
	}
	data, _, _, readErr := logs.GetStepLogContent("operation", "step", 0, -1)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(data), "context deadline exceeded") {
		t.Fatal(string(data))
	}
}

func TestManagerFailureWithoutUnitStillLogsCause(t *testing.T) {
	ctx, logs := operationLogContext(t)
	cause := errors.New("manager permission denied")
	err := cause
	finish := logOperation(ctx, "daemon-reload", "", &err)
	finish()
	data, _, _, readErr := logs.GetStepLogContent("operation", "step", 0, -1)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !errors.Is(err, cause) || !strings.Contains(string(data), "ERROR: systemctl daemon-reload: manager permission denied") || strings.Contains(string(data), "journalctl") {
		t.Fatalf("manager failure lost: %s", data)
	}
}

func TestOversizedPEMIsRedacted(t *testing.T) {
	bin := t.TempDir()
	payload := "-----BEGIN PRIVATE KEY-----\n" + strings.Repeat("QUJDQUJDQUJDQUJD\n", 70000) + "VERIFIER_FAKE_PRIVATE_KEY_BODY\n-----END PRIVATE KEY-----\n"
	body := filepath.Join(bin, "body")
	if err := os.WriteFile(body, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	writeDiagnosticCommand(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\nprintf 'ActiveState=failed\\n'\n")
	writeDiagnosticCommand(t, filepath.Join(bin, "journalctl"), "#!/bin/sh\n/bin/cat '"+body+"'\n")
	t.Setenv("PATH", bin)
	ctx, logs := operationLogContext(t)
	collectUnitDiagnostics(ctx, "verifier.service", time.Now())
	data, _, _, err := logs.GetStepLogContent("operation", "step", 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "VERIFIER_FAKE_PRIVATE_KEY_BODY") {
		t.Fatal("recognized PEM body leaked after 1MiB tail capture removed BEGIN marker")
	}
}

func TestDiagnosticsRedactsQuotedPasswords(t *testing.T) {
	bin := t.TempDir()
	body := filepath.Join(bin, "journal")
	payload := "{\"password\": \"fake-alpha fake-beta\", \"detail\": \"startup failed\"}\n" +
		"{\"password\" : \"fake-gamma\"}\n"
	if err := os.WriteFile(body, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	writeDiagnosticCommand(t, filepath.Join(bin, "systemctl"), "#!/bin/sh\nprintf 'ActiveState=failed\\n'\n")
	writeDiagnosticCommand(t, filepath.Join(bin, "journalctl"), "#!/bin/sh\n/bin/cat '"+body+"'\n")
	t.Setenv("PATH", bin)
	ctx, logs := operationLogContext(t)
	collectUnitDiagnostics(ctx, "example.service", time.Now())
	data, _, _, err := logs.GetStepLogContent("operation", "step", 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, secret := range []string{"fake-alpha", "fake-beta", "fake-gamma"} {
		if strings.Contains(content, secret) {
			t.Errorf("password content %q survived in task log", secret)
		}
	}
	if strings.Count(content, "[REDACTED]") != 2 || !strings.Contains(content, "startup failed") {
		t.Errorf("redaction lost diagnostic context: %s", content)
	}
}
