package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadMetadata(t *testing.T) {
	cwd := t.TempDir()
	md := Metadata{Name: "default", Shell: "bash@5.2", Profile: "strict"}
	if err := WriteMetadata(cwd, md); err != nil { t.Fatal(err) }

	got, err := ReadMetadata(cwd, "default")
	if err != nil { t.Fatal(err) }
	if got.Shell != "bash@5.2" { t.Fatalf("want bash@5.2, got %s", got.Shell) }

	if _, err := os.Stat(filepath.Join(EnvDir(cwd, "default"), "bin")); err != nil {
	 t.Fatalf("bin dir missing: %v", err)
	}
}
