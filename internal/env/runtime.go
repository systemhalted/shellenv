package env

import (
	"os"
	"path/filepath"
	"strings"
)

// RuntimeStatus reports whether a declared shell runtime could be resolved
// under $SHELLENV_HOME/installs.
type RuntimeStatus int

const (
	// RuntimeUnpinned means no "<shell>@<version>" was declared, so there is
	// nothing to resolve; commands fall through to the system shell.
	RuntimeUnpinned RuntimeStatus = iota
	// RuntimeMissing means a version was declared but its installs bin
	// directory does not exist.
	RuntimeMissing
	// RuntimeFound means the declared runtime's bin directory exists and
	// should be prepended to PATH.
	RuntimeFound
)

// ParseShellVersion splits a declared "<shell>@<version>" pair such as
// "bash@5.2". ok is false when either side is empty or there is no '@',
// meaning the declaration does not pin a version.
func ParseShellVersion(declared string) (shell, version string, ok bool) {
	i := strings.IndexByte(declared, '@')
	if i <= 0 || i == len(declared)-1 {
		return "", "", false
	}
	return declared[:i], declared[i+1:], true
}

// ResolveRuntime maps a declared "<shell>@<version>" to its installed bin
// directory under $SHELLENV_HOME/installs/<shell>/<version>/bin. A version
// directory without a bin/ counts as missing (placeholder installs create
// bin/, so absence means the runtime was never installed here).
func ResolveRuntime(declared string) (binDir string, status RuntimeStatus, err error) {
	shell, version, ok := ParseShellVersion(declared)
	if !ok {
		return "", RuntimeUnpinned, nil
	}
	inst, err := InstallsDir()
	if err != nil {
		return "", RuntimeMissing, err
	}
	dir := filepath.Join(inst, shell, version, "bin")
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return "", RuntimeMissing, nil
	}
	return dir, RuntimeFound, nil
}
