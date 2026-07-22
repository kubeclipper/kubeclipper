package create

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kubeclipper/kubeclipper/cmd/kcctl/app/options"
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
	if !strings.Contains(help, "--image-registry string") {
		t.Fatalf("image registry flag missing from help:\n%s", help)
	}
	if strings.Contains(help, "--local-registry") || strings.Contains(help, "--insecure-registry") {
		t.Fatalf("removed registry flags remain in help:\n%s", help)
	}
}

func TestNewClusterLeavesOnlineImageRegistryEmpty(t *testing.T) {
	opts := NewCreateClusterOptions(options.IOStreams{})
	opts.Name = "demo"
	opts.Masters = []string{"node-1"}
	cluster := opts.newCluster()
	if cluster.ImageRegistry != "" {
		t.Fatalf("imageRegistry = %q, want empty", cluster.ImageRegistry)
	}
}
