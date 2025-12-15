package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
)

func init() { rootCmd.AddCommand(installCmd) }

var installCmd = &cobra.Command{
	Use:   "install <shell>@<version>",
	Short: "Install (or declare) a shell runtime version (placeholder)",
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
		dir := filepath.Join(inst, parts[0], parts[1], "bin")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "installed.txt"), []byte("placeholder runtime\n"), 0o644); err != nil {
			return err
		}
		fmt.Printf("Installed %s@%s into %s (placeholder)\n", parts[0], parts[1], filepath.Dir(dir))
		return nil
	},
}
