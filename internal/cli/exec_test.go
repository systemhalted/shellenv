package cli

import (
	"errors"
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

func TestExecIsolatesHomeAndTmpdir(t *testing.T) {
	dir := t.TempDir()

	envDir := project.EnvDir(dir, "default")
	binDir := filepath.Join(envDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	// Script records HOME/TMPDIR and writes marker files into each.
	script := filepath.Join(binDir, "probe")
	content := "#!/bin/sh\nprintf \"%s\\n%s\\n\" \"$HOME\" \"$TMPDIR\" > \"$1\"\n: > \"$HOME/marker\"\n: > \"$TMPDIR/tmpmarker\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	output := filepath.Join(dir, "out.txt")

	if _, stderr, err := runCLI(t, dir, "exec", "--", "probe", output); err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected HOME and TMPDIR lines, got %q", data)
	}
	homeSuffix := filepath.Join(".shellenv", "default", "home")
	tmpSuffix := filepath.Join(".shellenv", "default", "home", "tmp")
	if !strings.HasSuffix(lines[0], homeSuffix) {
		t.Fatalf("HOME %q does not point at sandbox home %q", lines[0], homeSuffix)
	}
	if !strings.HasSuffix(lines[1], tmpSuffix) {
		t.Fatalf("TMPDIR %q does not point at sandbox tmp %q", lines[1], tmpSuffix)
	}
	// Writes must land inside the sandbox, not the real home/temp dir.
	sandbox := project.SandboxHomeDir(dir, "default")
	if _, err := os.Stat(filepath.Join(sandbox, "marker")); err != nil {
		t.Fatalf("HOME marker not written to sandbox: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sandbox, "tmp", "tmpmarker")); err != nil {
		t.Fatalf("TMPDIR marker not written to sandbox tmp: %v", err)
	}
}

func TestExecWithProfileSourcesProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	// Project-local profile that exports a sentinel variable.
	profDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profDir, "strict.sh"), []byte("export SE_PROFILE_OK=yes\n"), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2", "--profile", "strict"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	output := filepath.Join(dir, "profile-out.txt")
	cmd := fmt.Sprintf("printf %%s \"$SE_PROFILE_OK\" > %q", output)

	// With --profile the sentinel is sourced and reaches the command.
	if _, stderr, err := runCLI(t, dir, "exec", "--profile", "--", "bash", "-c", cmd); err != nil {
		t.Fatalf("exec --profile returned error: %v (stderr: %s)", err, stderr)
	}
	if got, _ := os.ReadFile(output); string(got) != "yes" {
		t.Fatalf("expected profile sentinel \"yes\", got %q", got)
	}

	// Without --profile the profile is not sourced, so the sentinel is empty.
	_ = os.Remove(output)
	if _, stderr, err := runCLI(t, dir, "exec", "--", "bash", "-c", cmd); err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}
	if got, _ := os.ReadFile(output); string(got) != "" {
		t.Fatalf("expected no profile sentinel without --profile, got %q", got)
	}
}

func TestExecWithProfileAppliesShellOptionsToDirectCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	profDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profDir, "strict.sh"), []byte("set -e\n"), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2", "--profile", "strict"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	// `false` runs directly in the profiled shell, so errexit aborts it.
	_, _, err := runCLI(t, dir, "exec", "--profile", "--", "false")
	if err == nil {
		t.Fatalf("expected non-zero exit from errexit-aborted command, got nil")
	}
}

func TestExecWithProfileErrorsWhenProfileMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	// Env exists but no resolvable profile is available.
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2", "--profile", "strict"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, _, err := runCLI(t, dir, "exec", "--profile", "--", "bash", "-c", "true")
	if err == nil || !strings.Contains(err.Error(), "profile \"strict\" not found") {
		t.Fatalf("expected profile-not-found error, got %v", err)
	}
}

func TestExecPropagatesChildExitCode(t *testing.T) {
	dir := t.TempDir()

	binDir := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	script := filepath.Join(binDir, "boom")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "boom")
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %v", err)
	}
	if ee.code != 3 {
		t.Fatalf("expected exit code 3, got %d", ee.code)
	}
	// The child's failure must not trigger Cobra's usage dump.
	if strings.Contains(stderr, "Usage:") {
		t.Fatalf("did not expect usage output on child exit, got: %s", stderr)
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

// installRuntimeTool places an executable script into the placeholder
// runtime bin dir that `shellenv install` created for shell@version.
func installRuntimeTool(t *testing.T, home, shell, version, name, content string) string {
	t.Helper()
	binDir := filepath.Join(home, "installs", shell, version, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime bin: %v", err)
	}
	tool := filepath.Join(binDir, name)
	if err := os.WriteFile(tool, []byte(content), 0o755); err != nil {
		t.Fatalf("write runtime tool: %v", err)
	}
	return binDir
}

func TestExecPrependsInstalledRuntimeBin(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	output := filepath.Join(dir, "out.txt")
	installRuntimeTool(t, home, "bash", "5.2", "pinnedtool",
		fmt.Sprintf("#!/bin/sh\necho pinned > %q\n", output))
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "pinnedtool")
	if err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}
	if got, _ := os.ReadFile(output); strings.TrimSpace(string(got)) != "pinned" {
		t.Fatalf("expected pinned tool to run, got output %q", got)
	}
	if strings.Contains(stderr, "warning") {
		t.Fatalf("expected no warning for installed runtime, got: %s", stderr)
	}
}

