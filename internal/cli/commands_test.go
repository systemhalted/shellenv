package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/systemhalted/shellenv/internal/project"
)

func resetCLIState() {
	createName = ""
	createShell = ""
	createWithTools = false
	createProfile = "strict"
	actShellType = ""
	actStrictShell = false
	actIsolateHome = false
	deactShellType = ""
	execWithProfile = false
	execContainer = ""
	execStrictShell = false
	execEphemeral = false
}

func runCLI(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()
	resetCLIState()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir %s: %v", dir, err)
		}
		defer func() {
			_ = os.Chdir(oldwd)
		}()
	}

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr
	outDone := make(chan struct{})
	errDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(stdoutBuf, rOut)
		close(outDone)
	}()
	go func() {
		_, _ = io.Copy(stderrBuf, rErr)
		close(errDone)
	}()

	execErr := ExecuteWithArgs(args, stdoutBuf, stderrBuf)

	_ = wOut.Close()
	_ = wErr.Close()
	<-outDone
	<-errDone
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdoutBuf.String(), stderrBuf.String(), execErr
}

func TestVersionFlag(t *testing.T) {
	// Note: cobra prints the version through the writer set by SetOut, which
	// runCLI's pipe-copier can clobber (see the R3 harness note), so assert
	// on the wiring rather than captured stdout.
	if rootCmd.Version != version {
		t.Fatalf("rootCmd.Version = %q, want %q", rootCmd.Version, version)
	}
	if version == "" {
		t.Fatalf("version must never be empty (cobra would drop the --version flag)")
	}
	if _, _, err := runCLI(t, "", "--version"); err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
}

func TestInitCommandCreatesMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	stdout, stderr, err := runCLI(t, "", "init")
	if err != nil {
		t.Fatalf("init returned error: %v (stderr: %s)", err, stderr)
	}

	marker := filepath.Join(home, ".initialized")
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected initialization marker: %v", err)
	}
	if string(data) != "ok\n" {
		t.Fatalf("marker contents = %q", data)
	}
	if !strings.Contains(stdout, "Initialized shellenv at "+home) {
		t.Fatalf("unexpected init output: %s", stdout)
	}
	// The old shims PATH hint is gone (R4): init must not tell users to
	// edit their shell rc file.
	if strings.Contains(stdout, "shell rc file") || strings.Contains(stdout, "shims") {
		t.Fatalf("init should no longer print the shims PATH hint, got: %s", stdout)
	}
}

func TestInstallValidatesInput(t *testing.T) {
	t.Setenv("SHELLENV_HOME", t.TempDir())

	_, _, err := runCLI(t, "", "install", "bash")
	if err == nil || !strings.Contains(err.Error(), "expected <shell>@<version>") {
		t.Fatalf("expected format error, got %v", err)
	}
}

func TestInstallRejectsUnsupportedShell(t *testing.T) {
	t.Setenv("SHELLENV_HOME", t.TempDir())

	_, _, err := runCLI(t, "", "install", "fish@3.7")
	if err == nil || !strings.Contains(err.Error(), "no installer for shell") {
		t.Fatalf("expected unsupported-shell error, got %v", err)
	}
}

