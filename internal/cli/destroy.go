package cli

import (
	"fmt"
	"os"

	"github.com/example/shellenv/internal/project"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(destroyCmd) }

var destroyCmd = &cobra.Command{
	Use:   "destroy <env>",
	Short: "Remove a project env",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		name := args[0]
		pdir := project.EnvDir(cwd, name)
		if _, err := os.Stat(pdir); err != nil {
			return fmt.Errorf("env %q not found at %s", name, pdir)
		}
		if err := os.RemoveAll(pdir); err != nil { return err }
		fmt.Printf("Destroyed env %q\n", name)
		return nil
	},
}
