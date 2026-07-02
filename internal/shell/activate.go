package shell

import (
	"fmt"
	"os"
	"path/filepath"
)

// ActivationOptions carries the optional pieces of an activation snippet.
type ActivationOptions struct {
	// ProfilePath, when non-empty, is sourced for bash/zsh/posix shells.
	ProfilePath string
	// RuntimeBinDir, when non-empty, is the pinned shell runtime's bin dir;
	// it goes on PATH after the env bin so project-local tools keep winning.
	RuntimeBinDir string
}

func ActivationCode(shellName, projectEnvDir, envName string) string {
	return ActivationCodeWithOptions(shellName, projectEnvDir, envName, ActivationOptions{})
}

// ActivationCodeWithProfile emits shell activation snippet and, if profilePath is non-empty,
// sources it for bash/zsh/posix shells.
func ActivationCodeWithProfile(shellName, projectEnvDir, envName, profilePath string) string {
	return ActivationCodeWithOptions(shellName, projectEnvDir, envName, ActivationOptions{ProfilePath: profilePath})
}

// ActivationCodeWithOptions emits the activation snippet with any optional
// profile sourcing and pinned-runtime PATH entries applied.
func ActivationCodeWithOptions(shellName, projectEnvDir, envName string, opts ActivationOptions) string {
	s := detectShell(shellName)
	envBin := filepath.Join(projectEnvDir, "bin")

	if s == "fish" {
		pathDirs := envBin
		if opts.RuntimeBinDir != "" {
			pathDirs += " " + opts.RuntimeBinDir
		}
		code := fmt.Sprintf(
			"set -gx SHELLENV_ACTIVE 1; set -gx SHELLENV_ENV_NAME %s; set -gx PATH %s $PATH; set -gx PS1 \"(shellenv:%s) $PS1\";",
			envName, pathDirs, envName,
		)
		// Only a fish-syntax profile may be sourced here; the caller resolves
		// the .fish variant (ResolveProfileForShell) and omits ProfilePath
		// when none exists.
		if opts.ProfilePath != "" {
			code += fmt.Sprintf(" test -f %s; and source %s;", opts.ProfilePath, opts.ProfilePath)
		}
		return code
	}

	pathDirs := envBin
	if opts.RuntimeBinDir != "" {
		pathDirs += ":" + opts.RuntimeBinDir
	}

	// bash/zsh/posix
	if opts.ProfilePath != "" {
		// Source only if file exists (defensive)
		return fmt.Sprintf(
			"export SHELLENV_ACTIVE=1; export SHELLENV_ENV_NAME=%s; export PATH=%s:$PATH; export PS1=\"(shellenv:%s) ${PS1:-}\"; [ -f %q ] && . %q;",
			envName, pathDirs, envName, opts.ProfilePath, opts.ProfilePath,
		)
	}

	return fmt.Sprintf(
		"export SHELLENV_ACTIVE=1; export SHELLENV_ENV_NAME=%s; export PATH=%s:$PATH; export PS1=\"(shellenv:%s) ${PS1:-}\";",
		envName, pathDirs, envName,
	)
}

// DetectShell reports the shell type an activation snippet will target:
// the explicit input if given, else the basename of $SHELL, else bash.
func DetectShell(input string) string {
	return detectShell(input)
}

func detectShell(input string) string {
	if input != "" {
		return input
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		for i := len(sh) - 1; i >= 0; i-- {
			if sh[i] == '/' {
				return sh[i+1:]
			}
		}
		return sh
	}
	return "bash"
}
