package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/shell"
)

var (
	execWithProfile bool
	execContainer   string
	execStrictShell bool
	execEphemeral   bool
)

func init() {
	execCmd.Flags().BoolVar(&execWithProfile, "profile", false,
		"source the env's declared profile (e.g. strict) before running the command")
	execCmd.Flags().StringVar(&execContainer, "container", "",
		"run the command inside the specified container image (e.g., ubuntu:22.04)")
	execCmd.Flags().BoolVar(&execStrictShell, "strict-shell", false,
		"require the declared shell runtime to be installed; fail instead of falling back to the system shell")
	execCmd.Flags().BoolVar(&execEphemeral, "ephemeral", false,
		"use a throwaway sandbox home that is removed after the command exits")
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

		md := loadMetadata(cwd, envName)

		// Resolve the declared shell runtime (host mode only: an installs
		// path from the host is meaningless inside a container image).
		var runtimeBin string
		if execContainer != "" {
			if execStrictShell {
				return fmt.Errorf("--strict-shell is not supported with --container (declared shell resolution is host-only; pick an image that provides the shell)")
			}
		} else {
			rb, status, err := env.ResolveRuntime(md.Shell)
			if err != nil {
				return err
			}
			switch status {
			case env.RuntimeFound:
				runtimeBin = rb
			case env.RuntimeMissing:
				if execStrictShell {
					return fmt.Errorf("declared shell %q is not installed (run 'shellenv install %s', or omit --strict-shell)", md.Shell, md.Shell)
				}
				fmt.Fprintf(os.Stderr, "warning: declared shell %q is not installed; falling back to the system shell (run 'shellenv install %s' or use --strict-shell to enforce)\n", md.Shell, md.Shell)
			case env.RuntimeUnpinned:
				if execStrictShell {
					return fmt.Errorf("env %q declares no <shell>@<version> to enforce (shell: %q); --strict-shell has nothing to pin", envName, md.Shell)
				}
			}
		}

		binDir := filepath.Join(envDir, "bin")
		// Env bin must stay first so project-local tools keep winning; the
		// pinned runtime sits between it and the system PATH.
		childEnv := prependPath(prependPath(os.Environ(), runtimeBin), binDir)
		childEnv = upsertEnv(childEnv, "SHELLENV_ENV_NAME", envName)
		childEnv = upsertEnv(childEnv, "SHELLENV_ACTIVE", "1")

		// Isolate HOME/TMPDIR/XDG so scripts write to a per-env sandbox rather
		// than the user's real home or system temp dir. With --ephemeral the
		// sandbox is a throwaway dir removed after the child exits; it lives
		// under the env dir (not the system temp) so it stays inside the
		// workspace mount in container mode.
		sandboxHome := project.SandboxHomeDir(cwd, envName)
		if execEphemeral {
			eph, err := os.MkdirTemp(envDir, "home-ephemeral-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(eph)
			sandboxHome = eph
		}
		sb, err := project.EnsureSandboxDirs(sandboxHome)
		if err != nil {
			return err
		}
		childEnv = upsertEnv(childEnv, "HOME", sb.Home)
		childEnv = upsertEnv(childEnv, "TMPDIR", sb.Tmp)
		childEnv = upsertEnv(childEnv, "XDG_CONFIG_HOME", sb.XDGConfig)
		childEnv = upsertEnv(childEnv, "XDG_CACHE_HOME", sb.XDGCache)
		childEnv = upsertEnv(childEnv, "XDG_DATA_HOME", sb.XDGData)

		var child *exec.Cmd
		if execContainer != "" {
			engine, err := findContainerEngine()
			if err != nil {
				return err
			}
			var containerArgs []string
			containerArgs = append(containerArgs, "run", "--rm")
			if isTTY(os.Stdin.Fd()) && isTTY(os.Stdout.Fd()) {
				containerArgs = append(containerArgs, "-i", "-t")
			} else {
				containerArgs = append(containerArgs, "-i")
			}
			containerArgs = append(containerArgs, "-v", fmt.Sprintf("%s:%s", cwd, cwd))
			containerArgs = append(containerArgs, "-w", cwd)
			for _, envVar := range childEnv {
				if strings.HasPrefix(envVar, "PATH=") {
					continue
				}
				containerArgs = append(containerArgs, "-e", envVar)
			}
			containerArgs = append(containerArgs, execContainer)

			var shellScript strings.Builder
			shellScript.WriteString(fmt.Sprintf("export PATH=%s:$PATH; ", singleQuote(binDir)))
			if execWithProfile {
				profilePath, ok := shell.ResolveProfile(cwd, md.Profile)
				if !ok {
					return fmt.Errorf("profile %q not found for env %q (looked in SHELLENV_PROFILES, ./profiles, and beside the binary)", md.Profile, envName)
				}
				shellScript.WriteString(fmt.Sprintf("[ -f %s ] && . %s; ", singleQuote(profilePath), singleQuote(profilePath)))
			}
			shellScript.WriteString(`exec "$@"`)

			containerArgs = append(containerArgs, "sh", "-c", shellScript.String(), "--")
			containerArgs = append(containerArgs, cmdArgs...)
			child = exec.Command(engine, containerArgs...)
		} else if execWithProfile {
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
		if execContainer == "" {
			child.Env = childEnv
		}
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

func findContainerEngine() (string, error) {
	for _, engine := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(engine); err == nil {
			return engine, nil
		}
	}
	return "", fmt.Errorf("container engine not found (neither 'docker' nor 'podman' is on your PATH)")
}
