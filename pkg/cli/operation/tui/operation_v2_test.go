package tui

import (
	"strings"
	"testing"

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
	op := &operationsv1alpha1.Operation{Spec: operationsv1alpha1.OperationSpec{Steps: []operationsv1alpha1.OperationStep{{ID: "step", Targets: []operationsv1alpha1.NodeReference{{Name: "node", UID: "node-uid"}}}}}}
	tasks := []operationsv1alpha1.OperationTask{
		{ObjectMeta: metav1.ObjectMeta{Name: "task-1"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node", UID: "node-uid"}, Attempt: 1}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskSucceeded}},
		{ObjectMeta: metav1.ObjectMeta{Name: "task-0"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "step", NodeRef: operationsv1alpha1.NodeReference{Name: "node", UID: "node-uid"}, Attempt: 0}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskFailed}},
	}
	entries := buildStepEntries(op, tasks)
	if len(entries) != 1 || len(entries[0].Tasks) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].Tasks[0].Attempt != 0 || entries[0].Tasks[1].Attempt != 1 {
		t.Fatalf("attempt order = %#v", entries[0].Tasks)
	}
	if entries[0].Status != string(operationsv1alpha1.TaskSucceeded) {
		t.Fatalf("step status = %q", entries[0].Status)
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
