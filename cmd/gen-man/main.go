// Command gen-man generates man pages for shellenv from the live Cobra
// command tree, so the pages always match --help. It is a build-time tool:
// keeping it out of cmd/shellenv means its markdown/roff dependencies are
// never linked into the shipped binary.
//
// Usage: go run ./cmd/gen-man <output-dir>   (default: man)
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/systemhalted/shellenv/internal/cli"
)

func main() {
	dir := "man"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := run(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	root := cli.RootCmd()
	escapePlaceholders(root)
	header := &doc.GenManHeader{
		Title:   "SHELLENV",
		Section: "1",
		Source:  "shellenv",
		Manual:  "shellenv manual",
	}
	return doc.GenManTree(root, header, dir)
}

// escapePlaceholders backslash-escapes <env>-style placeholders in Use lines
// so md2man renders them literally instead of stripping them as HTML tags.
// Idempotent, so repeated runs (and tests) don't double-escape.
func escapePlaceholders(cmd *cobra.Command) {
	if !strings.Contains(cmd.Use, `\<`) {
		cmd.Use = strings.ReplaceAll(cmd.Use, "<", `\<`)
		cmd.Use = strings.ReplaceAll(cmd.Use, ">", `\>`)
	}
	for _, c := range cmd.Commands() {
		escapePlaceholders(c)
	}
}
