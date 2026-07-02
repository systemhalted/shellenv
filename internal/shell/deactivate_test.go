package shell

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDeactivationCode(t *testing.T) {
	bash := DeactivationCode("bash")
	for _, want := range []string{
		`if [ -n "${SHELLENV_OLD_PATH+x}" ]; then export PATH="$SHELLENV_OLD_PATH"; fi`,
		"SHELLENV_OLD_PS1",
		"SHELLENV_OLD_HOME",
		"SHELLENV_OLD_TMPDIR",
		"SHELLENV_OLD_XDG_CONFIG_HOME",
		"unset SHELLENV_ACTIVE SHELLENV_ENV_NAME",
	} {
		if !strings.Contains(bash, want) {
			t.Fatalf("bash deactivation missing %q, got: %s", want, bash)
		}
	}

	fish := DeactivationCode("fish")
	for _, want := range []string{
		"set -q SHELLENV_OLD_PATH; and set -gx PATH $SHELLENV_OLD_PATH",
		"SHELLENV_OLD_HOME",
		"set -e SHELLENV_ACTIVE",
	} {
		if !strings.Contains(fish, want) {
			t.Fatalf("fish deactivation missing %q, got: %s", want, fish)
		}
	}
}

func TestDeactivationCode_NoopWhenNothingActive(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	// With no SHELLENV_* set, the snippet must be a silent success even
	// under set -eu.
	script := "set -eu\n" + DeactivationCode("bash") + "\necho noop-ok"
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "noop-ok" {
		t.Fatalf("deactivate no-op failed: %v\noutput:\n%s", err, out)
	}
}
