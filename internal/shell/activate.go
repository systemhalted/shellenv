package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/systemhalted/shellenv/internal/project"
)

// ActivationOptions carries the optional pieces of an activation snippet.
type ActivationOptions struct {
	// ProfilePath, when non-empty, is sourced (bash/zsh/posix source the
	// .sh profile; fish sources a .fish variant the caller resolved).
	ProfilePath string
	// RuntimeBinDir, when non-empty, is the pinned shell runtime's bin dir;
	// it goes on PATH after the env bin so project-local tools keep winning.
	RuntimeBinDir string
	// Sandbox, when non-nil, redirects HOME/TMPDIR/XDG_* at the sandbox
	// (activate --isolate-home). The caller must have created the dirs —
	// the snippet contains no mkdir so stdout stays minimal shell code.
	Sandbox *project.SandboxPaths
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
// profile sourcing, pinned-runtime PATH entries, and sandbox isolation
// applied. Saved SHELLENV_OLD_* values are write-once so a double activation
// still restores the pre-first-activation state on deactivate.
func ActivationCodeWithOptions(shellName, projectEnvDir, envName string, opts ActivationOptions) string {
	s := detectShell(shellName)
	envBin := filepath.Join(projectEnvDir, "bin")

	if s == "fish" {
		var b strings.Builder
		b.WriteString("set -q SHELLENV_OLD_PATH; or set -gx SHELLENV_OLD_PATH $PATH; ")
		b.WriteString("set -q SHELLENV_OLD_PS1; or set -gx SHELLENV_OLD_PS1 \"$PS1\"; ")
		if sb := opts.Sandbox; sb != nil {
			for _, v := range sandboxVars(sb) {
				b.WriteString(fmt.Sprintf("set -q SHELLENV_OLD_%s; or set -gx SHELLENV_OLD_%s \"$%s\"; set -gx %s %s; ",
					v.name, v.name, v.name, v.name, v.dir))
			}
		}
		pathDirs := envBin
		if opts.RuntimeBinDir != "" {
			pathDirs += " " + opts.RuntimeBinDir
		}
		b.WriteString(fmt.Sprintf(
			"set -gx SHELLENV_ACTIVE 1; set -gx SHELLENV_ENV_NAME %s; set -gx PATH %s $PATH; set -gx PS1 \"(shellenv:%s) $PS1\";",
			envName, pathDirs, envName,
		))
		// Only a fish-syntax profile may be sourced here; the caller resolves
		// the .fish variant (ResolveProfileForShell) and omits ProfilePath
		// when none exists.
		if opts.ProfilePath != "" {
			b.WriteString(fmt.Sprintf(" test -f %s; and source %s;", opts.ProfilePath, opts.ProfilePath))
		}
		return b.String()
	}

	// bash/zsh/posix
	var b strings.Builder
	b.WriteString(`[ -n "${SHELLENV_OLD_PATH+x}" ] || export SHELLENV_OLD_PATH="$PATH"; `)
	b.WriteString(`[ -n "${SHELLENV_OLD_PS1+x}" ] || export SHELLENV_OLD_PS1="${PS1-}"; `)
	if sb := opts.Sandbox; sb != nil {
		for _, v := range sandboxVars(sb) {
			b.WriteString(fmt.Sprintf(`[ -n "${SHELLENV_OLD_%s+x}" ] || export SHELLENV_OLD_%s="${%s-}"; export %s=%s; `,
				v.name, v.name, v.name, v.name, v.dir))
		}
	}

	pathDirs := envBin
	if opts.RuntimeBinDir != "" {
		pathDirs += ":" + opts.RuntimeBinDir
	}
	b.WriteString(fmt.Sprintf(
		"export SHELLENV_ACTIVE=1; export SHELLENV_ENV_NAME=%s; export PATH=%s:$PATH; export PS1=\"(shellenv:%s) ${PS1:-}\";",
		envName, pathDirs, envName,
	))
	if opts.ProfilePath != "" {
		// Source only if file exists (defensive)
		b.WriteString(fmt.Sprintf(" [ -f %q ] && . %q;", opts.ProfilePath, opts.ProfilePath))
	}
	return b.String()
}

// sandboxVars pairs each isolated environment variable with its sandbox
// location, in a stable order shared by activation and documentation.
func sandboxVars(sb *project.SandboxPaths) []struct{ name, dir string } {
	return []struct{ name, dir string }{
		{"HOME", sb.Home},
		{"TMPDIR", sb.Tmp},
		{"XDG_CONFIG_HOME", sb.XDGConfig},
		{"XDG_CACHE_HOME", sb.XDGCache},
		{"XDG_DATA_HOME", sb.XDGData},
	}
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
