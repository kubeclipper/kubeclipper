package operation

import (
	"bytes"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func makeTestOperation(status v1.OperationStatusType, stepStatus v1.StepStatusType) *v1.Operation {
	return &v1.Operation{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-op-123",
			CreationTimestamp: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
			Labels: map[string]string{
				common.LabelClusterName:      "test-cluster",
				common.LabelOperationAction:  "InstallCluster",
				common.LabelOperationSponsor: "admin",
			},
		},
		Steps: []v1.Step{
			{
				ID:   "step-1",
				Name: "init-cluster",
				Nodes: []v1.StepNode{
					{ID: "node-1", IPv4: "10.0.0.1", Hostname: "node1"},
				},
			},
		},
		Status: v1.OperationStatus{
			Status: status,
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{
							Node:    "node-1",
							Status:  stepStatus,
							StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 52, 0, time.UTC)},
							EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
						},
					},
				},
			},
		},
	}
}

func TestRenderOperation(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		op := makeTestOperation(v1.OperationStatusSuccessful, v1.StepStatusSuccessful)
		var buf bytes.Buffer
		renderOperation(&buf, op)
		output := buf.String()

		if !strings.Contains(output, "test-op-123") {
			t.Errorf("expected output to contain operation name, got: %s", output)
		}
		if !strings.Contains(output, "InstallCluster") {
			t.Errorf("expected output to contain action, got: %s", output)
		}
		if !strings.Contains(output, "test-cluster") {
			t.Errorf("expected output to contain cluster name, got: %s", output)
		}
		if !strings.Contains(output, "admin") {
			t.Errorf("expected output to contain sponsor, got: %s", output)
		}
		if !strings.Contains(output, "successful") {
			t.Errorf("expected output to contain status 'successful', got: %s", output)
		}
		if !strings.Contains(output, "init-cluster") {
			t.Errorf("expected output to contain step name, got: %s", output)
		}
		if !strings.Contains(output, "node1") {
			t.Errorf("expected output to contain hostname, got: %s", output)
		}
		if !strings.Contains(output, "10.0.0.1") {
			t.Errorf("expected output to contain IP, got: %s", output)
		}
	})

	t.Run("failed operation shows reason and message", func(t *testing.T) {
		op := makeTestOperation(v1.OperationStatusFailed, v1.StepStatusFailed)
		op.Status.Conditions[0].Status[0].Reason = "InstallTimeout"
		op.Status.Conditions[0].Status[0].Message = "containerd installation timed out after 10m"

		var buf bytes.Buffer
		renderOperation(&buf, op)
		output := buf.String()

		if !strings.Contains(output, "failed") {
			t.Errorf("expected output to contain status 'failed', got: %s", output)
		}
		if !strings.Contains(output, "InstallTimeout") {
			t.Errorf("expected output to contain reason, got: %s", output)
		}
		if !strings.Contains(output, "containerd installation timed out after 10m") {
			t.Errorf("expected output to contain message, got: %s", output)
		}
	})

	t.Run("running operation", func(t *testing.T) {
		op := makeTestOperation(v1.OperationStatusRunning, v1.StepStatusSuccessful)
		// Add a pending node to simulate partial progress
		op.Steps[0].Nodes = append(op.Steps[0].Nodes, v1.StepNode{
			ID: "node-2", IPv4: "10.0.0.2", Hostname: "node2",
		})

		var buf bytes.Buffer
		renderOperation(&buf, op)
		output := buf.String()

		if !strings.Contains(output, "running") {
			t.Errorf("expected output to contain 'running', got: %s", output)
		}
	})

	t.Run("multiple steps", func(t *testing.T) {
		op := makeTestOperation(v1.OperationStatusSuccessful, v1.StepStatusSuccessful)
		op.Steps = append(op.Steps, v1.Step{
			ID:   "step-2",
			Name: "configure-network",
			Nodes: []v1.StepNode{
				{ID: "node-1", IPv4: "10.0.0.1", Hostname: "node1"},
			},
		})
		op.Status.Conditions = append(op.Status.Conditions, v1.OperationCondition{
			StepID: "step-2",
			Status: []v1.StepStatus{
				{
					Node:    "node-1",
					Status:  v1.StepStatusSuccessful,
					StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
					EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 10, 5, 0, time.UTC)},
				},
			},
		})

		var buf bytes.Buffer
		renderOperation(&buf, op)
		output := buf.String()

		if !strings.Contains(output, "init-cluster") {
			t.Errorf("expected output to contain step-1 name, got: %s", output)
		}
		if !strings.Contains(output, "configure-network") {
			t.Errorf("expected output to contain step-2 name, got: %s", output)
		}
	})
}

