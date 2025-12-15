package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestResolveProfile_EnvOverride(t *testing.T) {
	t.Setenv("SHELLENV_PROFILES", t.TempDir())
	want := writeTmp(t, os.Getenv("SHELLENV_PROFILES"), "custom.sh", "#!/bin/sh\necho env\n")

	got, ok := ResolveProfile("", "custom")
	if !ok {
		t.Fatalf("expected profile to resolve")
	}
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveProfile_ProjectDirectory(t *testing.T) {
	projectDir := t.TempDir()
	want := writeTmp(t, projectDir, "profiles/strict.sh", "#!/bin/sh\n")

	got, ok := ResolveProfile(projectDir, "strict")
	if !ok {
		t.Fatalf("expected profile to resolve")
	}
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestResolveProfile_NotFound(t *testing.T) {
	if _, ok := ResolveProfile("", "missing"); ok {
		t.Fatalf("expected missing profile to return ok=false")
	}
}
