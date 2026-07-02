package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
)

// warnDeclaringEnvs is a best-effort check that envs in the current
// directory's project don't still pin the removed runtime. shellenv keeps no
// registry of projects, so envs elsewhere on disk are invisible to this.
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
