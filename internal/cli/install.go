package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/installer"
)

func init() { rootCmd.AddCommand(installCmd) }

var installCmd = &cobra.Command{
	Use:   "install <shell>@<version>",
	Short: "Download, build, and install a shell runtime from source",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pair := args[0]
		name, version, ok := env.ParseShellVersion(pair)
		if !ok {
			return fmt.Errorf("expected <shell>@<version>, got %q", pair)
		}
		home, err := env.EnsureHome()
		if err != nil {
			return err
		}
		prefix, err := installer.New(home, os.Stdout).Install(name, version)
		if err != nil {
			return err
		}
		fmt.Printf("Installed %s@%s into %s\n", name, version, prefix)
		return nil
	},
}
