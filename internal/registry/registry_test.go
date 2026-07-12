package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileIsEmptyNotError(t *testing.T) {
	home := t.TempDir()

	f, err := Load(home)
	if err != nil {
		t.Fatalf("missing registry must load as empty, got error: %v", err)
	}
	if f.Version != 1 || len(f.Envs) != 0 {
		t.Fatalf("expected empty version-1 file, got %+v", f)
	}
}

func TestAddCreatesFileAndRoundTrips(t *testing.T) {
	home := t.TempDir()

	e := Entry{Root: "/proj/a", Name: "default", Shell: "bash@5.2", Registered: "2026-07-12T00:00:00Z"}
	if err := Add(home, e); err != nil {
		t.Fatalf("add: %v", err)
	}

	f, err := Load(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if f.Version != 1 || len(f.Envs) != 1 || f.Envs[0] != e {
		t.Fatalf("round-trip mismatch: %+v", f)
	}
}

func TestAddUpsertsByRootAndName(t *testing.T) {
	home := t.TempDir()

	if err := Add(home, Entry{Root: "/proj/a", Name: "default", Shell: "bash@5.2"}); err != nil {
		t.Fatal(err)
	}
	// Same (root, name), new shell: must replace, not duplicate.
	if err := Add(home, Entry{Root: "/proj/a", Name: "default", Shell: "zsh@5.9"}); err != nil {
		t.Fatal(err)
	}
	// Different name under the same root: a second entry.
	if err := Add(home, Entry{Root: "/proj/a", Name: "ci", Shell: "bash@5.2"}); err != nil {
		t.Fatal(err)
	}

	f, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Envs) != 2 {
		t.Fatalf("expected 2 entries after upsert, got %+v", f.Envs)
	}
	// Sorted by root then name: ci before default.
	if f.Envs[0].Name != "ci" || f.Envs[1].Name != "default" || f.Envs[1].Shell != "zsh@5.9" {
		t.Fatalf("unexpected entries: %+v", f.Envs)
	}
}

func TestRemoveExistingAndAbsent(t *testing.T) {
	home := t.TempDir()

	if err := Add(home, Entry{Root: "/proj/a", Name: "default", Shell: "bash@5.2"}); err != nil {
		t.Fatal(err)
	}
	if err := Remove(home, "/proj/a", "default"); err != nil {
		t.Fatalf("remove existing: %v", err)
	}
	f, err := Load(home)
	if err != nil || len(f.Envs) != 0 {
		t.Fatalf("entry not removed: %+v, %v", f, err)
	}

	// Absent entry and missing file are both silent no-ops.
	if err := Remove(home, "/proj/a", "default"); err != nil {
		t.Fatalf("remove absent: %v", err)
	}
	if err := Remove(t.TempDir(), "/proj/x", "default"); err != nil {
		t.Fatalf("remove with no registry file: %v", err)
	}
}

func TestLoadCorruptFileReturnsEmptyAndError(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(Path(home), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := Load(home)
	if err == nil {
		t.Fatalf("corrupt registry must surface an error")
	}
	if len(f.Envs) != 0 {
		t.Fatalf("corrupt registry must load as empty, got %+v", f)
	}

	// A subsequent Add rewrites a valid file over the corrupt one.
	if err := Add(home, Entry{Root: "/proj/a", Name: "default"}); err != nil {
		t.Fatalf("add over corrupt file: %v", err)
	}
	if f, err := Load(home); err != nil || len(f.Envs) != 1 {
		t.Fatalf("registry not repaired: %+v, %v", f, err)
	}
}

func TestSaveLeavesNoTempFile(t *testing.T) {
	home := t.TempDir()
	if err := Save(home, File{Version: 1, Envs: []Entry{{Root: "/p", Name: "default"}}}); err != nil {
		t.Fatal(err)
	}

	ents, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.Name() != filepath.Base(Path(home)) {
			t.Fatalf("unexpected leftover file %q", e.Name())
		}
	}
	b, _ := os.ReadFile(Path(home))
	if !strings.Contains(string(b), "\"version\": 1") {
		t.Fatalf("expected indented JSON, got %q", b)
	}
}
