package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		parts := strings.Split(pair, "@")
		if len(parts) != 2 {
			return fmt.Errorf("expected <shell>@<version>, got %q", pair)
		}
		inst, err := env.InstallsDir()
		if err != nil {
			return err
		}
		dir := filepath.Join(inst, parts[0], parts[1])
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		fmt.Printf("Uninstalled %s@%s\n", parts[0], parts[1])
		return nil
	},
}
