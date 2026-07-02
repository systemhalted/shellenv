package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseShellVersion(t *testing.T) {
	cases := []struct {
		in      string
		shell   string
		version string
		ok      bool
	}{
		{"bash@5.2", "bash", "5.2", true},
		{"zsh@5.9", "zsh", "5.9", true},
		{"bash", "", "", false},
		{"", "", "", false},
		{"zsh@", "", "", false},
		{"@5.2", "", "", false},
	}
	for _, c := range cases {
		shell, version, ok := ParseShellVersion(c.in)
		if shell != c.shell || version != c.version || ok != c.ok {
			t.Errorf("ParseShellVersion(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.in, shell, version, ok, c.shell, c.version, c.ok)
		}
	}
}

func TestResolveRuntimeStatuses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	// Unpinned: no @version declared.
	if _, status, err := ResolveRuntime("bash"); err != nil || status != RuntimeUnpinned {
		t.Fatalf("ResolveRuntime(\"bash\") = status %v, err %v; want RuntimeUnpinned, nil", status, err)
	}
	if _, status, err := ResolveRuntime(""); err != nil || status != RuntimeUnpinned {
		t.Fatalf("ResolveRuntime(\"\") = status %v, err %v; want RuntimeUnpinned, nil", status, err)
	}

	// Missing: declared but never installed.
	if _, status, err := ResolveRuntime("bash@5.2"); err != nil || status != RuntimeMissing {
		t.Fatalf("ResolveRuntime uninstalled = status %v, err %v; want RuntimeMissing, nil", status, err)
	}

	// Missing: version dir exists but has no bin/.
	if err := os.MkdirAll(filepath.Join(home, "installs", "zsh", "5.9"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := ResolveRuntime("zsh@5.9"); status != RuntimeMissing {
		t.Fatalf("ResolveRuntime without bin dir = status %v; want RuntimeMissing", status)
	}

	// Found: bin dir exists; binDir points at it.
	binDir := filepath.Join(home, "installs", "bash", "5.2", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	got, status, err := ResolveRuntime("bash@5.2")
	if err != nil || status != RuntimeFound || got != binDir {
		t.Fatalf("ResolveRuntime installed = (%q, %v, %v); want (%q, RuntimeFound, nil)", got, status, err, binDir)
	}
}
