package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/installer"
)

var installRequireChecksum bool

func init() {
	installCmd.Flags().BoolVar(&installRequireChecksum, "require-checksum", false, "fail instead of warning when the version has no pinned checksum")
	rootCmd.AddCommand(installCmd)
}

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
		inst := installer.New(home, os.Stdout)
		inst.RequireChecksum = installRequireChecksum
		// The installer reports its own outcome ("already installed" for
		// no-ops, "Installed ... into ..." after a real build).
		_, err = inst.Install(name, version)
		return err
	},
}
