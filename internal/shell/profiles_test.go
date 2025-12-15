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

func TestResolveProfile_PrefersEnvOverride(t *testing.T) {
	// Arrange: three possible locations with the same profile name
	tmp := t.TempDir()
	envDir := filepath.Join(tmp, "envprofiles")
	projectDir := filepath.Join(tmp, "project")
	binDir := filepath.Join(tmp, "bin") // we won't rely on os.Executable() in this test
	_ = os.MkdirAll(envDir, 0o755)
	_ = os.MkdirAll(_
