package operationv2

import (
	"bytes"
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	operations "github.com/kubeclipper/kubeclipper/pkg/scheme/operations/v1alpha1"
)

func TestNoopExecutor(t *testing.T) {
	var log bytes.Buffer
	result, err := (NoopExecutor{}).Reconcile(context.Background(), &operations.OperationTask{Spec: operations.OperationTaskSpec{
		Payload: runtime.RawExtension{Raw: []byte(`{"outputs":{"token":"abc"}}`)},
	}}, &log)
	if err != nil || result.Outputs["token"] != "abc" || log.String() == "" {
		t.Fatalf("unexpected noop result: %#v, %v, %q", result, err, log.String())
	}
}
