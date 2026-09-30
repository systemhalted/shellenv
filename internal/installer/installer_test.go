package installer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceFor(t *testing.T) {
	src, err := SourceFor("bash", "5.2")
	if err != nil {
		t.Fatalf("bash@5.2: %v", err)
	}
	if src.URL != "https://ftp.gnu.org/gnu/bash/bash-5.2.tar.gz" {
		t.Fatalf("bash URL = %q", src.URL)
	}
	if src.SHA256 != "a139c166df7ff4471c5e0733051642ee5556c1cc8a4a78f145583c5c81ab32fb" {
		t.Fatalf("bash@5.2 checksum = %q", src.SHA256)
	}

	src, err = SourceFor("zsh", "5.9")
	if err != nil {
		t.Fatalf("zsh@5.9: %v", err)
	}
	if src.URL != "https://downloads.sourceforge.net/project/zsh/zsh/5.9/zsh-5.9.tar.xz" {
		t.Fatalf("zsh URL = %q", src.URL)
	}
	if src.SHA256 != "9b8d1ecedd5b5e81fbf1918e876752a7dd948e05c1a0dba10ab863842d45acd5" {
		t.Fatalf("zsh@5.9 checksum = %q", src.SHA256)
	}

	// macOS's /bin/bash, pinned so tools that target it can test against it.
	src, err = SourceFor("bash", "3.2.57")
	if err != nil {
		t.Fatalf("bash@3.2.57: %v", err)
	}
	if src.URL != "https://ftp.gnu.org/gnu/bash/bash-3.2.57.tar.gz" {
		t.Fatalf("bash@3.2.57 URL = %q", src.URL)
	}
	if src.SHA256 != "3fa9daf85ebf35068f090ce51283ddeeb3c75eb5bc70b1a4a7cb05868bfe06a4" {
		t.Fatalf("bash@3.2.57 checksum = %q", src.SHA256)
	}

	// Unpinned version: URL is derived, checksum empty.
	src, err = SourceFor("bash", "5.1")
	if err != nil || src.SHA256 != "" || !strings.Contains(src.URL, "bash-5.1.tar.gz") {
		t.Fatalf("bash@5.1 = (%+v, %v)", src, err)
	}

	// Unsupported shell: clear error naming what is supported.
	if _, err := SourceFor("fish", "3.7"); err == nil || !strings.Contains(err.Error(), "bash") || !strings.Contains(err.Error(), "zsh") {
		t.Fatalf("expected unsupported-shell error naming bash/zsh, got %v", err)
	}
}

// fakeInstaller returns an Installer whose externals are stubbed, plus
// recorders for commands run and URLs fetched.
func fakeInstaller(t *testing.T, home string) (*Installer, *[][]string, *[]string) {
	t.Helper()
	var cmds [][]string
	var fetched []string
	in := New(home, &bytes.Buffer{})
	in.LookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	in.Fetch = func(url, dest string) error {
		fetched = append(fetched, url)
		return os.WriteFile(dest, []byte("fake-tarball"), 0o644)
	}
	in.Run = func(dir string, logTo *os.File, name string, args ...string) error {
		cmds = append(cmds, append([]string{name}, args...))
		switch name {
		case "tar":
			// Simulate extraction: create the source dir inside the build root.
			return os.MkdirAll(filepath.Join(dir, "src-dir"), 0o755)
		case "make":
			if len(args) > 0 && args[len(args)-1] == "install" {
				// Simulate `make install` creating the prefix bin.
				return os.MkdirAll(filepath.Join(in.prefix("bash", "5.1"), "bin"), 0o755)
			}
		}
		return nil
	}
	return in, &cmds, &fetched
}