func TestExecRuntimeBinOrderedAfterEnvBin(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	output := filepath.Join(dir, "out.txt")
	installRuntimeTool(t, home, "bash", "5.2", "whoami-tool",
		fmt.Sprintf("#!/bin/sh\necho runtime > %q\n", output))
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	envBin := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if err := os.MkdirAll(envBin, 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}
	script := fmt.Sprintf("#!/bin/sh\necho envbin > %q\n", output)
	if err := os.WriteFile(filepath.Join(envBin, "whoami-tool"), []byte(script), 0o755); err != nil {
		t.Fatalf("write env tool: %v", err)
	}

	if _, stderr, err := runCLI(t, dir, "exec", "--", "whoami-tool"); err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}
	if got, _ := os.ReadFile(output); strings.TrimSpace(string(got)) != "envbin" {
		t.Fatalf("expected env bin to win over runtime bin, got %q", got)
	}
}

func TestExecWarnsWhenDeclaredShellMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@9.9"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	envBin := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if err := os.MkdirAll(envBin, 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}
	output := filepath.Join(dir, "out.txt")
	script := fmt.Sprintf("#!/bin/sh\necho ran > %q\n", output)
	if err := os.WriteFile(filepath.Join(envBin, "tool"), []byte(script), 0o755); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "tool")
	if err != nil {
		t.Fatalf("exec should fall back to system shell, got error: %v", err)
	}
	if got, _ := os.ReadFile(output); strings.TrimSpace(string(got)) != "ran" {
		t.Fatalf("expected command to run despite missing runtime, got %q", got)
	}
	if !strings.Contains(stderr, "bash@9.9") || !strings.Contains(stderr, "not installed") {
		t.Fatalf("expected missing-runtime warning naming bash@9.9, got: %s", stderr)
	}
}

func TestExecStrictShellErrorsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@9.9"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, _, err := runCLI(t, dir, "exec", "--strict-shell", "--", "true")
	if err == nil || !strings.Contains(err.Error(), "bash@9.9") || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected strict-shell missing-runtime error, got %v", err)
	}
	if !strings.Contains(err.Error(), "shellenv install bash@9.9") {
		t.Fatalf("expected actionable install hint in error, got %v", err)
	}
}

func TestExecStrictShellRejectedWithContainer(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, _, err := runCLI(t, dir, "exec", "--strict-shell", "--container", "alpine", "--", "true")
	if err == nil || !strings.Contains(err.Error(), "--strict-shell") || !strings.Contains(err.Error(), "--container") {
		t.Fatalf("expected strict-shell/container conflict error, got %v", err)
	}
}

func TestExecUnversionedShellSkipsResolution(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	envBin := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if err := os.MkdirAll(envBin, 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "true")
	if err != nil {
		t.Fatalf("exec returned error: %v", err)
	}
	if strings.Contains(stderr, "warning") {
		t.Fatalf("expected no warning for unversioned shell, got: %s", stderr)
	}
}

func TestExecIsolatesXDGDirsIntoSandbox(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	envDir := project.EnvDir(dir, "default")
	binDir := filepath.Join(envDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	script := filepath.Join(binDir, "xdgprobe")
	content := "#!/bin/sh\n: > \"$XDG_CONFIG_HOME/cfg\"\n: > \"$XDG_CACHE_HOME/cache\"\n: > \"$XDG_DATA_HOME/data\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if _, stderr, err := runCLI(t, dir, "exec", "--", "xdgprobe"); err != nil {
		t.Fatalf("exec returned error: %v (stderr: %s)", err, stderr)
	}

	sandbox := project.SandboxHomeDir(dir, "default")
	for _, p := range []string{
		filepath.Join(sandbox, ".config", "cfg"),
		filepath.Join(sandbox, ".cache", "cache"),
		filepath.Join(sandbox, ".local", "share", "data"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("XDG write did not land in sandbox at %s: %v", p, err)
		}
	}
}

func TestExecEphemeralUsesThrowawayHomeAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	envDir := project.EnvDir(dir, "default")
	binDir := filepath.Join(envDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	script := filepath.Join(binDir, "probe")
	content := "#!/bin/sh\nprintf %s \"$HOME\" > \"$1\"\n: > \"$HOME/marker\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	output := filepath.Join(dir, "out.txt")

	if _, stderr, err := runCLI(t, dir, "exec", "--ephemeral", "--", "probe", output); err != nil {
		t.Fatalf("exec --ephemeral returned error: %v (stderr: %s)", err, stderr)
	}

	home, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(home), "home-ephemeral") {
		t.Fatalf("expected ephemeral HOME, got %q", home)
	}
	// The throwaway home (and the marker inside it) must be gone afterwards.
	if _, err := os.Stat(string(home)); !os.IsNotExist(err) {
		t.Fatalf("ephemeral home should be removed, stat err=%v", err)
	}
	// The persistent sandbox home must not have been touched.
	if _, err := os.Stat(filepath.Join(project.SandboxHomeDir(dir, "default"), "marker")); !os.IsNotExist(err) {
		t.Fatalf("marker leaked into the persistent sandbox home, stat err=%v", err)
	}
}

func TestExecEphemeralCleansUpOnChildFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	envDir := project.EnvDir(dir, "default")
	binDir := filepath.Join(envDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	script := filepath.Join(binDir, "boom")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 7\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	_, _, err := runCLI(t, dir, "exec", "--ephemeral", "--", "boom")
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 7 {
		t.Fatalf("expected exit code 7 through --ephemeral, got %v", err)
	}

	ents, err := os.ReadDir(envDir)
	if err != nil {
		t.Fatalf("read env dir: %v", err)
	}
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "home-ephemeral") {
			t.Fatalf("ephemeral home %q left behind after child failure", e.Name())
		}
	}
}

func TestExecWarnsOnCorruptMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	envDir := project.EnvDir(dir, "default")
	if err := os.MkdirAll(filepath.Join(envDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "metadata.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write corrupt metadata: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "true")
	if err != nil {
		t.Fatalf("exec should run despite corrupt metadata, got: %v", err)
	}
	if !strings.Contains(stderr, "metadata.json") || !strings.Contains(stderr, "ignoring") {
		t.Fatalf("expected corrupt-metadata warning, got: %s", stderr)
	}
}

func TestExecSilentWhenMetadataMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	envDir := project.EnvDir(dir, "default")
	if err := os.MkdirAll(filepath.Join(envDir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir env bin: %v", err)
	}

	_, stderr, err := runCLI(t, dir, "exec", "--", "true")
	if err != nil {
		t.Fatalf("exec returned error: %v", err)
	}
	if strings.Contains(stderr, "metadata") {
		t.Fatalf("missing metadata should stay silent, got: %s", stderr)
	}
}

func TestExecWithContainerCLI(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2", "--profile", "strict"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	mockBinDir := t.TempDir()
	mockDocker := filepath.Join(mockBinDir, "docker")

	argsOutput := filepath.Join(dir, "docker-args.txt")
	scriptContent := fmt.Sprintf("#!/bin/sh\nfor arg in \"$@\"; do echo \"$arg\" >> %s; done\nexit 0\n", argsOutput)
	if err := os.WriteFile(mockDocker, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", mockBinDir+string(os.PathListSeparator)+oldPath)

	_, _, err := runCLI(t, dir, "exec", "--container", "alpine", "--", "echo", "hello")
	if err != nil {
		t.Fatalf("exec with container failed: %v", err)
	}

	data, err := os.ReadFile(argsOutput)
	if err != nil {
		t.Fatalf("read docker args output: %v", err)
	}

	args := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(args) < 5 {
		t.Fatalf("too few args passed to docker: %q", args)
	}
	if args[0] != "run" {
		t.Errorf("expected 'run', got %s", args[0])
	}

	hasRm := false
	hasVolume := false
	hasWorkdir := false
	hasImage := false
	hasSh := false
	hasEcho := false

	for _, arg := range args {
		if arg == "--rm" {
			hasRm = true
		}
		if strings.HasPrefix(arg, "-v") || strings.Contains(arg, fmt.Sprintf("%s:%s", dir, dir)) {
			hasVolume = true
		}
		if arg == dir {
			hasWorkdir = true
		}
		if arg == "alpine" {
			hasImage = true
		}
		if arg == "sh" {
			hasSh = true
		}
		if arg == "echo" {
			hasEcho = true
		}
	}

	if !hasRm {
		t.Errorf("missing --rm flag")
	}
	if !hasVolume {
		t.Errorf("missing volume mount for %s", dir)
	}
	if !hasWorkdir {
		t.Errorf("missing workdir or volume directory reference")
	}
	if !hasImage {
		t.Errorf("missing container image 'alpine'")
	}
	if !hasSh {
		t.Errorf("missing sh entry point wrapping")
	}
	if !hasEcho {
		t.Errorf("missing execution command 'echo'")
	}
}

func TestExecWithContainerExitCode(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2", "--profile", "strict"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	mockBinDir := t.TempDir()
	mockDocker := filepath.Join(mockBinDir, "docker")

	scriptContent := "#!/bin/sh\nexit 123\n"
	if err := os.WriteFile(mockDocker, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("write mock docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	defer os.Setenv("PATH", oldPath)
	os.Setenv("PATH", mockBinDir+string(os.PathListSeparator)+oldPath)

	_, _, err := runCLI(t, dir, "exec", "--container", "alpine", "--", "echo")
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exitError, got %v", err)
	}
	if ee.code != 123 {
		t.Fatalf("expected exit code 123, got %d", ee.code)
	}
}
