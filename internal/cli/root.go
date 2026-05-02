package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Execute is the entry point invoked from cmd/wtmux/main.go.
// Returns the process exit code.
func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wtmux",
		Short:         "Coordinated git worktrees across sibling repos for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	cmd.SetVersionTemplate("{{.Version}}\n")
	return cmd
}
