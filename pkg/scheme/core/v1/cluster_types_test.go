package v1

import (
	"encoding/json"
	"testing"
)

func TestLegacyRegistryFieldsAreIgnored(t *testing.T) {
	var cluster Cluster
	legacy := `{"localRegistry":"legacy.example.com/k8s","containerRuntime":{"type":"containerd","insecureRegistry":["legacy.example.com"],"registries":[{"insecureRegistry":"other.example.com"}]}}`
	if err := json.Unmarshal([]byte(legacy), &cluster); err != nil {
		t.Fatal(err)
	}
	if cluster.ImageRegistry != "" {
		t.Fatalf("legacy localRegistry unexpectedly populated imageRegistry: %q", cluster.ImageRegistry)
	}
	if len(cluster.ContainerRuntime.Registries) != 1 || cluster.ContainerRuntime.Registries[0].RegistryRef != nil {
		t.Fatalf("legacy insecure registry unexpectedly populated the new registry model: %#v", cluster.ContainerRuntime)
	}
}