func TestInstallSkipsAlreadyInstalledRuntime(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	// A runtime whose bin/<shell> exists is treated as installed; no
	// download or build is attempted (so this test needs no network).
	binDir := filepath.Join(home, "installs", "bash", "5.2", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "bash"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake bash: %v", err)
	}

	stdout, stderr, err := runCLI(t, "", "install", "bash@5.2")
	if err != nil {
		t.Fatalf("install returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "already installed") {
		t.Fatalf("expected already-installed message, got: %s", stdout)
	}
}

func TestUninstallValidatesInput(t *testing.T) {
	t.Setenv("SHELLENV_HOME", t.TempDir())

	_, _, err := runCLI(t, "", "uninstall", "zsh")
	if err == nil || !strings.Contains(err.Error(), "expected <shell>@<version>") {
		t.Fatalf("expected format error, got %v", err)
	}
}

func TestUninstallRemovesRuntime(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	target := filepath.Join(home, "installs", "zsh", "5.9")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "installed.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	stdout, stderr, err := runCLI(t, "", "uninstall", "zsh@5.9")
	if err != nil {
		t.Fatalf("uninstall returned error: %v (stderr: %s)", err, stderr)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected runtime to be removed, stat err=%v", err)
	}
	if !strings.Contains(stdout, "Uninstalled zsh@5.9") {
		t.Fatalf("unexpected uninstall output: %s", stdout)
	}
}

func TestVersionsShowsMessageWhenEmpty(t *testing.T) {
	t.Setenv("SHELLENV_HOME", t.TempDir())

	stdout, stderr, err := runCLI(t, "", "versions")
	if err != nil {
		t.Fatalf("versions returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "No runtimes installed yet") {
		t.Fatalf("unexpected versions output: %s", stdout)
	}
}

func TestVersionsListsInstalledRuntimes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	for _, p := range []string{
		filepath.Join(home, "installs", "bash", "5.2", "bin"),
		filepath.Join(home, "installs", "zsh", "5.9", "bin"),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "installs", "bash", "note.txt"), []byte("skip\n"), 0o644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	stdout, stderr, err := runCLI(t, "", "versions")
	if err != nil {
		t.Fatalf("versions returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "bash@5.2") || !strings.Contains(stdout, "zsh@5.9") {
		t.Fatalf("unexpected versions output: %s", stdout)
	}
	if strings.Contains(stdout, "note.txt") {
		t.Fatalf("versions should ignore non-version entries: %s", stdout)
	}
}

func TestCreateRequiresShellFlag(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "create")
	if err == nil || !strings.Contains(err.Error(), "--shell is required") {
		t.Fatalf("expected missing shell error, got %v", err)
	}
}

func TestCreateWritesMetadataAndActivate(t *testing.T) {
	dir := t.TempDir()

	stdout, stderr, err := runCLI(t, dir, "create", "--name", "demo", "--shell", "bash@5.2", "--profile", "interactive", "--with-tools")
	if err != nil {
		t.Fatalf("create returned error: %v (stderr: %s)", err, stderr)
	}

	md, err := project.ReadMetadata(dir, "demo")
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	if md.Name != "demo" || md.Shell != "bash@5.2" || md.Profile != "interactive" {
		t.Fatalf("unexpected metadata: %+v", md)
	}
	if len(md.Tools) != 1 || md.Tools[0] != "busybox@placeholder" {
		t.Fatalf("expected placeholder tools, got %v", md.Tools)
	}
	if md.Created == "" {
		t.Fatalf("expected Created timestamp to be set")
	}

	activatePath := filepath.Join(project.EnvDir(dir, "demo"), "activate.sh")
	data, err := os.ReadFile(activatePath)
	if err != nil {
		t.Fatalf("activate script missing: %v", err)
	}
	if !strings.Contains(string(data), "shellenv activate") {
		t.Fatalf("unexpected activate content: %q", data)
	}
	if fi, err := os.Stat(activatePath); err != nil || fi.Mode()&0o111 == 0 {
		t.Fatalf("activate script should be executable, err=%v mode=%v", err, fi.Mode())
	}
	if !strings.Contains(stdout, "demo") {
		t.Fatalf("expected env name in output, got: %s", stdout)
	}
}

func TestUseValidatesEnvPresence(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "use", "missing")
	if err == nil || !strings.Contains(err.Error(), "env \"missing\" not found") {
		t.Fatalf("expected missing env error, got %v", err)
	}
}

func TestUseWritesCurrentEnv(t *testing.T) {
	dir := t.TempDir()

	envDir := project.EnvDir(dir, "work")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir env: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "use", "work")
	if err != nil {
		t.Fatalf("use returned error: %v (stderr: %s)", err, stderr)
	}

	currentPath := filepath.Join(dir, ".shellenv", "current")
	data, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("current marker missing: %v", err)
	}
	if string(data) != "work\n" {
		t.Fatalf("unexpected current contents: %q", data)
	}
	if !strings.Contains(stdout, "Now using env") {
		t.Fatalf("unexpected use output: %s", stdout)
	}
}

func TestDestroyMissingEnv(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "destroy", "ghost")
	if err == nil || !strings.Contains(err.Error(), "env \"ghost\" not found") {
		t.Fatalf("expected missing env error, got %v", err)
	}
}

func TestDestroyRemovesEnv(t *testing.T) {
	dir := t.TempDir()

	envDir := project.EnvDir(dir, "old")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "metadata.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write metadata stub: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "destroy", "old")
	if err != nil {
		t.Fatalf("destroy returned error: %v (stderr: %s)", err, stderr)
	}
	if _, err := os.Stat(envDir); !os.IsNotExist(err) {
		t.Fatalf("expected env dir removed, stat err=%v", err)
	}
	if !strings.Contains(stdout, "Destroyed env") {
		t.Fatalf("unexpected destroy output: %s", stdout)
	}
}

func TestListPrintsEnvNames(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"bravo", "alpha"} {
		if err := os.MkdirAll(project.EnvDir(dir, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	stdout, stderr, err := runCLI(t, dir, "list")
	if err != nil {
		t.Fatalf("list returned error: %v (stderr: %s)", err, stderr)
	}
	lines := strings.Fields(stdout)
	expected := []string{"alpha", "bravo"}
	if len(lines) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, lines)
	}
	for i, name := range expected {
		if lines[i] != name {
			t.Fatalf("expected %s at position %d, got %s", name, i, lines[i])
		}
	}
}

