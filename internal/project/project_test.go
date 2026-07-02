package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndReadMetadata(t *testing.T) {
	cwd := t.TempDir()
	md := Metadata{Name: "default", Shell: "bash@5.2", Profile: "strict"}
	if err := WriteMetadata(cwd, md); err != nil {
		t.Fatal(err)
	}

	got, err := ReadMetadata(cwd, "default")
	if err != nil {
		t.Fatal(err)
	}
	if got.Shell != "bash@5.2" {
		t.Fatalf("want bash@5.2, got %s", got.Shell)
	}

	if _, err := os.Stat(filepath.Join(EnvDir(cwd, "default"), "bin")); err != nil {
		t.Fatalf("bin dir missing: %v", err)
	}
}

func TestSandboxLayoutAndEnsureDirs(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")

	sb := SandboxLayout(home)
	if sb.Home != home ||
		sb.Tmp != filepath.Join(home, "tmp") ||
		sb.XDGConfig != filepath.Join(home, ".config") ||
		sb.XDGCache != filepath.Join(home, ".cache") ||
		sb.XDGData != filepath.Join(home, ".local", "share") {
		t.Fatalf("unexpected layout: %+v", sb)
	}

	got, err := EnsureSandboxDirs(home)
	if err != nil {
		t.Fatal(err)
	}
	if got != sb {
		t.Fatalf("EnsureSandboxDirs = %+v, want %+v", got, sb)
	}
	for _, d := range []string{sb.Home, sb.Tmp, sb.XDGConfig, sb.XDGCache, sb.XDGData} {
		if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
			t.Fatalf("dir %s not created: %v", d, err)
		}
	}
}

func TestListEnvs(t *testing.T) {
	cwd := t.TempDir()
	// empty is fine
	names, err := ListEnvs(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("expected no envs, got %v", names)
	}

	_ = os.MkdirAll(EnvDir(cwd, "alpha"), 0o755)
	_ = os.MkdirAll(EnvDir(cwd, "beta"), 0o755)

	names, err = ListEnvs(cwd)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(names, ",")
	if got != "alpha,beta" {
		t.Fatalf("expected alpha,beta got %s", got)
	}
}
