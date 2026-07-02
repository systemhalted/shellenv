package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/shell"
)

var (
	actShellType   string
	actStrictShell bool
)

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

			// Load metadata for profile and declared shell
			md, _ := project.ReadMetadata(cwd, name)
			var profilePath string
			if md.Profile != "" {
				if p, ok := shell.ResolveProfile(cwd, md.Profile); ok {
					profilePath = p
				}
			}

			// Resolve the declared shell runtime. Anything that is not pure
			// shell code must go to stderr: stdout is eval'd by the caller.
			var runtimeBin string
			rb, status, err := env.ResolveRuntime(md.Shell)
			if err != nil {
				return err
			}
			switch status {
			case env.RuntimeFound:
				runtimeBin = rb
			case env.RuntimeMissing:
				if actStrictShell {
					return fmt.Errorf("declared shell %q is not installed (run 'shellenv install %s', or omit --strict-shell)", md.Shell, md.Shell)
				}
				fmt.Fprintf(os.Stderr, "warning: declared shell %q is not installed; falling back to the system shell (run 'shellenv install %s' or use --strict-shell to enforce)\n", md.Shell, md.Shell)
			case env.RuntimeUnpinned:
				if actStrictShell {
					return fmt.Errorf("env %q declares no <shell>@<version> to enforce (shell: %q); --strict-shell has nothing to pin", name, md.Shell)
				}
			}

			code := shell.ActivationCodeWithOptions(actShellType, pdir, name, shell.ActivationOptions{
				ProfilePath:   profilePath,
				RuntimeBinDir: runtimeBin,
			})
			fmt.Println(code)
			return nil
		},
	}
	c.Flags().StringVar(&actShellType, "shell-type", "", "override detected shell type (bash|zsh|fish)")
	c.Flags().BoolVar(&actStrictShell, "strict-shell", false,
		"require the declared shell runtime to be installed; fail instead of falling back to the system shell")
	rootCmd.AddCommand(c)
}
