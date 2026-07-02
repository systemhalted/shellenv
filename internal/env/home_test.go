package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHomeResolutionOverride(t *testing.T) {
	t.Setenv("SHELLENV_HOME", "/tmp/shellenv-home-test")
	h, err := Home()
	if err != nil {
		t.Fatal(err)
	}
	if h != "/tmp/shellenv-home-test" {
		t.Fatalf("expected override, got %s", h)
	}
}

func TestEnsureHomeCreatesSubdirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SHELLENV_HOME", tmp)
	h, err := EnsureHome()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"installs", "cache", "tmp"} {
		if _, err := os.Stat(filepath.Join(h, d)); err != nil {
			t.Fatalf("missing subdir %s: %v", d, err)
		}
	}
	// shims/ was removed (R4): per-env PATH pinning supersedes global shims.
	if _, err := os.Stat(filepath.Join(h, "shims")); !os.IsNotExist(err) {
		t.Fatalf("shims subdir should no longer be created, stat err=%v", err)
	}
}
