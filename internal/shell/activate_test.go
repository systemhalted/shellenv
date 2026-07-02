package shell

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivationCodeWithRuntimeBin(t *testing.T) {
	envDir := filepath.Join("/proj", ".shellenv", "default")
	envBin := filepath.Join(envDir, "bin")
	runtimeBin := "/homedir/.shellenv/installs/bash/5.2/bin"

	bash := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{RuntimeBinDir: runtimeBin})
	if !strings.Contains(bash, "export PATH="+envBin+":"+runtimeBin+":$PATH") {
		t.Fatalf("bash activation should order env bin before runtime bin, got: %s", bash)
	}

	fish := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{RuntimeBinDir: runtimeBin})
	if !strings.Contains(fish, "set -gx PATH "+envBin+" "+runtimeBin+" $PATH") {
		t.Fatalf("fish activation should order env bin before runtime bin, got: %s", fish)
	}

	// Without a runtime bin the snippet keeps its original single-dir form.
	plain := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{})
	if !strings.Contains(plain, "export PATH="+envBin+":$PATH") {
		t.Fatalf("activation without runtime bin should only prepend env bin, got: %s", plain)
	}
}

func TestActivationCodeFishSourcesProfile(t *testing.T) {
	envDir := filepath.Join("/proj", ".shellenv", "default")

	fish := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{
		ProfilePath: "/profiles/strict.fish",
	})
	if !strings.Contains(fish, "source /profiles/strict.fish") {
		t.Fatalf("fish activation should source the fish profile, got: %s", fish)
	}

	// Without a profile, no source statement appears.
	plain := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{})
	if strings.Contains(plain, "source ") {
		t.Fatalf("fish activation without profile must not source anything, got: %s", plain)
	}
}

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
