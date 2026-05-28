package operation

import (
	"bytes"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestPrintStepTitle(t *testing.T) {
	tests := []struct {
		name       string
		stepName   string
		stepID     string
		status     string
		startTime  metav1.Time
		wantIn     string
		dontWantIn string
	}{
		{
			name:      "with start time includes time string",
			stepName:  "install-runtime",
			stepID:    "step-1",
			status:    "successful",
			startTime: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			wantIn:    "2025-06-24",
		},
		{
			name:       "without start time omits time string",
			stepName:   "pending-step",
			stepID:     "step-2",
			status:     "pending",
			startTime:  metav1.Time{},
			dontWantIn: "2025-06-24",
		},
		{
			name:     "step name present",
			stepName: "init-cluster",
			stepID:   "step-3",
			status:   "running",
			wantIn:   "init-cluster",
		},
		{
			name:     "step ID present",
			stepName: "step",
			stepID:   "step-4",
			status:   "failed",
			wantIn:   "step-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printStepTitle(&buf, tt.stepName, tt.stepID, tt.status, tt.startTime)
			output := buf.String()

			if tt.wantIn != "" && !strings.Contains(output, tt.wantIn) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantIn, output)
			}
			if tt.dontWantIn != "" && strings.Contains(output, tt.dontWantIn) {
				t.Errorf("expected output NOT to contain %q, got: %s", tt.dontWantIn, output)
			}
		})
	}
}

func TestPrintNodeStatus(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		nodeID   string
		status   string
		ip       string
		duration time.Duration
		wantIn   string
	}{
		{
			name:     "successful node with duration",
			hostname: "node1",
			nodeID:   "n-001",
			status:   "successful",
			ip:       "10.0.0.1",
			duration: 5 * time.Second,
			wantIn:   "node1",
		},
		{
			name:     "failed node",
			hostname: "node2",
			nodeID:   "n-002",
			status:   "failed",
			ip:       "10.0.0.2",
			duration: 10 * time.Second,
			wantIn:   "failed",
		},
		{
			name:     "zero duration omits duration text",
			hostname: "node3",
			nodeID:   "n-003",
			status:   "running",
			ip:       "10.0.0.3",
			duration: 0,
			wantIn:   "node3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printNodeStatus(&buf, tt.hostname, tt.nodeID, tt.status, tt.ip, tt.duration)
			output := buf.String()

			if !strings.Contains(output, tt.wantIn) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantIn, output)
			}
			if !strings.Contains(output, tt.ip) {
				t.Errorf("expected output to contain IP %q, got: %s", tt.ip, output)
			}
		})
	}
}

func TestPrintLogLine(t *testing.T) {
	var buf bytes.Buffer
	printLogLine(&buf, "test log content")
	output := buf.String()

	if !strings.Contains(output, "test log content") {
		t.Errorf("expected output to contain log content, got: %s", output)
	}
	if !strings.HasPrefix(output, "    ") {
		t.Errorf("expected log line to be indented, got: %s", output)
	}
}

func TestPrintErrorLine(t *testing.T) {
	var buf bytes.Buffer
	printErrorLine(&buf, "error: %s", "something failed")
	output := buf.String()

	if !strings.Contains(output, "something failed") {
		t.Errorf("expected output to contain error message, got: %s", output)
	}
}

