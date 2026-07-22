package create

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
	v1 "github.com/kubeclipper/kubeclipper/pkg/scheme/core/v1"
)

func TestCreateClusterRegistryFlags(t *testing.T) {
	out := &bytes.Buffer{}
	streams := options.IOStreams{In: strings.NewReader(""), Out: out, ErrOut: out}
	cmd := NewCmdCreateCluster(streams)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	if !strings.Contains(help, "--image-repository string") || !strings.Contains(help, `(default "registry.k8s.io")`) {
		t.Fatalf("image repository flag or default missing from help:\n%s", help)
	}
	if strings.Contains(help, "--local-registry") || strings.Contains(help, "--insecure-registry") {
		t.Fatalf("removed registry flags remain in help:\n%s", help)
	}
}

func TestNewClusterUsesImageRepository(t *testing.T) {
	opts := NewCreateClusterOptions(options.IOStreams{})
	opts.Name = "demo"
	opts.Masters = []string{"node-1"}
	cluster := opts.newCluster()
	if cluster.ImageRepository != v1.DefaultImageRepository {
		t.Fatalf("imageRepository = %q, want %q", cluster.ImageRepository, v1.DefaultImageRepository)
	}
}
