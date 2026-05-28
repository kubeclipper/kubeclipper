package kc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	corev1 "github.com/kubeclipper/kubeclipper/pkg/apis/core/v1"
)

func TestGetStepNodeLogURL(t *testing.T) {
	tests := []struct {
		name       string
		opID       string
		stepID     string
		nodeID     string
		offset     int64
		wantPath   string
		wantParams map[string]string
	}{
		{
			name:     "basic request with zero offset",
			opID:     "op-001",
			stepID:   "step-1",
			nodeID:   "node-1",
			offset:   0,
			wantPath: "/api/core.kubeclipper.io/v1/logs",
			wantParams: map[string]string{
				"operation": "op-001",
				"step":      "step-1",
				"node":      "node-1",
				"offset":    "0",
			},
		},
		{
			name:     "request with non-zero offset",
			opID:     "op-002",
			stepID:   "step-2",
			nodeID:   "node-2",
			offset:   1500,
			wantPath: "/api/core.kubeclipper.io/v1/logs",
			wantParams: map[string]string{
				"operation": "op-002",
				"step":      "step-2",
				"node":      "node-2",
				"offset":    "1500",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest *http.Request
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				stepLog := corev1.StepLog{
					Content:      "test log content",
					Node:         tt.nodeID,
					DeliverySize: 16,
					LogSize:      16,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(stepLog)
			}))
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			cli := &Client{
				host:   u.Host,
				scheme: "http",
				client: http.DefaultClient,
			}

			_, err := cli.GetStepNodeLog(context.Background(), tt.opID, tt.stepID, tt.nodeID, tt.offset)
			if err != nil {
				t.Fatalf("GetStepNodeLog() error: %v", err)
			}

			if capturedRequest == nil {
				t.Fatal("no request captured")
			}

			if capturedRequest.URL.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", capturedRequest.URL.Path, tt.wantPath)
			}

			for key, want := range tt.wantParams {
				got := capturedRequest.URL.Query().Get(key)
				if got != want {
					t.Errorf("query param %q = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestRetryOperationURL(t *testing.T) {
	var capturedRequest *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cli := &Client{
		host:   u.Host,
		scheme: "http",
		client: http.DefaultClient,
	}

	opID := "op-retry-123"
	err := cli.RetryOperation(context.Background(), opID)
	if err != nil {
		t.Fatalf("RetryOperation() error: %v", err)
	}

	if capturedRequest == nil {
		t.Fatal("no request captured")
	}

	expectedPath := fmt.Sprintf("%s/%s/%s", operationPath, opID, "retry")
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("path = %q, want %q", capturedRequest.URL.Path, expectedPath)
	}
	if capturedRequest.Method != "POST" {
		t.Errorf("method = %q, want POST", capturedRequest.Method)
	}
}

func TestTerminateOperationURL(t *testing.T) {
	var capturedRequest *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cli := &Client{
		host:   u.Host,
		scheme: "http",
		client: http.DefaultClient,
	}

	opID := "op-term-456"
	err := cli.TerminateOperation(context.Background(), opID)
	if err != nil {
		t.Fatalf("TerminateOperation() error: %v", err)
	}

	if capturedRequest == nil {
		t.Fatal("no request captured")
	}

	expectedPath := fmt.Sprintf("%s/%s/%s", operationPath, opID, "termination")
	if capturedRequest.URL.Path != expectedPath {
		t.Errorf("path = %q, want %q", capturedRequest.URL.Path, expectedPath)
	}
	if capturedRequest.Method != "POST" {
		t.Errorf("method = %q, want POST", capturedRequest.Method)
	}
}
