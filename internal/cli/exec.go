package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/example/shellenv/internal/project"
	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "exec [<env>] -- <cmd>",
		Short: "Run a command within a project env without interactive activation",
		Ru
