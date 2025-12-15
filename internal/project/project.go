package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Metadata struct {
	Name    string   `json:"name"`
	Shell   string   `json:"shell"`
	Tools   []string `json:"tools"`
	Profile string   `json:"profile"`
	Created string   `json:"created"`
}

func EnvDir(cwd, name string) string {
	return filepath.Join(cwd, ".shellenv", name)
}

func WriteMetadata(cwd string, md Metadata) error {
	dir := EnvDir(cwd, md.Name)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		return err
	}
	md.Created = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "metadata.json"), b, 0o644)
}

func ReadMetadata(cwd, name string) (Metadata, error) {
	var md Metadata
	p := filepath.Join(EnvDir(cwd, name), "metadata.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return md, err
	}
	if err := json.Unmarshal(b, &md); err != nil {
		return md, err
	}
	return md, nil
}

func WriteCurrent(cwd, name string) error {
	dir := filepath.Join(cwd, ".shellenv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "current"), []byte(name+"\n"), 0o644)
}

func ReadCurrent(cwd string) (string, error) {
	b, err := os.ReadFile(filepath.Join(cwd, ".shellenv", "current"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func ListEnvs(cwd string) ([]string, error) {
	base := filepath.Join(cwd, ".shellenv")
	ents, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
