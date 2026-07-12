package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/registry"
)

// warnDeclaringEnvs is a best-effort check that envs don't still pin the
// removed runtime: it scans the current directory's project (which needs no
// registration), then the global registry (decision 15) for envs elsewhere.
// Registry entries are re-validated against metadata.json on disk — vanished
// projects are pruned, and pre-registry envs are simply not seen.
func warnDeclaringEnvs(pair string) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	names, err := project.ListEnvs(cwd)
	if err != nil {
		return
	}
	for _, n := range names {
		if md, err := project.ReadMetadata(cwd, n); err == nil && md.Shell == pair {
			fmt.Fprintf(os.Stderr, "warning: env %q in this directory still declares %s\n", n, pair)
		}
	}

	home, err := env.Home()
	if err != nil {
		return
	}
	reg, err := registry.Load(home)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read env registry: %v\n", err)
		return
	}
	for _, e := range reg.Envs {
		if e.Root == cwd {
			continue // already covered by the scan above
		}
		md, err := project.ReadMetadata(e.Root, e.Name)
		if os.IsNotExist(err) {
			// The project is gone; silently drop the stale entry —
			// stale registry state is noise, not news.
			_ = registry.Remove(home, e.Root, e.Name)
			continue
		}
		if err == nil && md.Shell == pair {
			fmt.Fprintf(os.Stderr, "warning: env %q at %s still declares %s\n", e.Name, e.Root, pair)
		}
	}
}

func init() { rootCmd.AddCommand(uninstallCmd) }

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <shell>@<version>",
	Short: "Remove an installed shell runtime",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pair := args[0]
		name, version, ok := env.ParseShellVersion(pair)
		if !ok {
			return fmt.Errorf("expected <shell>@<version>, got %q", pair)
		}
		inst, err := env.InstallsDir()
		if err != nil {
			return err
		}
		dir := filepath.Join(inst, name, version)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		warnDeclaringEnvs(pair)
		fmt.Printf("Uninstalled %s@%s\n", name, version)
		return nil
	},
}
