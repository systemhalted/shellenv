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
	actIsolateHome bool
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

			// Load metadata for profile and declared shell. The profile
			// variant depends on the target shell (fish sources .fish).
			md := loadMetadata(cwd, name)
			shellType := shell.DetectShell(actShellType)
			var profilePath string
			if md.Profile != "" {
				if p, ok := shell.ResolveProfileForShell(cwd, md.Profile, shellType); ok {
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

			opts := shell.ActivationOptions{
				ProfilePath:   profilePath,
				RuntimeBinDir: runtimeBin,
			}
			if actIsolateHome {
				// Pre-create the sandbox here so stdout stays pure shell
				// code and a mkdir failure surfaces as a normal error.
				sb, err := project.EnsureSandboxDirs(project.SandboxHomeDir(cwd, name))
				if err != nil {
					return err
				}
				opts.Sandbox = &sb
				fmt.Fprintf(os.Stderr, "note: HOME/TMPDIR/XDG_* now point at %s for this session; restore with 'eval \"$(shellenv deactivate)\"'\n", sb.Home)
			}

			code := shell.ActivationCodeWithOptions(shellType, pdir, name, opts)
			fmt.Println(code)
			return nil
		},
	}
	c.Flags().StringVar(&actShellType, "shell-type", "", "override detected shell type (bash|zsh|fish)")
	c.Flags().BoolVar(&actStrictShell, "strict-shell", false,
		"require the declared shell runtime to be installed; fail instead of falling back to the system shell")
	c.Flags().BoolVar(&actIsolateHome, "isolate-home", false,
		"redirect HOME/TMPDIR/XDG_* to the env sandbox for this session; restore with 'shellenv deactivate'")
	rootCmd.AddCommand(c)
}
