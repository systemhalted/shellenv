package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shellenv",
	Short: "Per-project shell sandboxes (like pyenv, but for shells)",
	Long:  "shellenv creates isolated environments for testing shell scripts across different shells and option profiles.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
