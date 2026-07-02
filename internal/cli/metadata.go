package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/systemhalted/shellenv/internal/project"
)

// loadMetadata reads an env's metadata, treating a missing file as an empty
// declaration (unpinned shell, no profile) but warning on anything else —
// silently ignoring a corrupt metadata.json would run the command with no
// profile or shell pinning while the user believes both are in effect.
func loadMetadata(cwd, envName string) project.Metadata {
	md, err := project.ReadMetadata(cwd, envName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "warning: cannot read %s (%v); ignoring it — no profile or shell pinning will apply\n",
			filepath.Join(project.EnvDir(cwd, envName), "metadata.json"), err)
	}
	return md
}
