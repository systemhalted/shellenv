package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/shell"
)

var execWithProfile bool

func init() {
	execCmd.Flags().BoolVar(&execWithProfile, "profile", false,
		"source the env's declared profile (e.g. strict) before running the command")
	rootCmd.AddCommand(execCmd)
}

var execCmd = &cobra.Command{
	Use:   "exec [<env>] [--profile] -- <cmd> [args...]",
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
		childEnv = upsertEnv(childEnv, "SHELLENV_ENV_NAME", envName)
		childEnv = upsertEnv(childEnv, "SHELLENV_ACTIVE", "1")

		// Isolate HOME/TMPDIR/XDG so scripts write to a per-env sandbox rather
		// than the user's real home or system temp dir.
		sandboxHome := project.SandboxHomeDir(cwd, envName)
		tmpDir := filepath.Join(sandboxHome, "tmp")
		xdgConfig := filepath.Join(sandboxHome, ".config")
		xdgCache := filepath.Join(sandboxHome, ".cache")
		xdgData := filepath.Join(sandboxHome, ".local", "share")
		for _, d := range []string{sandboxHome, tmpDir, xdgConfig, xdgCache, xdgData} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return err
			}
		}
		childEnv = upsertEnv(childEnv, "HOME", sandboxHome)
		childEnv = upsertEnv(childEnv, "TMPDIR", tmpDir)
		childEnv = upsertEnv(childEnv, "XDG_CONFIG_HOME", xdgConfig)
		childEnv = upsertEnv(childEnv, "XDG_CACHE_HOME", xdgCache)
		childEnv = upsertEnv(childEnv, "XDG_DATA_HOME", xdgData)

		var child *exec.Cmd
		if execWithProfile {
			md, _ := project.ReadMetadata(cwd, envName)
			profilePath, ok := shell.ResolveProfile(cwd, md.Profile)
			if !ok {
				return fmt.Errorf("profile %q not found for env %q (looked in SHELLENV_PROFILES, ./profiles, and beside the binary)", md.Profile, envName)
			}
			shellName := profileShell(md.Shell)
			shellPath, err := resolveCommandPath(shellName, childEnv)
			if err != nil {
				return err
			}
			// Source the profile, then run the requested command *in* that shell
			// (not exec) so the profile's options govern how this shell runs it.
			// $0 is the shell name, so the command and its args become $1.. for
			// "$@". Note: a command that re-invokes an interpreter (e.g. a script
			// with its own shebang, or `bash -c`) starts fresh and only inherits
			// the profile's exported environment, not its shell options.
			script := ". " + singleQuote(profilePath) + "; \"$@\""
			argv := append([]string{"-c", script, shellName}, cmdArgs...)
			child = exec.Command(shellPath, argv...)
		} else {
			cmdPath, err := resolveCommandPath(cmdArgs[0], childEnv)
			if err != nil {
				return err
			}
			child = exec.Command(cmdPath, cmdArgs[1:]...)
		}
		child.Env = childEnv
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		child.Stdin = os.Stdin
		child.Dir = cwd

		if err := child.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code := ee.ExitCode()
				if code < 0 {
					code = 1 // terminated by signal; report a generic failure
				}
				return &exitError{code: code}
			}
			return err
		}
		return nil
	},
}

// upsertEnv sets key=value in env, replacing any existing entry for key or
// appending a new one. It mutates and returns the slice.
func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	entry := prefix + value
	for i, v := range env {
		if strings.HasPrefix(v, prefix) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}

// profileShell picks a POSIX-compatible interpreter to source a profile,
// derived from the declared "<shell>@<version>" metadata. Profiles are
// bash/POSIX scripts, so fish (and an unset shell) fall back to bash.
func profileShell(declared string) string {
	name := declared
	if i := strings.IndexByte(name, '@'); i >= 0 {
		name = name[:i]
	}
	switch name {
	case "bash", "zsh", "sh":
		return name
	default:
		return "bash"
	}
}

// singleQuote wraps s in single quotes for safe interpolation into a shell -c
// script, escaping any embedded single quotes.
func singleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
