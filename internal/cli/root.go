package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shellenv",
	Short: "Per-project shell sandboxes (like pyenv, but for shells)",
	Long:  "shellenv creates isolated environments for testing shell scripts across different shells and option profiles.",
}

func Execute() {
	if err := ExecuteWithArgs(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ExecuteWithArgs allows tests to run commands with custom args and writers.
func ExecuteWithArgs(args []string, stdout, stderr io.Writer) error {
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	return rootCmd.Execute()
}
