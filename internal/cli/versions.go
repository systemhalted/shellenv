package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/shellenv/internal/env"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(versionsCmd) }

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List installed shell runtimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		inst, err := env.InstallsDir()
		if err != nil { return err }
		_ = os.MkdirAll(inst, 0o755)
		entries, _ := os.ReadDir(inst)
		if len(entries) == 0 {
			fmt.Println("No runtimes installed yet. Try: shellenv install bash@5.2")
			return nil
		}
		for _, sh := range entries {
			if !sh.IsDir() { continue }
			vers, _ := os.ReadDir(filepath.Join(inst, sh.Name()))
			for _, v := range vers {
				if v.IsDir() { fmt.Printf("%s@%s\n", sh.Name(), v.Name()) }
			}
		}
		return nil
	},
}
