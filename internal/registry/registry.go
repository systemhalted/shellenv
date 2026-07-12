// Package registry maintains an advisory index of project envs under
// $SHELLENV_HOME/registry.json so commands like uninstall can warn about envs
// beyond the current directory. It is best-effort by design: entries may be
// stale (projects move or vanish), writers are last-one-wins, and no caller
// may fail a user command on a registry error.
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Entry records one project env: where it lives and what it declared.
type Entry struct {
	Root       string `json:"root"`       // absolute project directory (holds ./.shellenv)
	Name       string `json:"name"`       // env name within the project
	Shell      string `json:"shell"`      // declared <shell>@<version>; display only, never trusted
	Registered string `json:"registered"` // RFC3339 timestamp of the last create
}

// File is the on-disk registry document.
type File struct {
	Version int     `json:"version"`
	Envs    []Entry `json:"envs"`
}

// Path returns the registry file location under the resolved SHELLENV_HOME.
func Path(home string) string {
	return filepath.Join(home, "registry.json")
}

// Load reads the registry. A missing file is an empty registry, not an error;
// a corrupt file loads as empty with the error returned so callers can warn
// and continue (the next Save rewrites a valid file).
func Load(home string) (File, error) {
	empty := File{Version: 1}
	b, err := os.ReadFile(Path(home))
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return empty, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return empty, err
	}
	if f.Version == 0 {
		f.Version = 1
	}
	return f, nil
}

// Save writes the registry atomically (temp file + rename, like the
// installer's .partial downloads) so a concurrent reader never sees a
// half-written document.
func Save(home string, f File) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := Path(home) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, Path(home)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Add upserts an entry keyed by (Root, Name), keeping the file sorted so
// diffs and listings are stable. A corrupt existing file is overwritten.
func Add(home string, e Entry) error {
	f, _ := Load(home) // corrupt → start from empty; Add repairs the file
	replaced := false
	for i, cur := range f.Envs {
		if cur.Root == e.Root && cur.Name == e.Name {
			f.Envs[i] = e
			replaced = true
			break
		}
	}
	if !replaced {
		f.Envs = append(f.Envs, e)
	}
	sort.Slice(f.Envs, func(i, j int) bool {
		if f.Envs[i].Root != f.Envs[j].Root {
			return f.Envs[i].Root < f.Envs[j].Root
		}
		return f.Envs[i].Name < f.Envs[j].Name
	})
	return Save(home, f)
}

// Remove drops the entry keyed by (root, name); absent entries and a missing
// registry are silent no-ops.
func Remove(home, root, name string) error {
	f, err := Load(home)
	if err != nil || len(f.Envs) == 0 {
		return nil // nothing trustworthy to rewrite
	}
	kept := f.Envs[:0]
	changed := false
	for _, e := range f.Envs {
		if e.Root == root && e.Name == name {
			changed = true
			continue
		}
		kept = append(kept, e)
	}
	if !changed {
		return nil
	}
	f.Envs = kept
	return Save(home, f)
}
