package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/shell"
)

var actShellType string

func init() {
	c := &cobra.Command{
		Use:   "activate [<env>]",
		Short: "Print shell code to activate an env (eval it in your shell)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			if name == "" {
				if cur, err := project.ReadCurrent(cwd); err == nil {
					name = cur
				}
			}
			if name == "" {
				name = "default"
			}

			pdir := project.EnvDir(cwd, name)
			if _, err := os.Stat(pdir); err != nil {
				return fmt.Errorf("env %q not found at %s (run 'shellenv create' first)", name, pdir)
			}

			// Load metadata for profile
			md, _ := project.ReadMetadata(cwd, name)
			var profilePath string
			if md.Profile != "" {
				if p, ok := shell.ResolveProfile(cwd, md.Profile); ok {
					profilePath = p
				}
			}

			code := shell.ActivationCodeWithProfile(actShellType, pdir, name, profilePath)
			fmt.Println(code)
			return nil
		},
	}
	c.Flags().StringVar(&actShellType, "shell-type", "", "override detected shell type (bash|zsh|fish)")
	rootCmd.AddCommand(c)
}
