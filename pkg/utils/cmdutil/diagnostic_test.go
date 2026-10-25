//go:build darwin || linux

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

package cmdutil

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticCommandKillsIgnoringTermGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	ec, err := RunDiagnosticCmdWithContext(ctx, "/bin/sh", "-c", `trap '' TERM; sleep 30 & echo $!; wait`)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cause lost: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("cleanup took %s", elapsed)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(ec.StdOut()))
	if err != nil {
		t.Fatal(err)
	}
	// A killed orphan can briefly remain a zombie until the host init reaps it.
	for start := time.Now(); ; {
		status := diagnosticProcessState(t, pid)
		if status == "" || strings.HasPrefix(status, "Z") {
			break
		}
		if time.Since(start) > time.Second {
			t.Fatalf("live diagnostic child %d: %s", pid, status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDiagnosticCommandCleansDescendantAfterLeaderExit(t *testing.T) {
	ec, err := RunDiagnosticCmdWithContext(context.Background(), "/bin/sh", "-c", "sleep 30 & echo $!; exit 0")
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("expected held-pipe error: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(ec.StdOut()))
	if err != nil {
		t.Fatal(err)
	}
	if status := diagnosticProcessState(t, pid); status != "" && !strings.HasPrefix(status, "Z") {
		t.Fatalf("live child: %s", status)
	}
}

func diagnosticProcessState(t *testing.T, pid int) string {
	t.Helper()
	//nolint:gosec // ps receives an integer PID from this test's child process, without a shell.
	out, err := exec.CommandContext(context.Background(), "ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	var exitErr *exec.ExitError
	if err != nil && (!errors.As(err, &exitErr) || exitErr.ExitCode() != 1) {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestDiagnosticCommandDoesNotPersistRawOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RunDiagnosticCmdWithContext(ctx, "/bin/sh", "-c", "echo should-not-run")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = RunDiagnosticCmdWithContext(context.Background(), "/definitely/missing/command")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}