func TestActivateFailsWhenEnvMissing(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "activate", "ghost")
	if err == nil || !strings.Contains(err.Error(), "env \"ghost\" not found") {
		t.Fatalf("expected missing env error, got %v", err)
	}
}

func TestActivateEmitsProfileSource(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())
	profileDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	profilePath := filepath.Join(profileDir, "custom.sh")
	if err := os.WriteFile(profilePath, []byte("#!/bin/sh\necho custom\n"), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	md := project.Metadata{Name: "demo", Shell: "bash@5.2", Profile: "custom"}
	if err := project.WriteMetadata(dir, md); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate", "demo")
	if err != nil {
		t.Fatalf("activate returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "SHELLENV_ENV_NAME=demo") {
		t.Fatalf("expected env export in output: %s", stdout)
	}
	if !strings.Contains(stdout, profilePath) {
		t.Fatalf("expected profile path in activation code, got: %s", stdout)
	}
}

func TestActivateUsesCurrentEnvFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())
	if err := project.WriteMetadata(dir, project.Metadata{Name: "current-env", Shell: "bash@5.0"}); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := project.WriteCurrent(dir, "current-env"); err != nil {
		t.Fatalf("write current: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate")
	if err != nil {
		t.Fatalf("activate returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "SHELLENV_ENV_NAME=current-env") {
		t.Fatalf("expected fallback to current env, got: %s", stdout)
	}
}

func TestActivateIncludesRuntimeBinWhenInstalled(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELLENV_HOME", home)

	runtimeBinDir := filepath.Join(home, "installs", "bash", "5.2", "bin")
	if err := os.MkdirAll(runtimeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime bin: %v", err)
	}
	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@5.2"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate", "--shell-type", "bash")
	if err != nil {
		t.Fatalf("activate returned error: %v (stderr: %s)", err, stderr)
	}
	runtimeBin := filepath.Join(home, "installs", "bash", "5.2", "bin")
	envBin := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if !strings.Contains(stdout, "PATH="+envBin+":"+runtimeBin+":$PATH") {
		t.Fatalf("expected PATH with env bin then runtime bin, got: %s", stdout)
	}
}

func TestActivateWarnsOnMissingRuntimeToStderrOnly(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@9.9"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate", "--shell-type", "bash")
	if err != nil {
		t.Fatalf("activate returned error: %v (stderr: %s)", err, stderr)
	}
	// stdout is eval'd by the user's shell: it must carry no warning text
	// and no reference to the uninstalled runtime path.
	if strings.Contains(stdout, "warning") || strings.Contains(stdout, "installs") {
		t.Fatalf("activation stdout must stay pure shell code, got: %s", stdout)
	}
	if !strings.Contains(stderr, "bash@9.9") || !strings.Contains(stderr, "not installed") {
		t.Fatalf("expected missing-runtime warning on stderr, got: %s", stderr)
	}
}

func TestActivateStrictShellErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash@9.9"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	_, _, err := runCLI(t, dir, "activate", "--strict-shell", "--shell-type", "bash")
	if err == nil || !strings.Contains(err.Error(), "bash@9.9") || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected strict-shell missing-runtime error, got %v", err)
	}
}