func TestInstallHappyPathUnpinnedWarns(t *testing.T) {
	home := t.TempDir()
	out := &bytes.Buffer{}
	in, cmds, fetched := fakeInstaller(t, home)
	in.Out = out

	prefix, err := in.Install("bash", "5.1")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	want := filepath.Join(home, "installs", "bash", "5.1")
	if prefix != want {
		t.Fatalf("prefix = %q, want %q", prefix, want)
	}
	if len(*fetched) != 1 || !strings.Contains((*fetched)[0], "bash-5.1.tar.gz") {
		t.Fatalf("fetched = %v", *fetched)
	}
	// tar extract, configure, make, make install — in order.
	var names []string
	for _, c := range *cmds {
		names = append(names, c[0])
	}
	if got := strings.Join(names, " "); got != "tar ./configure make make" {
		t.Fatalf("command sequence = %q", got)
	}
	if !strings.Contains((*cmds)[1][1], "--prefix="+want) {
		t.Fatalf("configure args = %v", (*cmds)[1])
	}
	// bash 5.x still has pre-C23 declarations, so it builds as gnu17 and
	// gets nothing else.
	if got := strings.Join((*cmds)[1][2:], " | "); got != "CFLAGS=-O2 -std=gnu17 | CFLAGS_FOR_BUILD=-O2 -std=gnu17" {
		t.Fatalf("configure extra args for bash@5.1 = %q", got)
	}
	// No pinned checksum: a warning is printed but the build proceeds.
	if !strings.Contains(out.String(), "no pinned checksum") {
		t.Fatalf("expected unpinned-checksum warning, got: %s", out.String())
	}
	// Metadata records the source; the build root is cleaned up.
	marker, err := os.ReadFile(filepath.Join(want, "installed.txt"))
	if err != nil || !strings.Contains(string(marker), "bash-5.1.tar.gz") {
		t.Fatalf("installed.txt = %q, %v", marker, err)
	}
	ents, _ := os.ReadDir(filepath.Join(home, "tmp"))
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "build-") {
			t.Fatalf("build dir %s not cleaned up", e.Name())
		}
	}
}

func TestInstallVerifiesPinnedChecksum(t *testing.T) {
	home := t.TempDir()
	in, _, _ := fakeInstaller(t, home)

	// Fake fetch writes bytes that cannot match bash@5.2's pinned checksum.
	_, err := in.Install("bash", "5.2")
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
	// The corrupt download must not be left in the cache for a retry to trust.
	if _, statErr := os.Stat(filepath.Join(home, "cache", "bash-5.2.tar.gz")); !os.IsNotExist(statErr) {
		t.Fatalf("corrupt cache file should be removed, stat err=%v", statErr)
	}
}

func TestInstallAcceptsCorrectPinnedChecksum(t *testing.T) {
	home := t.TempDir()
	in, _, _ := fakeInstaller(t, home)
	content := []byte("good-tarball")
	sum := sha256.Sum256(content)
	in.Fetch = func(url, dest string) error { return os.WriteFile(dest, content, 0o644) }
	in.pinned = map[string]string{"bash@5.1": hex.EncodeToString(sum[:])}

	if _, err := in.Install("bash", "5.1"); err != nil {
		t.Fatalf("install with matching checksum failed: %v", err)
	}
}

func TestInstallRequireChecksumRejectsUnpinned(t *testing.T) {
	home := t.TempDir()
	in, cmds, fetched := fakeInstaller(t, home)
	var lookups []string
	in.LookPath = func(file string) (string, error) {
		lookups = append(lookups, file)
		return "/usr/bin/" + file, nil
	}
	in.RequireChecksum = true

	_, err := in.Install("bash", "5.1")
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected refusing-unverified error, got %v", err)
	}
	// Refusal must happen before preflight and download: nothing touched.
	if len(*cmds) != 0 || len(*fetched) != 0 || len(lookups) != 0 {
		t.Fatalf("no externals may run on refusal (cmds=%v fetched=%v lookups=%v)", *cmds, *fetched, lookups)
	}
}

func TestInstallRequireChecksumAcceptsPinned(t *testing.T) {
	home := t.TempDir()
	in, _, _ := fakeInstaller(t, home)
	content := []byte("good-tarball")
	sum := sha256.Sum256(content)
	in.Fetch = func(url, dest string) error { return os.WriteFile(dest, content, 0o644) }
	in.pinned = map[string]string{"bash@5.1": hex.EncodeToString(sum[:])}
	in.RequireChecksum = true

	if _, err := in.Install("bash", "5.1"); err != nil {
		t.Fatalf("install with pinned checksum and --require-checksum failed: %v", err)
	}
}

