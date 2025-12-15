package shell

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivationCode_AllowsUnsetPS1(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	envDir := filepath.Join(t.TempDir(), ".shellenv", "default")
	code := ActivationCodeWithProfile("bash", envDir, "default", "")

	cmd := exec.Command("bash", "-c", "set -eu\n"+code+"\necho \"$SHELLENV_ENV_NAME\"")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("activation snippet failed: %v\noutput:\n%s", err, out)
	}
	got := strings.TrimSpace(string(out))
	if got != "default" {
		t.Fatalf("expected env name default, got %q (output: %s)", got, out)
	}
}
