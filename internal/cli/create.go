package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/systemhalted/shellenv/internal/project"
)

var (
	createName    string
	createShell   string
	createProfile string
)

func init() {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a project environment in ./.shellenv/<name>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if createShell == "" {
				return fmt.Errorf("--shell is required (e.g., --shell bash@5.2)")
			}
			cwd, _ := os.Getwd()
			if createName == "" {
				createName = "default"
			}
			md := project.Metadata{
				Name:    createName,
				Shell:   createShell,
				Profile: createProfile,
			}
			if err := project.WriteMetadata(cwd, md); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(project.EnvDir(cwd, createName), "activate.sh"), []byte("eval \"$(shellenv activate)\"\n"), 0o755); err != nil {
				return fmt.Errorf("writing activate.sh: %w", err)
			}
			fmt.Printf("Created env %q for shell %s at %s\n", createName, createShell, project.EnvDir(cwd, createName))
			return nil
		},
	}
	c.Flags().StringVar(&createName, "name", "", "name of the environment (default: default)")
	c.Flags().StringVar(&createShell, "shell", "", "shell runtime to use, e.g., bash@5.2 (required)")
	c.Flags().StringVar(&createProfile, "profile", "strict", "option profile: strict|posix|interactive")
	rootCmd.AddCommand(c)
}
