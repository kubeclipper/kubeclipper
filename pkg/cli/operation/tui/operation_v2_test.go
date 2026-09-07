package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestListViewUsesV2PhaseAndSteps(t *testing.T) {
	op := operationsv1alpha1.Operation{ObjectMeta: metav1.ObjectMeta{Name: "op"}, Spec: operationsv1alpha1.OperationSpec{Action: "Install", Steps: []operationsv1alpha1.OperationStep{{ID: "one"}}}, Status: operationsv1alpha1.OperationStatus{Phase: operationsv1alpha1.OperationRunning}}
	view := NewListModel([]operationsv1alpha1.Operation{op}, 100, 20).View()
	for _, wanted := range []string{"Install", "Running", "1"} {
		if !strings.Contains(view, wanted) {
			t.Fatalf("view missing %q: %s", wanted, view)
		}
	}
}

func TestBuildStepEntriesKeepsAttempts(t *testing.T) {
	op := &operationsv1alpha1.Operation{Spec: operationsv1alpha1.OperationSpec{Steps: []operationsv1alpha1.OperationStep{{ID: "step", Targets: []operationsv1alpha1.NodeReference{{Name: "node", UID: "node-uid", IP: "10.0.0.1"}}}}}}
	tasks := []operationsv1alpha1.OperationTask{
		{ObjectMeta: metav1.ObjectMeta{Name: "task-1"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node", UID: "node-uid", IP: "10.0.0.1"}, Attempt: 1}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskSucceeded}},
		{ObjectMeta: metav1.ObjectMeta{Name: "task-0"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node", UID: "node-uid", IP: "10.0.0.1"}, Attempt: 0}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskFailed}},
	}
	entries := buildStepEntries(op, tasks)
	if len(entries) != 1 || len(entries[0].Groups) != 1 || len(entries[0].Groups[0].Tasks) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].Groups[0].IP != "10.0.0.1" {
		t.Fatalf("group ip = %q", entries[0].Groups[0].IP)
	}
	if entries[0].Groups[0].Tasks[0].Attempt != 0 || entries[0].Groups[0].Tasks[1].Attempt != 1 {
		t.Fatalf("attempt order = %#v", entries[0].Groups[0].Tasks)
	}
	if entries[0].Status != string(operationsv1alpha1.TaskSucceeded) {
		t.Fatalf("step status = %q", entries[0].Status)
	}
}

func TestBuildStepEntriesGroupsTasksByIP(t *testing.T) {
	op := &operationsv1alpha1.Operation{Spec: operationsv1alpha1.OperationSpec{Steps: []operationsv1alpha1.OperationStep{{
		ID: "step",
		Targets: []operationsv1alpha1.NodeReference{
			{Name: "node-a", UID: "node-a-uid", IP: "10.0.0.1"},
			{Name: "node-b", UID: "node-b-uid", IP: "10.0.0.2"},
		},
	}}}}
	tasks := []operationsv1alpha1.OperationTask{
		{ObjectMeta: metav1.ObjectMeta{Name: "task-a-0"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node-a", UID: "node-a-uid", IP: "10.0.0.1"}, Attempt: 0}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskSucceeded}},
		{ObjectMeta: metav1.ObjectMeta{Name: "task-a-1"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node-a", UID: "node-a-uid", IP: "10.0.0.1"}, Attempt: 1}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskFailed}},
		{ObjectMeta: metav1.ObjectMeta{Name: "task-b-0"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node-b", UID: "node-b-uid", IP: "10.0.0.2"}, Attempt: 0}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskSucceeded}},
	}
	entries := buildStepEntries(op, tasks)
	if len(entries[0].Groups) != 2 {
		t.Fatalf("groups = %#v", entries[0].Groups)
	}
	if entries[0].Groups[0].IP != "10.0.0.1" || len(entries[0].Groups[0].Tasks) != 2 {
		t.Fatalf("first group = %#v", entries[0].Groups[0])
	}
	if entries[0].Groups[1].IP != "10.0.0.2" || len(entries[0].Groups[1].Tasks) != 1 {
		t.Fatalf("second group = %#v", entries[0].Groups[1])
	}

	view := NewLogModel(nil, op, 100, 20).View()
	if !strings.Contains(view, "10.0.0.1") || !strings.Contains(view, "10.0.0.2") {
		t.Fatalf("view does not show IP groups: %s", view)
	}
	for _, taskName := range []string{"task-a-0", "task-a-1", "task-b-0"} {
		if strings.Contains(view, taskName) {
			t.Fatalf("view exposes task UID/name %q: %s", taskName, view)
		}
	}
}

func TestTaskStatusMarks(t *testing.T) {
	if stepStatusMark(string(operationsv1alpha1.TaskSucceeded)) != StepSuccessMark {
		t.Fatal("Succeeded mark")
	}
	if stepStatusMark(string(operationsv1alpha1.TaskTimedOut)) != StepFailedMark {
		t.Fatal("TimedOut mark")
	}
}

func TestLogModelRestoresTaskLogAfterNavigation(t *testing.T) {
	m := NewLogModel(nil, &operationsv1alpha1.Operation{}, 100, 20)
	m.steps = []StepEntry{
		{ID: "first", Groups: []TaskGroup{{IP: "10.0.0.1", Tasks: []TaskEntry{{Name: "first-task"}}}}},
		{ID: "second", Groups: []TaskGroup{{IP: "10.0.0.2", Tasks: []TaskEntry{{Name: "second-task"}}}}},
	}
	if !m.showCurrentLog() {
		t.Fatal("expected first task to be selected")
	}

	m, _ = m.Update(logFetchedMsg{content: "first log\n", offset: 10, key: "first-task"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(logFetchedMsg{content: "second log\n", offset: 11, key: "second-task"})
	m.viewport.SetContent(strings.Repeat("second log\n", 50))
	m.viewport.GotoBottom()
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})

	if m.rawContent != "first log\n" {
		t.Fatalf("restored log = %q, want first task log", m.rawContent)
	}
	if m.lastOffset["first-task"] != 10 {
		t.Fatalf("first task offset = %d, want 10", m.lastOffset["first-task"])
	}
	if m.viewport.YOffset != 0 {
		t.Fatalf("viewport offset = %d, want top of restored log", m.viewport.YOffset)
	}
}

func TestLogModelIgnoresStaleLogForAnotherTask(t *testing.T) {
	m := NewLogModel(nil, &operationsv1alpha1.Operation{}, 100, 20)
	m.steps = []StepEntry{
		{ID: "first", Groups: []TaskGroup{{IP: "10.0.0.1", Tasks: []TaskEntry{{Name: "first-task"}}}}},
		{ID: "second", Groups: []TaskGroup{{IP: "10.0.0.2", Tasks: []TaskEntry{{Name: "second-task"}}}}},
	}
	if !m.showCurrentLog() {
		t.Fatal("expected first task to be selected")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(logFetchedMsg{content: "first log\n", offset: 10, key: "first-task"})

	if m.rawContent != "" {
		t.Fatalf("displayed log = %q, want empty second task log", m.rawContent)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.rawContent != "first log\n" {
		t.Fatalf("restored log = %q, want stale first task response", m.rawContent)
	}
}

func TestLogModelScrollsStepPanelWithLongTaskList(t *testing.T) {
	op := &operationsv1alpha1.Operation{Spec: operationsv1alpha1.OperationSpec{Steps: make([]operationsv1alpha1.OperationStep, 30)}}
	for i := range op.Spec.Steps {
		op.Spec.Steps[i] = operationsv1alpha1.OperationStep{
			ID:      "step-" + string(rune('a'+i)),
			Targets: []operationsv1alpha1.NodeReference{{Name: "node", UID: "node-uid", IP: "10.0.0.1"}},
		}
	}
	m := NewLogModel(nil, op, 100, 10)
	if m.stepViewport.YOffset != 0 {
		t.Fatalf("initial step offset = %d, want 0", m.stepViewport.YOffset)
	}
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.stepViewport.YOffset == 0 {
		t.Fatal("long step list did not scroll")
	}
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.cursor != 0 || m.stepViewport.YOffset != 0 {
		t.Fatalf("returned to cursor=%d offset=%d, want 0/0", m.cursor, m.stepViewport.YOffset)
	}
}
