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
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kubeclipper/kubeclipper/pkg/logger"
	"github.com/kubeclipper/kubeclipper/pkg/utils/cmdutil"
	"github.com/kubeclipper/kubeclipper/pkg/utils/redact"
)

// logOperation records the equivalent systemctl action before the D-Bus call,
// then its result. The returned function must be deferred by the caller.
//
//nolint:gocritic // The deferred closure must read and replace the caller's final error.
func logOperation(ctx context.Context, action, unit string, operationErr *error) func() {
	startedAt := time.Now()
	command := "systemctl " + action
	if unit != "" {
		command += " " + unit
	}
	recording := appendOperationLog(ctx, "+ "+command+" (via D-Bus)")
	return func() {
		if *operationErr == nil {
			appendOperationLog(ctx, "+ completed "+command)
			return
		}
		*operationErr = fmt.Errorf("%s: %w", command, *operationErr)
		appendOperationLog(ctx, "ERROR: "+(*operationErr).Error())
		if recording && unit != "" {
			collectUnitDiagnostics(ctx, unit, startedAt)
		}
	}
}

func appendOperationLog(ctx context.Context, message string) bool {
	line := fmt.Sprintf("[%s] %s\n\n", time.Now().Format(time.RFC3339), message)
	check, err := cmdutil.CheckContextAndAppendStepLogFile(ctx, []byte(line))
	if check && err != nil {
		logger.Errorf("write systemd operation to step log: %v", err)
	}
	return check
}

const (
	diagnosticOutputLimit  = 64 << 10
	diagnosticTimeout      = 5 * time.Second
	diagnosticJournalLimit = 50
)

func collectUnitDiagnostics(ctx context.Context, unit string, startedAt time.Time) {
	diagnosticCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), diagnosticTimeout)
	defer cancel()
	since := startedAt.Add(-time.Second).UTC().Format("2006-01-02 15:04:05.000000 UTC")
	until := time.Now().UTC().Format("2006-01-02 15:04:05.000000 UTC")
	for _, command := range [][]string{
		{"systemctl", "show", "--no-pager", "--property=LoadState,ActiveState,SubState,Result,ExecMainCode,ExecMainStatus", "--", unit},
		// One lookahead entry lets us explicitly identify a truncated journal.
		{"journalctl", "--boot=0", "--unit=" + unit, "--since=" + since, "--until=" + until,
			fmt.Sprintf("--lines=%d", diagnosticJournalLimit+1), "--no-pager", "--output=short-iso"},
	} {
		appendOperationLog(ctx, "+ diagnostics: "+strings.Join(command, " ")+" (journal limit: 50 entries, 65536 bytes)")
		ec, err := cmdutil.RunDiagnosticCmdWithContext(diagnosticCtx, command[0], command[1:]...)
		output := strings.ToValidUTF8(redact.Text(ec.StdOut()+ec.StdErr()), "�")
		truncated := false
		if command[0] == "journalctl" {
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
			if len(lines) > diagnosticJournalLimit {
				output = strings.Join(lines[len(lines)-diagnosticJournalLimit:], "\n")
				truncated = true
			}
		}
		if len(output) > diagnosticOutputLimit {
			output = output[len(output)-diagnosticOutputLimit:]
			for output != "" && !utf8.RuneStart(output[0]) {
				output = output[1:]
			}
			truncated = true
		}
		if truncated {
			appendOperationLog(ctx, "WARNING: diagnostic output truncated; showing latest entries within the stated limits")
		}
		if strings.TrimSpace(output) == "" || strings.TrimSpace(output) == "-- No entries --" {
			appendOperationLog(ctx, "No diagnostic output available from "+command[0]+" in the requested window")
		} else {
			appendOperationLog(ctx, output)
		}
		if err != nil {
			message := fmt.Sprintf("WARNING: collect diagnostics for %s (%s): %v", unit, command[0], err)
			message = redact.Text(message)
			logger.Warn(message)
			appendOperationLog(ctx, message)
		}
	}
}
