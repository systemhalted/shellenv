package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/registry"
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

// registerEnvBestEffort records the env in the global registry. The registry
// is advisory (decision 15): any failure is a stderr warning, never an error,
// so create works even with an unwritable or absent SHELLENV_HOME.
func registerEnvBestEffort(cwd string, md project.Metadata) {
	home, err := env.Home()
	if err == nil {
		// create must work before `shellenv init`; MkdirAll is cheap and
		// idempotent, and its failure surfaces via the registry.Add below.
		_ = os.MkdirAll(home, 0o755)
		err = registry.Add(home, registry.Entry{
			Root:       cwd,
			Name:       md.Name,
			Shell:      md.Shell,
			Registered: time.Now().UTC().Format(time.RFC3339),
		})
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update env registry: %v\n", err)
	}
}

// unregisterEnvBestEffort drops the env from the global registry; same
// advisory stance as registerEnvBestEffort.
func unregisterEnvBestEffort(cwd, name string) {
	home, err := env.Home()
	if err == nil {
		err = registry.Remove(home, cwd, name)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update env registry: %v\n", err)
	}
}
