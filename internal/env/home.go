package env

import (
	"fmt"
	"os"
	"path/filepath"
)

const DefaultHomeDirName = ".shellenv"

func Home() (string, error) {
	if h := os.Getenv("SHELLENV_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve user home: %w", err)
	}
	return filepath.Join(home, DefaultHomeDirName), nil
}

func EnsureHome() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	subdirs := []string{"installs", "cache", "tmp"}
	for _, sd := range subdirs {
		if err := os.MkdirAll(filepath.Join(h, sd), 0o755); err != nil {
			return "", err
		}
	}
	return h, nil
}

func InstallsDir() (string, error) {
	h, err := EnsureHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "installs"), nil
}
