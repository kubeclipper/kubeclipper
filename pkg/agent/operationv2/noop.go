package operationv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

const (
	NoopExecutorName = "Noop/v1"
	maxNoopOutputs   = 32
)

type NoopPayload struct {
	Outputs map[string]string `json:"outputs,omitempty"`
}

// NoopExecutor is a deterministic executor used for smoke tests and for
// validating the complete List/Watch/Task status path without node mutation.
type NoopExecutor struct{}

func (NoopExecutor) Reconcile(_ context.Context, task *operations.OperationTask, log io.Writer) (operations.TaskResult, error) {
	var payload NoopPayload
	decoder := json.NewDecoder(bytes.NewReader(task.Spec.Payload.Raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return operations.TaskResult{}, fmt.Errorf("decode Noop/v1 payload: %w", err)
	}
	if err := validateOutputs(payload.Outputs); err != nil {
		return operations.TaskResult{}, err
	}
	if _, err := io.WriteString(log, "noop completed\n"); err != nil {
		return operations.TaskResult{}, fmt.Errorf("write noop log: %w", err)
	}
	return operations.TaskResult{Outputs: payload.Outputs}, nil
}

func validateOutputs(outputs map[string]string) error {
	if len(outputs) > maxNoopOutputs {
		return fmt.Errorf("noop outputs exceed limit")
	}
	for key, value := range outputs {
		if key == "" || len(key) > operations.MaxOutputKeySize || len(value) > operations.MaxOutputValueSize {
			return fmt.Errorf("noop output is too large or has an empty key")
		}
	}
	return nil
}
