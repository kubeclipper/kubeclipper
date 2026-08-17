package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStagePackageKeepsSourceIndependentFromCleanupDirectory(t *testing.T) {
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "kc-amd64.tar.gz")
	contents := []byte("package contents")
	if err := os.WriteFile(source, contents, 0600); err != nil {
		t.Fatalf("write source package: %v", err)
	}

	tempDir := t.TempDir()
	staged, cleanup, err := stagePackage(source, tempDir)
	if err != nil {
		t.Fatalf("stage package: %v", err)
	}
	defer cleanup()
	if staged == source {
		t.Fatalf("staged package reused source path %q", source)
	}
	wantPrefix := filepath.Join(tempDir, "kc-source") + string(filepath.Separator)
	if !strings.HasPrefix(staged, wantPrefix) {
		t.Fatalf("staged package path = %q, want under %q", staged, wantPrefix)
	}

	if removeErr := os.Remove(source); removeErr != nil {
		t.Fatalf("remove source package: %v", removeErr)
	}
	got, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("read staged package after source removal: %v", err)
	}
	if !bytes.Equal(got, contents) {
		t.Fatalf("staged package contents = %q, want %q", got, contents)
	}
}
