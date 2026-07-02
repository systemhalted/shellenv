package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
)

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
		fmt.Printf("Uninstalled %s@%s\n", name, version)
		return nil
	},
}
