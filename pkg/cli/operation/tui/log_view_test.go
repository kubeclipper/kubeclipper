package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func makeTestOperationForLogView() *v1.Operation {
	return &v1.Operation{
		ObjectMeta: metav1.ObjectMeta{
			Name: "op-log-test",
		},
		Steps: []v1.Step{
			{
				ID:   "step-1",
				Name: "install-runtime",
				Nodes: []v1.StepNode{
					{ID: "node-1", IPv4: "10.0.0.1", Hostname: "node1"},
				},
			},
			{
				ID:   "step-2",
				Name: "configure-network",
				Nodes: []v1.StepNode{
					{ID: "node-1", IPv4: "10.0.0.1", Hostname: "node1"},
				},
			},
		},
		Status: v1.OperationStatus{
			Status: v1.OperationStatusRunning,
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
				{
					StepID: "step-2",
					Status: []v1.StepStatus{
						{
							Node:    "node-1",
							Status:  v1.StepStatusSuccessful,
							StartAt: metav1.Time{Time: time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)},
							EndAt:   metav1.Time{Time: time.Date(2025, 6, 24, 8, 10, 2, 0, time.UTC)},
						},
					},
				},
			},
		},
	}
}

func TestBuildStepEntries(t *testing.T) {
	op := makeTestOperationForLogView()
	entries := buildStepEntries(op)

	if len(entries) != 2 {
		t.Fatalf("expected 2 step entries, got %d", len(entries))
	}

	if entries[0].Name != "install-runtime" {
		t.Errorf("step 0 name = %q, want %q", entries[0].Name, "install-runtime")
	}
	if entries[0].Status != "successful" {
		t.Errorf("step 0 status = %q, want %q", entries[0].Status, "successful")
	}
	if entries[1].Name != "configure-network" {
		t.Errorf("step 1 name = %q, want %q", entries[1].Name, "configure-network")
	}

	// Check nodes are populated
	if len(entries[0].Nodes) != 1 {
		t.Errorf("step 0 nodes = %d, want 1", len(entries[0].Nodes))
	}
	if entries[0].Nodes[0].Hostname != "node1" {
		t.Errorf("node hostname = %q, want %q", entries[0].Nodes[0].Hostname, "node1")
	}
}

func TestLogModelStepSelection(t *testing.T) {
	op := makeTestOperationForLogView()
	m := NewLogModel(nil, op, 80, 24)

	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	if len(m.steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(m.steps))
	}

	// Move down to step 2
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}
	// rawContent should be cleared for new step fetch
	if m.rawContent != "" {
		t.Errorf("rawContent should be cleared on step change, got %q", m.rawContent)
	}

	// Move back up to step 1
	m, _ = m.Update(keyMsg(DefaultKeyMap.Up))
	if m.cursor != 0 {
		t.Errorf("after up: cursor = %d, want 0", m.cursor)
	}
}

func TestLogModelFollowModeToggle(t *testing.T) {
	op := makeTestOperationForLogView()
	m := NewLogModel(nil, op, 80, 24)

	if m.followMode {
		t.Error("follow mode should be off initially")
	}

	// Toggle follow mode on
	m, _ = m.Update(keyMsg(DefaultKeyMap.Follow))
	if !m.followMode {
		t.Error("follow mode should be on after toggle")
	}

	// Toggle follow mode off
	m, _ = m.Update(keyMsg(DefaultKeyMap.Follow))
	if m.followMode {
		t.Error("follow mode should be off after second toggle")
	}
}

func TestLogModelBackMessage(t *testing.T) {
	op := makeTestOperationForLogView()
	m := NewLogModel(nil, op, 80, 24)

	_, cmd := m.Update(keyMsg(DefaultKeyMap.Back))
	if cmd == nil {
		t.Fatal("expected a command from back key")
	}
}

func TestLogModelCursorBounds(t *testing.T) {
	op := makeTestOperationForLogView()
	m := NewLogModel(nil, op, 80, 24)

	// Try to move up past first step
	m, _ = m.Update(keyMsg(DefaultKeyMap.Up))
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Move to last step
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	// Try to move down past last step
	m, _ = m.Update(keyMsg(DefaultKeyMap.Down))
	if m.cursor != 1 {
		t.Errorf("cursor should stay at 1, got %d", m.cursor)
	}
}

func TestGetStepEndTime(t *testing.T) {
	early := time.Date(2025, 6, 24, 8, 9, 57, 0, time.UTC)
	late := time.Date(2025, 6, 24, 8, 10, 2, 0, time.UTC)

	op := &v1.Operation{
		Status: v1.OperationStatus{
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{Node: "n1", EndAt: metav1.Time{Time: early}},
						{Node: "n2", EndAt: metav1.Time{Time: late}},
					},
				},
			},
		},
	}

	got := getStepEndTime(op, "step-1")
	if !got.Time.Equal(late) {
		t.Errorf("getStepEndTime() = %v, want %v", got.Time, late)
	}
}

