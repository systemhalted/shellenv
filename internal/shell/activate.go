package shell

import (
	"fmt"
	"os"
	"path/filepath"
)

func ActivationCode(shellName, projectEnvDir, envName string) string {
	return ActivationCodeWithProfile(shellName, projectEnvDir, envName, "")
}

// ActivationCodeWithProfile emits shell activation snippet and, if profilePath is non-empty,
// sources it for bash/zsh/posix shells.
func ActivationCodeWithProfile(shellName, projectEnvDir, envName, profilePath string) string {
	s := detectShell(shellName)
	envBin := filepath.Join(projectEnvDir, "bin")

	if s == "fish" {
		// Fish: no POSIX 'source'; skip profile unless you add fish-specific scripts.
		return fmt.Sprintf(
			"set -gx SHELLENV_ACTIVE 1; set -gx SHELLENV_ENV_NAME %s; set -gx PATH %s $PATH; set -gx PS1 \"(shellenv:%s) $PS1\";",
			envName, envBin, envName,
		)
	}

	// bash/zsh/posix
	if profilePath != "" {
		// Source only if file exists (defensive)
		return fmt.Sprintf(
			"export SHELLENV_ACTIVE=1; export SHELLENV_ENV_NAME=%s; export PATH=%s:$PATH; export PS1=\"(shellenv:%s) ${PS1:-}\"; [ -f %q ] && . %q;",
			envName, envBin, envName, profilePath, profilePath,
		)
	}

	return fmt.Sprintf(
		"export SHELLENV_ACTIVE=1; export SHELLENV_ENV_NAME=%s; export PATH=%s:$PATH; export PS1=\"(shellenv:%s) ${PS1:-}\";",
		envName, envBin, envName,
	)
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
