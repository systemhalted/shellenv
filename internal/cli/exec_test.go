package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/systemhalted/shellenv/internal/project"
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

func TestExecRequiresDashSeparator(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "exec", "no-dash")
	if err == nil || !strings.Contains(err.Error(), "usage: shellenv exec") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestExecRunsCommandWithEnvVars(t *testing.T) {
	dir := t.TempDir()

	envDir := project.EnvDir(dir, "default")
	binDir := filepath.Join(envDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	script := filepath.Join(binDir, "dump")
	content := "#!/bin/sh\nprintf \"%s\\n%s\" \"$SHELLENV_ENV_NAME\" \"$PATH\" > \"$1\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	output := filepath.Join(dir, "out.txt")

	_, stderr, err := runCLI(t, dir, "exec", "--", "dump", output)
	if err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %q", data)
	}
	if lines[0] != "default" {
		t.Fatalf("expected env name default, got %s", lines[0])
	}
	resolvedBin := binDir
	if r, err := filepath.EvalSymlinks(binDir); err == nil {
		resolvedBin = r
	}
	if !strings.HasPrefix(lines[1], binDir) && !strings.HasPrefix(lines[1], resolvedBin) {
		t.Fatalf("expected PATH to start with env bin dir, got %s", lines[1])
	}
}

func TestExecErrorsWhenCommandMissing(t *testing.T) {
	dir := t.TempDir()

	envDir := project.EnvDir(dir, "default")
	if err := os.MkdirAll(filepath.Join(envDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}

	_, _, err := runCLI(t, dir, "exec", "--", "does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "command \"does-not-exist\" not found") {
		t.Fatalf("expected missing command error, got %v", err)
	}
}
