package project

import (
    "encoding/json"
    "os"
    "path/filepath"
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
