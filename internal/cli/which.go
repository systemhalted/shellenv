package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/example/shellenv/internal/project"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(whichCmd) }

var whichCmd = &cobra.Command{
	Use:   "which <binary>",
	Short: "Resolve a binary inside the active project env",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		name := "default"
		if cur, err := project.ReadCurrent(cwd); err == nil && cur != "" {
			name = cur
		}
		pdir := project.EnvDir(cwd, name)
		binDir := filepath.Join(pdir, "bin")
		// Prefer env bin dir
		if p, err := exec.LookPath(filepath.Join(binDir, args[0])); err == nil {
			fmt.Println(p)
			return nil
		}
		// Fallback to PATH search (with env/bin first if user activated)
		if p, err := exec.LookPath(args[0]); err == nil {
			fmt.Println(p)
			return nil
		}
		return fmt.Errorf("binary %q not found in env %q (looked in %s and PATH)", args[0], name, binDir)
	},
}