func TestInstallSkipsWhenAlreadyInstalled(t *testing.T) {
	home := t.TempDir()
	in, cmds, fetched := fakeInstaller(t, home)
	binDir := filepath.Join(home, "installs", "bash", "5.1", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "bash"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := in.Install("bash", "5.1"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(*cmds) != 0 || len(*fetched) != 0 {
		t.Fatalf("already-installed runtime must not rebuild (cmds=%v fetched=%v)", *cmds, *fetched)
	}
}

func TestInstallPreflightReportsMissingTools(t *testing.T) {
	home := t.TempDir()
	in, _, _ := fakeInstaller(t, home)
	in.LookPath = func(file string) (string, error) { return "", fmt.Errorf("not found") }

	_, err := in.Install("bash", "5.1")
	if err == nil || !strings.Contains(err.Error(), "cc") || !strings.Contains(err.Error(), "make") {
		t.Fatalf("expected preflight error naming missing tools, got %v", err)
	}
}

func TestInstallBuildFailurePointsAtLog(t *testing.T) {
	home := t.TempDir()
	in, _, _ := fakeInstaller(t, home)
	in.Run = func(dir string, logTo *os.File, name string, args ...string) error {
		if name == "tar" {
			return os.MkdirAll(filepath.Join(dir, "src-dir"), 0o755)
		}
		return fmt.Errorf("configure exploded")
	}

	_, err := in.Install("bash", "5.1")
	if err == nil || !strings.Contains(err.Error(), "build.log") {
		t.Fatalf("expected build failure pointing at build.log, got %v", err)
	}
}

// bash 3.2 is pre-C99 code. Current gcc and clang default to C23, where
// `f()` means "no arguments" and implicit declarations are errors, so both
// the build and the build-time helpers (mkbuiltins, via CFLAGS_FOR_BUILD)
// need gnu89 and the relaxed diagnostics.
func TestInstallBash32PassesLegacyCompilerFlags(t *testing.T) {
	home := t.TempDir()
	in, cmds, _ := fakeInstaller(t, home)
	sum := sha256.Sum256([]byte("fake-tarball"))
	in.pinned = map[string]string{"bash@3.2.57": hex.EncodeToString(sum[:])}
	run := in.Run
	in.Run = func(dir string, logTo *os.File, name string, args ...string) error {
		if name == "make" && len(args) > 0 && args[len(args)-1] == "install" {
			return os.MkdirAll(filepath.Join(in.prefix("bash", "3.2.57"), "bin"), 0o755)
		}
		return run(dir, logTo, name, args...)
	}

	if _, err := in.Install("bash", "3.2.57"); err != nil {
		t.Fatalf("install: %v", err)
	}
	var configure []string
	for _, c := range *cmds {
		if c[0] == "./configure" {
			configure = c
		}
	}
	args := strings.Join(configure, "\n")
	for _, want := range []string{"CFLAGS=", "CFLAGS_FOR_BUILD=", "-std=gnu89", "-Wno-implicit-function-declaration"} {
		if !strings.Contains(args, want) {
			t.Fatalf("configure args missing %q: %v", want, configure)
		}
	}
	for _, a := range configure[1:] {
		if strings.HasPrefix(a, "CFLAGS") && !strings.Contains(a, "-std=gnu89") {
			t.Fatalf("%s lacks -std=gnu89", a)
		}
	}
}

// bash 5.2's mkbuiltins declares functions with `()`, which C23 (the gcc 15+
// and clang 16+ default) reads as "no arguments", so 5.x needs -std=gnu17 in
// both CFLAGS and CFLAGS_FOR_BUILD.
func TestInstallBash52PassesPreC23CompilerFlags(t *testing.T) {
	home := t.TempDir()
	in, cmds, _ := fakeInstaller(t, home)
	sum := sha256.Sum256([]byte("fake-tarball"))
	in.pinned = map[string]string{"bash@5.2": hex.EncodeToString(sum[:])}
	run := in.Run
	in.Run = func(dir string, logTo *os.File, name string, args ...string) error {
		if name == "make" && len(args) > 0 && args[len(args)-1] == "install" {
			return os.MkdirAll(filepath.Join(in.prefix("bash", "5.2"), "bin"), 0o755)
		}
		return run(dir, logTo, name, args...)
	}

	if _, err := in.Install("bash", "5.2"); err != nil {
		t.Fatalf("install: %v", err)
	}
	var configure []string
	for _, c := range *cmds {
		if c[0] == "./configure" {
			configure = c
		}
	}
	for _, want := range []string{"CFLAGS=-O2 -std=gnu17", "CFLAGS_FOR_BUILD=-O2 -std=gnu17"} {
		found := false
		for _, a := range configure {
			if a == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("configure args missing %q: %v", want, configure)
		}
	}
}
