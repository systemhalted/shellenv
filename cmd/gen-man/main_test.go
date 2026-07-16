package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratesManPageTree(t *testing.T) {
	dir := t.TempDir()

	if err := run(dir); err != nil {
		t.Fatalf("run: %v", err)
	}

	// The root page and one page per command, roff-formatted.
	root, err := os.ReadFile(filepath.Join(dir, "shellenv.1"))
	if err != nil {
		t.Fatalf("root man page missing: %v", err)
	}
	if !strings.Contains(string(root), `.TH "SHELLENV" "1"`) {
		t.Fatalf("expected roff .TH header, got: %.80q", root)
	}
	if !strings.Contains(string(root), "isolated environments") {
		t.Fatalf("root page should carry the CLI description, got: %.200q", root)
	}

	for _, cmd := range []string{"exec", "install", "activate", "create", "list"} {
		p := filepath.Join(dir, "shellenv-"+cmd+".1")
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected man page for %s: %v", cmd, err)
		}
	}

	// Angle-bracket placeholders must survive md2man (which otherwise strips
	// them as HTML tags): "exec [<env>] ... <cmd>" must not become "exec []".
	execPage, err := os.ReadFile(filepath.Join(dir, "shellenv-exec.1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(execPage), "cmd") || strings.Contains(string(execPage), "exec [] ") {
		t.Fatalf("placeholders were stripped from the exec synopsis: %.300q", execPage)
	}
}

func TestRunCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "man", "man1")

	if err := run(dir); err != nil {
		t.Fatalf("run into missing dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "shellenv.1")); err != nil {
		t.Fatalf("man page not written: %v", err)
	}
}
