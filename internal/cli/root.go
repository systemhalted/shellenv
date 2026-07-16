package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shellenv",
	Short: "Per-project shell sandboxes (like pyenv, but for shells)",
	Long:  "shellenv creates isolated environments for testing shell scripts across different shells and option profiles.",
	// Errors are reported by Execute(); don't let Cobra dump the usage block or
	// a duplicate "Error:" line on every runtime failure (e.g. a command run via
	// `exec` exiting non-zero).
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	err := ExecuteWithArgs(os.Args[1:], os.Stdout, os.Stderr)
	if err == nil {
		return
	}
	// A command run via `exec` that exits non-zero already wrote its own output;
	// mirror its status without printing a redundant shellenv-level message.
	var ee *exitError
	if errors.As(err, &ee) {
		os.Exit(ee.code)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// exitError carries a child process's exit status so `exec` can propagate it
// to shellenv's own exit code without printing an extra error message.
type exitError struct{ code int }

func (e *exitError) Error() string {
	return fmt.Sprintf("command exited with status %d", e.code)
}

// ExecuteWithArgs allows tests to run commands with custom args and writers.
func ExecuteWithArgs(args []string, stdout, stderr io.Writer) error {
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	return rootCmd.Execute()
}

// RootCmd exposes the assembled command tree for out-of-binary tooling
// (cmd/gen-man). Kept accessor-only so the doc generator's dependencies are
// never linked into the shellenv binary itself.
func RootCmd() *cobra.Command { return rootCmd }
