package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
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
		// Then the declared runtime's bin — the same priority exec gives it,
		// so which answers with the binary exec would actually run.
		md := loadMetadata(cwd, name)
		if runtimeBin, status, err := env.ResolveRuntime(md.Shell); err == nil && status == env.RuntimeFound {
			if p, err := exec.LookPath(filepath.Join(runtimeBin, args[0])); err == nil {
				fmt.Println(p)
				return nil
			}
		}
		// Fallback to PATH search (with env/bin first if user activated)
		if p, err := exec.LookPath(args[0]); err == nil {
			fmt.Println(p)
			return nil
		}
		return fmt.Errorf("binary %q not found in env %q (looked in %s, the declared runtime, and PATH)", args[0], name, binDir)
	},
}
