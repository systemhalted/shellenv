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

		argsBeforeDash := dash
		argsAfterDash := len(args) - dash
		if argsAfterDash == 0 {
			return fmt.Errorf("missing command after --")
		}
		var envName string
		var cmdArgs []string
		if argsBeforeDash == 1 {
			envName = args[0]
		}
		cmdArgs = args[dash:]

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

		cmdPath, err := resolveCommandPath(cmdArgs[0], childEnv)
		if err != nil {
			return err
		}

		child := exec.Command(cmdPath, cmdArgs[1:]...)
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

func resolveCommandPath(name string, env []string) (string, error) {
	// If the user provided a path (./cmd or /abs/path), use it directly.
	if strings.ContainsRune(name, os.PathSeparator) {
		return name, nil
	}
	pathVar := pathFromEnv(env)
	for _, dir := range filepath.SplitList(pathVar) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("command %q not found in PATH", name)
}

func pathFromEnv(env []string) string {
	for _, v := range env {
		if strings.HasPrefix(v, "PATH=") {
			return strings.TrimPrefix(v, "PATH=")
		}
	}
	return os.Getenv("PATH")
}

func isExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	return fi.Mode()&0o111 != 0
}
