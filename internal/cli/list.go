package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/env"
	"github.com/systemhalted/shellenv/internal/project"
	"github.com/systemhalted/shellenv/internal/registry"
)

var listAll bool

func init() {
	listCmd.Flags().BoolVar(&listAll, "all", false, "also list registered envs from other directories (NAME\\tSHELL\\tROOT)")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List project envs in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		if listAll {
			return listAllEnvs(cwd)
		}
		items, err := project.ListEnvs(cwd)
		if err != nil {
			return err
		}
		for _, n := range items {
			fmt.Println(n)
		}
		return nil
	},
}

// listAllEnvs prints every known env: registered ones (validated against
// metadata.json on disk, stale entries pruned) plus unregistered envs in the
// current directory, so `list --all` is a superset of `list` here.
func listAllEnvs(cwd string) error {
	seen := map[string]bool{}
	home, err := env.Home()
	if err == nil {
		reg, regErr := registry.Load(home)
		if regErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not read env registry: %v\n", regErr)
		}
		for _, e := range reg.Envs {
			md, mdErr := project.ReadMetadata(e.Root, e.Name)
			if os.IsNotExist(mdErr) {
				_ = registry.Remove(home, e.Root, e.Name) // stale: project is gone
				continue
			}
			if mdErr != nil {
				continue // unreadable right now; keep the entry, skip the row
			}
			fmt.Printf("%s\t%s\t%s\n", e.Name, md.Shell, e.Root)
			if e.Root == cwd {
				seen[e.Name] = true
			}
		}
	}

	// Envs here that predate the registry still deserve a row.
	items, err := project.ListEnvs(cwd)
	if err != nil {
		return err
	}
	for _, n := range items {
		if seen[n] {
			continue
		}
		md := loadMetadata(cwd, n)
		fmt.Printf("%s\t%s\t%s\n", n, md.Shell, cwd)
	}
	return nil
}
