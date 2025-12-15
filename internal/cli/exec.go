package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/project"
)

func init() { rootCmd.AddCommand(execCmd) }

var execCmd = &cobra.Command{
	Use:   "exec [<env>] -- <cmd> [args...]",
	Short: "Run a command within a project env without interactive activation",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dash := cmd.ArgsLenAtDash()
		if dash == -1 {
			return fmt.Errorf("usage: shellenv exec [env] -- <cmd> [args...]")
		}
		if dash > 1 {
			return fmt.Errorf("expected at most one env name before --")
		}
		if dash == len(args)-1 {
			return fmt.Errorf("missing command after --")
		}

		var envName string
		var cmdArgs []string
		if dash == 1 {
			envName = args[0]
			cmdArgs = args[1:]
		} else { // dash == 0
			cmdArgs = args
		}

		cwd, _ := os.Getwd()
		if envName == "" {
			if cur, err := project.ReadCurrent(cwd); err == nil {
				envName = cur
			}
		}
		if envName == "" {
			envName = "default"
		}

		envDir := project.EnvDir(cwd, envName)
		if _, err := os.Stat(envDir); err != nil {
			return fmt.Errorf("env %q not found at %s (run 'shellenv create' first)", envName, envDir)
		}

		binDir := filepath.Join(envDir, "bin")
		childEnv := prependPath(os.Environ(), binDir)
		childEnv = append(childEnv,
			fmt.Sprintf("SHELLENV_ENV_NAME=%s", envName),
			"SHELLENV_ACTIVE=1",
		)

		child := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		child.Env = childEnv
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Stdin = os.Stdin
		child.Dir = cwd

		return child.Run()
	},
}

func prependPath(env []string, bin string) []string {
	if bin == "" {
		return env
	}
	key := "PATH="
	for i, v := range env {
		if strings.HasPrefix(v, key) {
			env[i] = fmt.Sprintf("PATH=%s%c%s", bin, os.PathListSeparator, strings.TrimPrefix(v, key))
			return env
		}
	}
	return append(env, fmt.Sprintf("PATH=%s", bin))
}
