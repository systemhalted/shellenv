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
	Short: "Initialize global shellenv home and shims",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := env.EnsureHome()
		if err != nil {
			return err
		}
		shims, _ := env.Home()
		fmt.Printf("Initialized shellenv at %s\n", h)
		fmt.Printf("Add this to your shell rc file:\n\n")
		fmt.Printf("  export PATH=\"%s/shims:$PATH\"\n\n", shims)
		return os.WriteFile(filepath.Join(h, ".initialized"), []byte("ok\n"), 0o644)
	},
}
