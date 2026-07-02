package shell

import (
	"fmt"
	"strings"
)

// isolatedVars are the variables activate may have redirected; deactivate
// restores each only when its SHELLENV_OLD_* save exists.
var isolatedVars = []string{"HOME", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME"}

// DeactivationCode emits shell code restoring a session activated by
// `activate` (with or without --isolate-home). Every restore is guarded on
// its SHELLENV_OLD_* save being set, so the snippet is a silent success
// (even under `set -eu`) when nothing is active. A saved-but-empty value
// restores to unset — for HOME/TMPDIR/XDG_* consumers that is equivalent.
// Shell options changed by a sourced profile (set -e, set -o posix) cannot
// be restored from outside the shell and are left as-is.
func DeactivationCode(shellName string) string {
	if detectShell(shellName) == "fish" {
		var b strings.Builder
		b.WriteString("set -q SHELLENV_OLD_PATH; and set -gx PATH $SHELLENV_OLD_PATH; ")
		b.WriteString("set -q SHELLENV_OLD_PS1; and set -gx PS1 \"$SHELLENV_OLD_PS1\"; ")
		for _, v := range isolatedVars {
			b.WriteString(fmt.Sprintf("if set -q SHELLENV_OLD_%s; if test -n \"$SHELLENV_OLD_%s\"; set -gx %s \"$SHELLENV_OLD_%s\"; else; set -e %s; end; end; ",
				v, v, v, v, v))
		}
		for _, v := range append([]string{"ACTIVE", "ENV_NAME", "OLD_PATH", "OLD_PS1"}, prefixed("OLD_", isolatedVars)...) {
			b.WriteString(fmt.Sprintf("set -q SHELLENV_%s; and set -e SHELLENV_%s; ", v, v))
		}
		return strings.TrimSpace(b.String())
	}

	// bash/zsh/posix
	var b strings.Builder
	b.WriteString(`if [ -n "${SHELLENV_OLD_PATH+x}" ]; then export PATH="$SHELLENV_OLD_PATH"; fi; `)
	// PS1 is restored by assignment, not export — matching how it normally lives.
	b.WriteString(`if [ -n "${SHELLENV_OLD_PS1+x}" ]; then PS1="$SHELLENV_OLD_PS1"; fi; `)
	for _, v := range isolatedVars {
		b.WriteString(fmt.Sprintf(`if [ -n "${SHELLENV_OLD_%s+x}" ]; then if [ -n "$SHELLENV_OLD_%s" ]; then export %s="$SHELLENV_OLD_%s"; else unset %s; fi; fi; `,
			v, v, v, v, v))
	}
	b.WriteString("unset SHELLENV_ACTIVE SHELLENV_ENV_NAME SHELLENV_OLD_PATH SHELLENV_OLD_PS1")
	for _, v := range isolatedVars {
		b.WriteString(" SHELLENV_OLD_" + v)
	}
	b.WriteString(";")
	return b.String()
}

func prefixed(prefix string, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = prefix + n
	}
	return out
}
