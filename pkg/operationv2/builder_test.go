package operationv2

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubeclipper/kubeclipper/pkg/scheme/common"
	corev1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

type fakeNodes map[string]*corev1.Node

func (f fakeNodes) GetNodeEx(_ context.Context, name, _ string) (*corev1.Node, error) {
	return f[name], nil
}

func TestFromCoreOperationBuildsUIDBoundOrderedPlan(t *testing.T) {
	plan := &corev1.Operation{
		ObjectMeta: metav1.ObjectMeta{Name: "op-a", Labels: map[string]string{
			common.LabelOperationAction: corev1.OperationCreateCluster,
			common.LabelTimeoutSeconds:  "600",
		}},
		Steps: []corev1.Step{
			{ID: "init", Nodes: []corev1.StepNode{{ID: "node-a"}}, RetryTimes: 1},
			{ID: "join", Nodes: []corev1.StepNode{{ID: "node-b"}}},
		},
	}
	cluster := &corev1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster-a", UID: types.UID("cluster-uid")}}
	nodes := fakeNodes{
		"node-a": {ObjectMeta: metav1.ObjectMeta{Name: "node-a", UID: types.UID("node-a-uid")}},
		"node-b": {ObjectMeta: metav1.ObjectMeta{Name: "node-b", UID: types.UID("node-b-uid")}},
	}

	result, err := FromCoreOperation(context.Background(), plan, cluster, nodes)
	if err != nil {
		t.Fatal(err)
	}
	if result.Spec.TargetRef.UID != cluster.UID || result.Spec.Steps[0].Targets[0].UID != nodes["node-a"].UID {
		t.Fatalf("target identities were not bound: %#v", result.Spec)
	}
	if result.Spec.Steps[0].RetryLimit != 1 {
		t.Fatalf("retryLimit = %d, want 1", result.Spec.Steps[0].RetryLimit)
	}
	input := result.Spec.Steps[1].Inputs[0]
	if input.FromStepID != "init" || input.FromNodeUID != nodes["node-a"].UID || input.OutputKey != "response" {
		t.Fatalf("unexpected cross-step input: %#v", input)
	}
}
