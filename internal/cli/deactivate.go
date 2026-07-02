package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/shell"
)

var deactShellType string

func init() {
	c := &cobra.Command{
		Use:   "deactivate",
		Short: "Print shell code to restore the session (eval it in your shell)",
		Args:  cobra.NoArgs,
		// No env lookup and no error paths: the snippet is guard-everything,
		// so deactivate must succeed even with no .shellenv/ present and be
		// a silent no-op when nothing is active.
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(shell.DeactivationCode(shell.DetectShell(deactShellType)))
			return nil
		},
	}
	c.Flags().StringVar(&deactShellType, "shell-type", "", "override detected shell type (bash|zsh|fish)")
	rootCmd.AddCommand(c)
}
