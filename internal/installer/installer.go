// Package installer downloads, verifies, and builds pinned shell runtimes
// from their official source tarballs into $SHELLENV_HOME/installs.
package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Source describes where a shell version's source tarball lives.
type Source struct {
	URL    string
	SHA256 string // empty when the version has no pinned checksum
}

// pinnedChecksums holds SHA-256 digests of the official tarballs for
// well-known versions, computed from the upstream artifacts. Versions not
// listed here still install, with a loud warning that the download is
// unverified.
var pinnedChecksums = map[string]string{
	"bash@5.2": "a139c166df7ff4471c5e0733051642ee5556c1cc8a4a78f145583c5c81ab32fb",
	"zsh@5.9":  "9b8d1ecedd5b5e81fbf1918e876752a7dd948e05c1a0dba10ab863842d45acd5",
}

// SourceFor maps a shell name and version to its official source tarball.
// zsh downloads come from the SourceForge mirror — zsh.org/pub returns 404
// for direct tarball URLs.
func SourceFor(shell, version string) (Source, error) {
	var url string
	switch shell {
	case "bash":
		url = fmt.Sprintf("https://ftp.gnu.org/gnu/bash/bash-%s.tar.gz", version)
	case "zsh":
		url = fmt.Sprintf("https://downloads.sourceforge.net/project/zsh/zsh/%s/zsh-%s.tar.xz", version, version)
	default:
		return Source{}, fmt.Errorf("no installer for shell %q (supported: bash, zsh)", shell)
	}
	return Source{URL: url, SHA256: pinnedChecksums[shell+"@"+version]}, nil
}

// Installer performs download → verify → extract → build → install. Its
// externals (process execution, HTTP download, PATH lookup) are fields so
// tests can stub them.
type Installer struct {
	Home     string    // resolved $SHELLENV_HOME
	Out      io.Writer // progress and warnings
	Run      func(dir string, logTo *os.File, name string, args ...string) error
	Fetch    func(url, dest string) error
	LookPath func(file string) (string, error)

	// RequireChecksum refuses to install versions without a pinned checksum
	// instead of warning (the strict opt-in, mirroring --strict-shell).
	RequireChecksum bool

	pinned map[string]string // overridable in tests; nil means pinnedChecksums
}

func New(home string, out io.Writer) *Installer {
	return &Installer{
		Home:     home,
		Out:      out,
		Run:      runCommand,
		Fetch:    fetchURL,
		LookPath: exec.LookPath,
	}
}

func (in *Installer) prefix(shell, version string) string {
	return filepath.Join(in.Home, "installs", shell, version)
}

