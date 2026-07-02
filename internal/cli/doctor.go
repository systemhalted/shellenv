package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
)

func init() { rootCmd.AddCommand(doctorCmd) }

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment health",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := env.EnsureHome()
		if err != nil {
			return err
		}
		fmt.Printf("Home:  %s\n", h)
		if fi, err := os.Stat(h); err == nil && (fi.Mode()&0o002) != 0 {
			fmt.Println("Warning: home dir is world-writable; consider chmod 755 or 700")
		}
		fmt.Println("OK")
		return nil
	},
}
