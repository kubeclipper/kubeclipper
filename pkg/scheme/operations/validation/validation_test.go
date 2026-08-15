package validation

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func validOperation() *operations.Operation {
	return &operations.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "op-1"},
		Spec: operations.OperationSpec{
			TargetRef:    operations.ObjectReference{Kind: "Cluster", Name: "c1", UID: types.UID("cluster-1")},
			Action:       "Install",
			DesiredState: operations.OperationDesiredStateActive,
			Timeout:      metav1.Duration{Duration: time.Hour},
			Steps: []operations.OperationStep{{
				ID:       "step-1",
				Executor: "noop",
				Targets:  []operations.NodeReference{{Name: "node-1", UID: types.UID("node-1")}},
				Payload:  runtime.RawExtension{Raw: []byte(`{"ok":true}`)},
			}},
		},
	}
}

func TestValidateOperation(t *testing.T) {
	if errs := ValidateOperation(validOperation()); len(errs) != 0 {
		t.Fatalf("valid operation rejected: %v", errs)
	}
	op := validOperation()
	op.Spec.Steps[0].Inputs = []operations.StepInput{{FromStepID: "step-1", FromNodeUID: types.UID("node-1"), Field: "x", OutputKey: "y"}}
	if errs := ValidateOperation(op); len(errs) == 0 {
		t.Fatal("same-step input was accepted")
	}
}

func TestTerminalTransitionsAreImmutable(t *testing.T) {
	if err := ValidateTaskPhaseTransition(operations.TaskSucceeded, operations.TaskFailed); err == nil {
		t.Fatal("terminal task transition was accepted")
	}
	if err := ValidateOperationPhaseTransition(operations.OperationFailed, operations.OperationRunning); err == nil {
		t.Fatal("terminal operation transition was accepted")
	}
}
