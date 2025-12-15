package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/systemhalted/shellenv/internal/cli"
)

func TestHelpRuns(t *testing.T) {
	var out bytes.Buffer
	if err := cli.ExecuteWithArgs([]string{"--help"}, &out, &out); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "shellenv creates isolated environments") {
		t.Fatalf("unexpected help output: %s", text)
	}
}

func TestListSucceedsInEmptyDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	var out bytes.Buffer
	if err := cli.ExecuteWithArgs([]string{"list"}, &out, &out); err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected no environments listed in empty dir, got: %q", out.String())
	}
}
