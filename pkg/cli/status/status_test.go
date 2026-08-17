/*
 * Copyright 2026 KubeClipper Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package status

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kubeclipper/kubeclipper/pkg/platformstatus"
	"github.com/kubeclipper/kubeclipper/pkg/simple/client/kc"
)

func testStatus() *platformstatus.PlatformStatus {
	return &platformstatus.PlatformStatus{
		APIVersion: platformstatus.APIVersion,
		Kind:       platformstatus.Kind,
		Status:     platformstatus.Degraded,
		CheckedAt:  time.Date(2026, 7, 29, 8, 30, 12, 0, time.UTC),
		Components: []platformstatus.Component{
			{Name: "kc-server", Status: platformstatus.Healthy, Message: "all server subsystems ready", Checks: []platformstatus.Check{{Name: "api", Status: platformstatus.Healthy, Message: "API ready"}}},
			{Name: "kc-etcd", Status: platformstatus.Healthy, Message: "etcd is healthy"},
			{Name: "kc-agent", Status: platformstatus.Degraded, Message: "7/8 agents running"},
		},
	}
}

func TestPrintTable(t *testing.T) {
	var output bytes.Buffer
	if err := printStatus(&output, testStatus(), "table"); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"KubeClipper Platform Status: Degraded",
		"kc-server",
		"kc-etcd",
		"7/8 agents running",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("table output does not contain %q:\n%s", expected, output.String())
		}
	}
	if strings.Contains(output.String(), "kcctl doctor") {
		t.Fatalf("table output references an unavailable command:\n%s", output.String())
	}
}

func TestPrintTerminalTableUsesStatusColors(t *testing.T) {
	var output bytes.Buffer
	if err := printTerminalTable(&output, testStatus(), statusOutputStyle{enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"KubeClipper Platform Status",
		"\x1b[1;33m! Degraded",
		"\x1b[1;32m✓ Healthy",
		"7/8 agents running",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("terminal output does not contain %q:\n%s", expected, output.String())
		}
	}
}

func TestPadRightUsesDisplayCharacters(t *testing.T) {
	if got, want := padRight("✓ Healthy", statusColumnWidth), "✓ Healthy   "; got != want {
		t.Fatalf("padded status = %q, want %q", got, want)
	}
}

func TestPrintJSONPreservesChecks(t *testing.T) {
	var output bytes.Buffer
	if err := printStatus(&output, testStatus(), "json"); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"kind": "PlatformStatus"`, `"name": "api"`, `"status": "Degraded"`} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("JSON output does not contain %q:\n%s", expected, output.String())
		}
	}
}

func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name       string
		status     platformstatus.Status
		serverBody string
		delay      time.Duration
		timeout    time.Duration
		wantCode   int
	}{
		{name: "healthy", status: platformstatus.Healthy, wantCode: 0},
		{name: "degraded", status: platformstatus.Degraded, wantCode: 1},
		{name: "unhealthy", status: platformstatus.Unhealthy, wantCode: 1},
		{name: "unknown", status: platformstatus.Unknown, wantCode: 1},
		{name: "invalid response", serverBody: `{}`, wantCode: 2},
		{name: "request timeout", status: platformstatus.Healthy, delay: 100 * time.Millisecond, timeout: 10 * time.Millisecond, wantCode: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if test.delay > 0 {
					time.Sleep(test.delay)
				}
				writer.Header().Set("Content-Type", "application/json")
				if test.serverBody != "" {
					if _, err := writer.Write([]byte(test.serverBody)); err != nil {
						t.Errorf("write response: %v", err)
					}
					return
				}
				result := statusWithOverallStatus(test.status)
				if err := json.NewEncoder(writer).Encode(result); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer server.Close()

			client, err := kc.NewClientWithOpts(kc.WithEndpoint(server.URL), kc.WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			opts := &Options{client: client, output: "json", timeout: time.Second}
			opts.Out = &output
			if test.timeout > 0 {
				opts.timeout = test.timeout
			}
			err = opts.run(context.Background())
			if got := commandExitCode(err); got != test.wantCode {
				t.Fatalf("exit code = %d, want %d (error: %v)", got, test.wantCode, err)
			}
		})
	}
}

func statusWithOverallStatus(status platformstatus.Status) *platformstatus.PlatformStatus {
	result := testStatus()
	result.Status = status
	result.Components = append([]platformstatus.Component(nil), result.Components...)
	switch status {
	case platformstatus.Healthy:
		result.Components[2].Status = platformstatus.Healthy
	case platformstatus.Unhealthy:
		result.Components[2].Status = platformstatus.Unhealthy
	case platformstatus.Unknown:
		result.Components[2].Status = platformstatus.Unknown
	}
	return result
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}