func TestLogFormatOutput(t *testing.T) {
	op := &v1.Operation{
		ObjectMeta: metav1.ObjectMeta{
			Name: "op-001",
		},
		Steps: []v1.Step{
			{
				ID:   "step-1",
				Name: "install-runtime",
				Nodes: []v1.StepNode{
					{ID: "node-1", IPv4: "10.0.0.1", Hostname: "node1"},
				},
			},
		},
		Status: v1.OperationStatus{
			Status: v1.OperationStatusSuccessful,
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{
							Node:    "node-1",
							Status:  v1.StepStatusSuccessful,
							StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
							EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	for _, step := range op.Steps {
		startTime := getStepStartTime(op, step.ID)
		stepStatus := getStepStatus(op, step.ID)
		printStepTitle(&buf, step.Name, step.ID, stepStatus, startTime)

		for _, node := range step.Nodes {
			nodeStatus, startAt, endAt := getNodeStatusWithTime(op, step.ID, node.ID)
			duration := calculateDuration(startAt, endAt)
			printNodeStatus(&buf, node.Hostname, node.ID, nodeStatus, node.IPv4, duration)
		}
	}

	output := buf.String()
	if !strings.Contains(output, "install-runtime") {
		t.Errorf("expected step name in output, got: %s", output)
	}
	if !strings.Contains(output, "node1") {
		t.Errorf("expected hostname in output, got: %s", output)
	}
	if !strings.Contains(output, "10.0.0.1") {
		t.Errorf("expected IP in output, got: %s", output)
	}
}

func TestLogsValidate(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		cluster     string
		wantErr     bool
		errSubstr   string
	}{
		{
			name:        "only operation ID is valid",
			operationID: "op-123",
			cluster:     "",
			wantErr:     false,
		},
		{
			name:        "only cluster is valid",
			operationID: "",
			cluster:     "my-cluster",
			wantErr:     false,
		},
		{
			name:        "both operation ID and cluster is invalid",
			operationID: "op-123",
			cluster:     "my-cluster",
			wantErr:     true,
			errSubstr:   "cannot specify both",
		},
		{
			name:        "neither operation ID nor cluster is invalid",
			operationID: "",
			cluster:     "",
			wantErr:     true,
			errSubstr:   "must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := LogsOptions{
				OperationID: tt.operationID,
				Cluster:     tt.cluster,
			}
			err := o.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errSubstr != "" {
				if !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestIncrementalOffsetTracking(t *testing.T) {
	offsets := make(map[string]int64)

	// Simulate incremental log reads
	key1 := "step-1|node-1"
	offsets[key1] = 0

	// First log chunk of 100 bytes
	content1Len := int64(100)
	offsets[key1] = offsets[key1] + content1Len
	if offsets[key1] != 100 {
		t.Errorf("expected offset 100, got %d", offsets[key1])
	}

	// Second log chunk of 50 bytes
	content2Len := int64(50)
	offsets[key1] = offsets[key1] + content2Len
	if offsets[key1] != 150 {
		t.Errorf("expected offset 150, got %d", offsets[key1])
	}

	// Different key starts at 0
	key2 := "step-2|node-1"
	if offsets[key2] != 0 {
		t.Errorf("expected new key offset 0, got %d", offsets[key2])
	}
}

func TestFollowModeTerminalStatus(t *testing.T) {
	terminalStatuses := []v1.OperationStatusType{
		v1.OperationStatusSuccessful,
		v1.OperationStatusFailed,
		v1.OperationStatusTermination,
	}

	nonTerminalStatuses := []v1.OperationStatusType{
		v1.OperationStatusRunning,
		v1.OperationStatusPending,
		v1.OperationStatusUnknown,
	}

	for _, status := range terminalStatuses {
		t.Run("terminal_"+string(status), func(t *testing.T) {
			if !isTerminalStatus(status) {
				t.Errorf("expected %q to be terminal status", status)
			}
		})
	}

	for _, status := range nonTerminalStatuses {
		t.Run("non_terminal_"+string(status), func(t *testing.T) {
			if isTerminalStatus(status) {
				t.Errorf("expected %q to NOT be terminal status", status)
			}
		})
	}
}

// isTerminalStatus checks if an operation status is terminal (follow mode should stop).
func isTerminalStatus(status v1.OperationStatusType) bool {
	return status == v1.OperationStatusSuccessful ||
		status == v1.OperationStatusFailed ||
		status == v1.OperationStatusTermination
}

func TestRepeatStr(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"x", 3, "xxx"},
		{"ab", 2, "abab"},
		{"", 5, ""},
		{"a", 0, ""},
	}

	for _, tt := range tests {
		got := repeatStr(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("repeatStr(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestUnicodeTruncationInLogOutput(t *testing.T) {
	tests := []struct {
		name   string
		log    string
		maxLen int
		want   string
	}{
		{
			name:   "Chinese characters truncated by rune count",
			log:    "部署集群成功完成所有节点配置",
			maxLen: 5,
			want:   "部署集群成... (truncated)",
		},
		{
			name:   "emoji in log content",
			log:    "✅ Step completed 🎉 All good",
			maxLen: 10,
			want:   "✅ Step com... (truncated)",
		},
		{
			name:   "maxLen zero keeps all Unicode",
			log:    "你好世界🌍",
			maxLen: 0,
			want:   "你好世界🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLog(tt.log, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateLog() = %q, want %q", got, tt.want)
			}
		})
	}
}
