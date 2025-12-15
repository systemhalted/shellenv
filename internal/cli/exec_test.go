package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCommandPathPrefersEnvBin(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	target := filepath.Join(binDir, "hi")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}

	env := []string{fmt.Sprintf("PATH=%s", binDir)}
	got, err := resolveCommandPath("hi", env)
	if err != nil {
		t.Fatalf("resolveCommandPath returned error: %v", err)
	}
	if got != target {
		t.Fatalf("expected %s, got %s", target, got)
	}
}
