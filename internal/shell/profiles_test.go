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

func TestResolveProfileForShell_FishVariant(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SHELLENV_PROFILES", base)
	fishProfile := writeTmp(t, base, "strict.fish", "# fish strict\n")
	shProfile := writeTmp(t, base, "strict.sh", "set -euo pipefail\n")

	got, ok := ResolveProfileForShell("", "strict", "fish")
	if !ok || got != fishProfile {
		t.Fatalf("fish should resolve the .fish variant, got (%q, %v)", got, ok)
	}

	got, ok = ResolveProfileForShell("", "strict", "bash")
	if !ok || got != shProfile {
		t.Fatalf("bash should resolve the .sh variant, got (%q, %v)", got, ok)
	}
}

func TestResolveProfileForShell_FishWithoutFishVariant(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SHELLENV_PROFILES", base)
	writeTmp(t, base, "strict.sh", "set -euo pipefail\n")

	// fish cannot source POSIX profiles, so a missing .fish variant must
	// report not-found rather than fall back to the .sh file.
	if got, ok := ResolveProfileForShell("", "strict", "fish"); ok {
		t.Fatalf("fish must not fall back to the .sh profile, got %q", got)
	}
}
