package operation

import (
	"bytes"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	operationsv1alpha1 "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func testOperationAndTasks() (*operationsv1alpha1.Operation, []operationsv1alpha1.OperationTask) {
	started := metav1.NewTime(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	finished := metav1.NewTime(started.Add(2 * time.Minute))
	op := &operationsv1alpha1.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "op-1", UID: "op-uid", CreationTimestamp: started},
		Spec:       operationsv1alpha1.OperationSpec{Action: "InstallCluster", TargetRef: operationsv1alpha1.ObjectReference{Name: "cluster-a"}, Steps: []operationsv1alpha1.OperationStep{{ID: "join", Targets: []operationsv1alpha1.NodeReference{{Name: "node-a", UID: "node-uid"}}}}},
		Status:     operationsv1alpha1.OperationStatus{Phase: operationsv1alpha1.OperationSucceeded},
	}
	tasks := []operationsv1alpha1.OperationTask{
		{ObjectMeta: metav1.ObjectMeta{Name: "attempt-0"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "join", NodeRef: operationsv1alpha1.NodeReference{Name: "node-a", UID: "node-uid"}, Attempt: 0}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskFailed, StartedAt: &started, FinishedAt: &finished}},
		{ObjectMeta: metav1.ObjectMeta{Name: "attempt-1"}, Spec: operationsv1alpha1.OperationTaskSpec{StepID: "join", NodeRef: operationsv1alpha1.NodeReference{Name: "node-a", UID: "node-uid"}, Attempt: 1}, Status: operationsv1alpha1.OperationTaskStatus{Phase: operationsv1alpha1.TaskSucceeded, StartedAt: &started, FinishedAt: &finished}},
	}
	return op, tasks
}

func TestEffectiveTaskPrefersSucceededAttempt(t *testing.T) {
	_, tasks := testOperationAndTasks()
	got := effectiveTask(tasksByExecution(tasks), "join", types.UID("node-uid"))
	if got == nil || got.Name != "attempt-1" {
		t.Fatalf("effective Task = %#v", got)
	}
}

func TestRenderOperationUsesTaskExecutionFacts(t *testing.T) {
	op, tasks := testOperationAndTasks()
	var output bytes.Buffer
	renderOperation(&output, op, tasks)
	for _, wanted := range []string{"op-1", "InstallCluster", "cluster-a", "join", "node-a", "Succeeded"} {
		if !strings.Contains(output.String(), wanted) {
			t.Fatalf("output missing %q:\n%s", wanted, output.String())
		}
	}
}

func TestRenderCancelledOperationMarksUncreatedStepsCancelled(t *testing.T) {
	started := metav1.NewTime(time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC))
	op := &operationsv1alpha1.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "op-cancelled", CreationTimestamp: started},
		Spec: operationsv1alpha1.OperationSpec{Steps: []operationsv1alpha1.OperationStep{{
			ID: "not-started", Targets: []operationsv1alpha1.NodeReference{{Name: "node-a", UID: "node-uid"}},
		}}},
		Status: operationsv1alpha1.OperationStatus{Phase: operationsv1alpha1.OperationCancelled},
	}

	var output bytes.Buffer
	renderOperation(&output, op, nil)
	if !strings.Contains(output.String(), "Step: not-started [Cancelled]") || !strings.Contains(output.String(), "Cancelled node-a") {
		t.Fatalf("cancelled operation rendered uncreated step incorrectly:\n%s", output.String())
	}
}

func TestTasksForStepPreservesAttemptHistory(t *testing.T) {
	_, tasks := testOperationAndTasks()
	got := tasksForStep(tasks, "join")
	if len(got) != 2 || got[0].Spec.Attempt != 0 || got[1].Spec.Attempt != 1 {
		t.Fatalf("Tasks = %#v", got)
	}
}

func TestTruncateLogIsUnicodeSafe(t *testing.T) {
	if got := truncateLog("你好世界", 2); got != "你好"+truncatedSuffix {
		t.Fatalf("got %q", got)
	}
}