func TestGetStepStartTimeTUI(t *testing.T) {
	early := time.Date(2025, 6, 24, 8, 9, 50, 0, time.UTC)
	later := time.Date(2025, 6, 24, 8, 9, 55, 0, time.UTC)

	op := &v1.Operation{
		Status: v1.OperationStatus{
			Conditions: []v1.OperationCondition{
				{
					StepID: "step-1",
					Status: []v1.StepStatus{
						{Node: "n1", StartAt: metav1.Time{Time: later}},
						{Node: "n2", StartAt: metav1.Time{Time: early}},
					},
				},
			},
		},
	}

	got := getStepStartTime(op, "step-1")
	if !got.Time.Equal(early) {
		t.Errorf("getStepStartTime() = %v, want %v", got.Time, early)
	}
}

func TestStepStatusMark(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{string(v1.OperationStatusSuccessful), StepSuccessMark},
		{string(v1.OperationStatusFailed), StepFailedMark},
		{string(v1.OperationStatusRunning), StepRunningMark},
		{"pending", StepPendingMark},
		{"unknown", StepPendingMark},
	}

	for _, tt := range tests {
		got := stepStatusMark(tt.status)
		if got != tt.want {
			t.Errorf("stepStatusMark(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestLogModelViewNoSteps(t *testing.T) {
	op := &v1.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-op"},
	}
	m := NewLogModel(nil, op, 80, 24)
	view := m.View()
	if view != "No steps in this operation." {
		t.Errorf("empty log view = %q, want %q", view, "No steps in this operation.")
	}
}

func TestLogModelLogFetchedMsgUpdatesContent(t *testing.T) {
	op := makeTestOperationForLogView()
	m := NewLogModel(nil, op, 80, 24)

	// Simulate a logFetchedMsg
	m, _ = m.Update(logFetchedMsg{content: "line1\nline2\n", offset: 13})
	if m.rawContent != "line1\nline2\n" {
		t.Errorf("rawContent = %q, want %q", m.rawContent, "line1\nline2\n")
	}

	// Offset should be tracked
	key := "step-1|node-1"
	if m.lastOffset[key] != 13 {
		t.Errorf("lastOffset[%q] = %d, want 13", key, m.lastOffset[key])
	}
}

func TestLogModelTickRunningStatusContinuesFollow(t *testing.T) {
	op := makeTestOperationForLogView()
	op.Status.Status = v1.OperationStatusRunning
	m := NewLogModel(nil, op, 80, 24)
	m.followMode = true

	// Send tick message -- should keep follow mode for running status
	m, _ = m.Update(tickMsg{})
	if !m.followMode {
		t.Error("follow mode should remain on for running status")
	}
}

func TestLogModelOperationStatusMsgDisablesFollow(t *testing.T) {
	op := makeTestOperationForLogView()
	op.Status.Status = v1.OperationStatusRunning
	m := NewLogModel(nil, op, 80, 24)
	m.followMode = true

	// Simulate a refreshed operation with terminal status
	failedOp := op.DeepCopy()
	failedOp.Status.Status = v1.OperationStatusFailed
	m, cmd := m.Update(operationStatusMsg{operation: failedOp})
	if m.followMode {
		t.Error("follow mode should be disabled after operationStatusMsg with terminal status")
	}
	if cmd != nil {
		t.Error("expected no further commands when follow mode is disabled due to terminal status")
	}
	if !containsSubstring(m.rawContent, "Operation completed") {
		t.Errorf("rawContent should contain completion notice, got %q", m.rawContent)
	}
}

func TestLogModelOperationStatusMsgRunningContinuesFollow(t *testing.T) {
	op := makeTestOperationForLogView()
	op.Status.Status = v1.OperationStatusRunning
	m := NewLogModel(nil, op, 80, 24)
	m.followMode = true

	// Simulate a refreshed operation that is still running
	runningOp := op.DeepCopy()
	m, _ = m.Update(operationStatusMsg{operation: runningOp})
	if !m.followMode {
		t.Error("follow mode should remain on when operation is still running")
	}
}

func TestStatusStyle(t *testing.T) {
	tests := []struct {
		status string
	}{
		{string(v1.OperationStatusSuccessful)},
		{string(v1.OperationStatusFailed)},
		{string(v1.OperationStatusRunning)},
		{"pending"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			s := statusStyle(tt.status)
			// Just verify it doesn't panic and returns a non-zero style
			rendered := s.Render(tt.status)
			if rendered == "" {
				t.Errorf("statusStyle(%q).Render() returned empty string", tt.status)
			}
		})
	}
}

// containsSubstring checks if sub is in s.
func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Verify that the keyMsg helper from list_view_test.go also works here.
// We need it since both files are in the same package, but we re-declare
// the function here for completeness in case tests are run independently.
// However, since both files are in the same package, the keyMsg from
// list_view_test.go is shared. This comment is left for clarity.
var _ = tea.KeyMsg{} // ensure tea import is used
