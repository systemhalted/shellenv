package shell

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/systemhalted/shellenv/internal/project"
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

func TestActivationCodeSavesPathGuarded(t *testing.T) {
	envDir := filepath.Join("/proj", ".shellenv", "default")

	bash := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{})
	if !strings.Contains(bash, `[ -n "${SHELLENV_OLD_PATH+x}" ] || export SHELLENV_OLD_PATH="$PATH"`) {
		t.Fatalf("bash activation must save PATH write-once, got: %s", bash)
	}
	if !strings.Contains(bash, `[ -n "${SHELLENV_OLD_PS1+x}" ] || export SHELLENV_OLD_PS1="${PS1-}"`) {
		t.Fatalf("bash activation must save PS1 write-once, got: %s", bash)
	}

	fish := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{})
	if !strings.Contains(fish, "set -q SHELLENV_OLD_PATH; or set -gx SHELLENV_OLD_PATH $PATH") {
		t.Fatalf("fish activation must save PATH write-once, got: %s", fish)
	}
}

func TestActivationCodeIsolateHome(t *testing.T) {
	envDir := filepath.Join("/proj", ".shellenv", "default")
	sb := project.SandboxLayout(filepath.Join(envDir, "home"))

	bash := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{Sandbox: &sb})
	for _, want := range []string{
		`[ -n "${SHELLENV_OLD_HOME+x}" ] || export SHELLENV_OLD_HOME="${HOME-}"`,
		"export HOME=" + sb.Home,
		"export TMPDIR=" + sb.Tmp,
		"export XDG_CONFIG_HOME=" + sb.XDGConfig,
		"export XDG_CACHE_HOME=" + sb.XDGCache,
		"export XDG_DATA_HOME=" + sb.XDGData,
	} {
		if !strings.Contains(bash, want) {
			t.Fatalf("bash isolate-home activation missing %q, got: %s", want, bash)
		}
	}
	// The save must precede the override.
	if strings.Index(bash, "SHELLENV_OLD_HOME") > strings.Index(bash, "export HOME=") {
		t.Fatalf("HOME must be saved before it is overridden: %s", bash)
	}

	fish := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{Sandbox: &sb})
	for _, want := range []string{
		"set -q SHELLENV_OLD_HOME; or set -gx SHELLENV_OLD_HOME \"$HOME\"",
		"set -gx HOME " + sb.Home,
		"set -gx XDG_DATA_HOME " + sb.XDGData,
	} {
		if !strings.Contains(fish, want) {
			t.Fatalf("fish isolate-home activation missing %q, got: %s", want, fish)
		}
	}

	// Without Sandbox, no HOME override is emitted.
	plain := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{})
	if strings.Contains(plain, "export HOME=") || strings.Contains(plain, "TMPDIR=") {
		t.Fatalf("plain activation must not touch HOME/TMPDIR, got: %s", plain)
	}
}

func TestActivateDeactivateRoundTrip_Bash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	envDir := filepath.Join(t.TempDir(), ".shellenv", "default")
	sb := project.SandboxLayout(filepath.Join(envDir, "home"))

	activate := ActivationCodeWithOptions("bash", envDir, "default", ActivationOptions{Sandbox: &sb})
	deactivate := DeactivationCode("bash")

	script := "set -u\n" +
		"orig_path=\"$PATH\"; orig_home=\"$HOME\"\n" +
		activate + "\n" +
		"[ \"$HOME\" = " + sb.Home + " ] || { echo 'HOME not sandboxed'; exit 1; }\n" +
		deactivate + "\n" +
		"[ \"$PATH\" = \"$orig_path\" ] || { echo 'PATH not restored'; exit 1; }\n" +
		"[ \"$HOME\" = \"$orig_home\" ] || { echo 'HOME not restored'; exit 1; }\n" +
		"env | grep '^SHELLENV_' && { echo 'SHELLENV vars leaked'; exit 1; }\n" +
		"echo round-trip-ok"
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "round-trip-ok") {
		t.Fatalf("round trip failed: %v\noutput:\n%s", err, out)
	}
}

func TestActivateDeactivateRoundTrip_Fish(t *testing.T) {
	if _, err := exec.LookPath("fish"); err != nil {
		t.Skip("fish not available")
	}
	envDir := filepath.Join(t.TempDir(), ".shellenv", "default")
	sb := project.SandboxLayout(filepath.Join(envDir, "home"))

	activate := ActivationCodeWithOptions("fish", envDir, "default", ActivationOptions{Sandbox: &sb})
	deactivate := DeactivationCode("fish")

	script := "set orig_home $HOME\n" +
		activate + "\n" +
		"test \"$HOME\" = " + sb.Home + "; or begin; echo 'HOME not sandboxed'; exit 1; end\n" +
		deactivate + "\n" +
		"test \"$HOME\" = \"$orig_home\"; or begin; echo 'HOME not restored'; exit 1; end\n" +
		"set -q SHELLENV_ACTIVE; and begin; echo 'SHELLENV_ACTIVE leaked'; exit 1; end\n" +
		"echo round-trip-ok"
	out, err := exec.Command("fish", "-c", script).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "round-trip-ok") {
		t.Fatalf("fish round trip failed: %v\noutput:\n%s", err, out)
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
