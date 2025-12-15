package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/project"
)

func init() { rootCmd.AddCommand(useCmd) }

var useCmd = &cobra.Command{
	Use:   "use <env>",
	Short: "Set the default env for this project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		name := args[0]
		if _, err := os.Stat(project.EnvDir(cwd, name)); err != nil {
			return fmt.Errorf("env %q not found (run 'shellenv create --name %s ...')", name, name)
		}
		if err := project.WriteCurrent(cwd, name); err != nil {
			return err
		}
		fmt.Printf("Now using env %q in this project\n", name)
		return nil
	},
}
