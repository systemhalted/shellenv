package shell

import (
	"os"
	"path/filepath"
)

// ResolveProfile tries multiple locations for <profile>.sh.
// Returns (absolutePath, true) if found, else ("", false).
func ResolveProfile(cwd, profile string) (string, bool) {
	return ResolveProfileForShell(cwd, profile, "")
}

// ResolveProfileForShell resolves the profile variant for the given shell
// type: fish looks for <profile>.fish (fish cannot source POSIX scripts, so
// there is no fallback to .sh); everything else uses <profile>.sh.
func ResolveProfileForShell(cwd, profile, shellType string) (string, bool) {
	if profile == "" {
		return "", false
	}
	name := profile + ".sh"
	if shellType == "fish" {
		name = profile + ".fish"
	}

	// 1) SHELLENV_PROFILES override
	if base := os.Getenv("SHELLENV_PROFILES"); base != "" {
		if p := filepath.Join(base, name); fileExists(p) {
			return filepath.Clean(p), true
		}
	}

	// 2) <project>/profiles/<profile>.sh
	if cwd != "" {
		if p := filepath.Join(cwd, "profiles", name); fileExists(p) {
			return filepath.Clean(p), true
		}
	}

	// 3) near the executable: dist/../profiles/ and dist/profiles/
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		cands := []string{
			filepath.Join(exeDir, "..", "profiles", name),
			filepath.Join(exeDir, "profiles", name),
		}
		for _, p := range cands {
			if fileExists(p) {
				return filepath.Clean(p), true
			}
		}
	}
	return "", false
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
