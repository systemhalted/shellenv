package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize global shellenv home",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := env.EnsureHome()
		if err != nil {
			return err
		}
		fmt.Printf("Initialized shellenv at %s\n", h)
		return os.WriteFile(filepath.Join(h, ".initialized"), []byte("ok\n"), 0o644)
	},
}