// Install provisions shell@version and returns its install prefix. Already
// installed runtimes (prefix/bin/<shell> exists) are left untouched.
func (in *Installer) Install(shell, version string) (string, error) {
	src, err := SourceFor(shell, version)
	if err != nil {
		return "", err
	}
	if in.pinned != nil {
		src.SHA256 = in.pinned[shell+"@"+version]
	}

	prefix := in.prefix(shell, version)
	if _, err := os.Stat(filepath.Join(prefix, "bin", shell)); err == nil {
		fmt.Fprintf(in.Out, "%s@%s is already installed at %s\n", shell, version, prefix)
		return prefix, nil
	}

	// Refuse before preflight and download: don't fetch what we won't trust.
	if src.SHA256 == "" && in.RequireChecksum {
		return "", fmt.Errorf("no pinned checksum for %s@%s; refusing unverified install (omit --require-checksum to proceed with a warning)", shell, version)
	}

	if err := in.preflight(); err != nil {
		return "", err
	}

	archive, err := in.download(src)
	if err != nil {
		return "", err
	}

	buildRoot := filepath.Join(in.Home, "tmp", fmt.Sprintf("build-%s-%s", shell, version))
	if err := os.RemoveAll(buildRoot); err != nil {
		return "", err
	}
	if err := os.MkdirAll(buildRoot, 0o755); err != nil {
		return "", err
	}
	logPath := filepath.Join(buildRoot, "build.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return "", err
	}
	defer logFile.Close()

	fmt.Fprintf(in.Out, "Extracting %s...\n", filepath.Base(archive))
	if err := in.Run(buildRoot, logFile, "tar", "-xf", archive); err != nil {
		return "", fmt.Errorf("extracting %s failed: %w (see %s)", archive, err, logPath)
	}
	srcDir, err := extractedDir(buildRoot)
	if err != nil {
		return "", err
	}

	steps := [][]string{
		{"./configure", "--prefix=" + prefix},
		{"make", "-j" + strconv.Itoa(runtime.NumCPU())},
		{"make", "install"},
	}
	for _, step := range steps {
		fmt.Fprintf(in.Out, "Running %s...\n", strings.Join(step, " "))
		if err := in.Run(srcDir, logFile, step[0], step[1:]...); err != nil {
			// Keep the build dir so the log can be inspected.
			return "", fmt.Errorf("%s failed for %s@%s: %w (full output in %s)",
				strings.Join(step, " "), shell, version, err, logPath)
		}
	}

	marker := fmt.Sprintf("source: %s\nsha256: %s\n", src.URL, src.SHA256)
	if err := os.WriteFile(filepath.Join(prefix, "installed.txt"), []byte(marker), 0o644); err != nil {
		return "", err
	}
	if err := os.RemoveAll(buildRoot); err != nil {
		return "", err
	}
	fmt.Fprintf(in.Out, "Installed %s@%s into %s\n", shell, version, prefix)
	return prefix, nil
}

// preflight verifies the build toolchain is available before any download.
func (in *Installer) preflight() error {
	var missing []string
	if _, err := in.LookPath("cc"); err != nil {
		if _, err := in.LookPath("gcc"); err != nil {
			missing = append(missing, "cc/gcc")
		}
	}
	for _, tool := range []string{"make", "tar"} {
		if _, err := in.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("building shells from source requires %s on PATH (install your distro's build tools, e.g. build-essential)",
			strings.Join(missing, ", "))
	}
	return nil
}

// download fetches the tarball into the cache (reusing a prior download) and
// verifies it against the pinned checksum when one exists.
func (in *Installer) download(src Source) (string, error) {
	cacheDir := filepath.Join(in.Home, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(cacheDir, filepath.Base(src.URL))
	if _, err := os.Stat(dest); err != nil {
		fmt.Fprintf(in.Out, "Downloading %s...\n", src.URL)
		if err := in.Fetch(src.URL, dest); err != nil {
			return "", fmt.Errorf("downloading %s: %w", src.URL, err)
		}
	}
	if src.SHA256 == "" {
		fmt.Fprintf(in.Out, "warning: no pinned checksum for this version; the download is unverified (use --require-checksum to refuse instead)\n")
		return dest, nil
	}
	sum, err := fileSHA256(dest)
	if err != nil {
		return "", err
	}
	if sum != src.SHA256 {
		_ = os.Remove(dest)
		return "", fmt.Errorf("checksum mismatch for %s: got %s, want %s (corrupt download removed; retry)",
			filepath.Base(dest), sum, src.SHA256)
	}
	return dest, nil
}

// extractedDir returns the single directory the tarball unpacked into.
func extractedDir(buildRoot string) (string, error) {
	ents, err := os.ReadDir(buildRoot)
	if err != nil {
		return "", err
	}
	var dirs []string
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("expected one source directory in %s, found %d", buildRoot, len(dirs))
	}
	return filepath.Join(buildRoot, dirs[0]), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// runCommand executes name in dir, sending combined output to the log file.
func runCommand(dir string, logTo *os.File, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = logTo
	cmd.Stderr = logTo
	return cmd.Run()
}

// fetchURL downloads url to dest atomically (via a .partial file).
func fetchURL(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	partial := dest + ".partial"
	f, err := os.Create(partial)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		_ = os.Remove(partial)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(partial, dest)
}