func TestActivateFishUsesFishProfile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	profDir := filepath.Join(dir, "profiles")
	if err := os.MkdirAll(profDir, 0o755); err != nil {
		t.Fatalf("mkdir profiles: %v", err)
	}
	for name, content := range map[string]string{
		"custom.fish": "# fish variant\n",
		"custom.sh":   "set -euo pipefail\n",
	} {
		if err := os.WriteFile(filepath.Join(profDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if _, _, err := runCLI(t, dir, "create", "--shell", "fish", "--profile", "custom"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate", "--shell-type", "fish")
	if err != nil {
		t.Fatalf("activate returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "custom.fish") || !strings.Contains(stdout, "source") {
		t.Fatalf("fish activation should source the .fish profile, got: %s", stdout)
	}
	if strings.Contains(stdout, "custom.sh") {
		t.Fatalf("fish activation must not reference the POSIX profile, got: %s", stdout)
	}
}

func TestActivateStdoutNeverOverridesHomeOrTmp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout, _, err := runCLI(t, dir, "activate", "--shell-type", "bash")
	if err != nil {
		t.Fatalf("activate returned error: %v", err)
	}
	// Default activation's contract: the eval'd snippet adjusts
	// PATH/prompt/SHELLENV_* only — redirecting HOME/TMPDIR/XDG_* in the
	// user's live shell is opt-in via --isolate-home.
	for _, forbidden := range []string{"HOME=", "TMPDIR=", "XDG_"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("activation snippet must not set %s, got: %s", forbidden, stdout)
		}
	}
}

func TestActivateIsolateHomeEmitsSandboxExports(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELLENV_HOME", t.TempDir())

	if _, _, err := runCLI(t, dir, "create", "--shell", "bash"); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "activate", "--isolate-home", "--shell-type", "bash")
	if err != nil {
		t.Fatalf("activate --isolate-home returned error: %v (stderr: %s)", err, stderr)
	}
	sandbox := project.SandboxHomeDir(dir, "default")
	if !strings.Contains(stdout, "export HOME="+sandbox) {
		t.Fatalf("expected HOME redirect to %s, got: %s", sandbox, stdout)
	}
	if strings.Contains(stdout, "warning") || strings.Contains(stdout, "deactivate") {
		t.Fatalf("hint text must not reach eval'd stdout, got: %s", stdout)
	}
	// The CLI pre-creates the sandbox dirs (the snippet has no mkdir).
	for _, d := range []string{sandbox, filepath.Join(sandbox, "tmp"), filepath.Join(sandbox, ".config")} {
		if fi, statErr := os.Stat(d); statErr != nil || !fi.IsDir() {
			t.Fatalf("sandbox dir %s not pre-created: %v", d, statErr)
		}
	}
	// The restore hint goes to stderr.
	if !strings.Contains(stderr, "deactivate") {
		t.Fatalf("expected deactivate hint on stderr, got: %s", stderr)
	}
}

func TestDeactivateStdoutPureShellCode(t *testing.T) {
	dir := t.TempDir()

	// deactivate needs no env, no .shellenv, and must never fail.
	stdout, stderr, err := runCLI(t, dir, "deactivate", "--shell-type", "bash")
	if err != nil {
		t.Fatalf("deactivate returned error: %v", err)
	}
	if !strings.Contains(stdout, "SHELLENV_OLD_PATH") || !strings.Contains(stdout, "unset SHELLENV_ACTIVE") {
		t.Fatalf("unexpected deactivate snippet: %s", stdout)
	}
	if stderr != "" {
		t.Fatalf("deactivate must not write to stderr, got: %s", stderr)
	}
}

func TestDeactivateFishVariant(t *testing.T) {
	dir := t.TempDir()

	stdout, _, err := runCLI(t, dir, "deactivate", "--shell-type", "fish")
	if err != nil {
		t.Fatalf("deactivate returned error: %v", err)
	}
	if !strings.Contains(stdout, "set -e SHELLENV_ACTIVE") {
		t.Fatalf("expected fish unset syntax, got: %s", stdout)
	}
}

func TestWhichFindsBinaryInEnv(t *testing.T) {
	dir := t.TempDir()

	binDir := filepath.Join(project.EnvDir(dir, "default"), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	target := filepath.Join(binDir, "tool")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	stdout, stderr, err := runCLI(t, dir, "which", "tool")
	if err != nil {
		t.Fatalf("which returned error: %v (stderr: %s)", err, stderr)
	}
	got := strings.TrimSpace(stdout)
	if got != target {
		if resolved, err := filepath.EvalSymlinks(target); err != nil || got != resolved {
			t.Fatalf("expected %s, got %s", target, stdout)
		}
	}
}

func TestWhichErrorsWhenMissing(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runCLI(t, dir, "which", "missing")
	if err == nil || !strings.Contains(err.Error(), "binary \"missing\" not found") {
		t.Fatalf("expected missing binary error, got %v", err)
	}
}

func TestDoctorWarnsOnWorldWritable(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o777); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.Chmod(home, 0o777); err != nil {
		t.Fatalf("chmod home: %v", err)
	}
	t.Setenv("SHELLENV_HOME", home)

	stdout, stderr, err := runCLI(t, "", "doctor")
	if err != nil {
		t.Fatalf("doctor returned error: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, "Home:  "+home) {
		t.Fatalf("expected home path in output: %s", stdout)
	}
	if !strings.Contains(stdout, "Warning") {
		t.Fatalf("expected warning about permissions, got: %s", stdout)
	}
	if !strings.Contains(stdout, "OK") {
		t.Fatalf("expected OK line in output: %s", stdout)
	}
	if strings.Contains(stdout, "Shims") {
		t.Fatalf("doctor should no longer report shims, got: %s", stdout)
	}
	// Installers build from source, so doctor reports the toolchain state.
	if !strings.Contains(stdout, "Build toolchain") {
		t.Fatalf("doctor should report build toolchain status, got: %s", stdout)
	}
}
