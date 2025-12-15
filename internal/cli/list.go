package cli

import (
	"fmt"
	"os"

	"github.com/example/shellenv/internal/project"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(listCmd) }

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List project envs in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		items, err := project.ListEnvs(cwd)
		if err != nil { return err }
		for _, n := range items { fmt.Println(n) }
		return nil
	},
}