func TestGetStepStatus(t *testing.T) {
	tests := []struct {
		name   string
		op     *v1.Operation
		stepID string
		want   string
	}{
		{
			name: "all nodes successful",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{
							StepID: "step-1",
							Status: []v1.StepStatus{
								{Node: "n1", Status: v1.StepStatusSuccessful},
								{Node: "n2", Status: v1.StepStatusSuccessful},
							},
						},
					},
				},
			},
			stepID: "step-1",
			want:   "successful",
		},
		{
			name: "one node failed",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{
							StepID: "step-1",
							Status: []v1.StepStatus{
								{Node: "n1", Status: v1.StepStatusSuccessful},
								{Node: "n2", Status: v1.StepStatusFailed},
							},
						},
					},
				},
			},
			stepID: "step-1",
			want:   "failed",
		},
		{
			name: "node with empty status is pending",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{
							StepID: "step-1",
							Status: []v1.StepStatus{
								{Node: "n1", Status: v1.StepStatusSuccessful},
								{Node: "n2", Status: ""},
							},
						},
					},
				},
			},
			stepID: "step-1",
			want:   "pending",
		},
		{
			name: "no conditions returns pending",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{},
				},
			},
			stepID: "step-1",
			want:   "pending",
		},
		{
			name: "condition with empty status list returns successful (no failed or pending nodes)",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{StepID: "step-1", Status: []v1.StepStatus{}},
					},
				},
			},
			stepID: "step-1",
			want:   "successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStepStatus(tt.op, tt.stepID)
			if got != tt.want {
				t.Errorf("getStepStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetStepNodeErrorDetail(t *testing.T) {
	tests := []struct {
		name        string
		op          *v1.Operation
		stepID      string
		nodeID      string
		wantReason  string
		wantMessage string
	}{
		{
			name: "failed node with reason and message",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{
							StepID: "step-1",
							Status: []v1.StepStatus{
								{Node: "n1", Reason: "Timeout", Message: "timed out"},
							},
						},
					},
				},
			},
			stepID:      "step-1",
			nodeID:      "n1",
			wantReason:  "Timeout",
			wantMessage: "timed out",
		},
		{
			name: "node not found returns empty",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{
							StepID: "step-1",
							Status: []v1.StepStatus{
								{Node: "n1", Reason: "SomeReason", Message: "some msg"},
							},
						},
					},
				},
			},
			stepID:      "step-1",
			nodeID:      "n999",
			wantReason:  "",
			wantMessage: "",
		},
		{
			name: "step not found returns empty",
			op: &v1.Operation{
				Status: v1.OperationStatus{
					Conditions: []v1.OperationCondition{
						{StepID: "step-1", Status: []v1.StepStatus{}},
					},
				},
			},
			stepID:      "step-999",
			nodeID:      "n1",
			wantReason:  "",
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, message := getStepNodeErrorDetail(tt.op, tt.stepID, tt.nodeID)
			if reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
			if message != tt.wantMessage {
				t.Errorf("message = %q, want %q", message, tt.wantMessage)
			}
		})
	}
}
