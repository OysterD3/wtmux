package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
)

// Execute is the entry point invoked from cmd/wtmux/main.go.
// Returns the process exit code.
func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
		return wtmuxerrors.ExitCodeFor(err)
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "wtmux",
		Short:         "Coordinated git worktrees across sibling repos for AI agents",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version(),
		Args:          userArgs(cobra.NoArgs),
		Run: func(c *cobra.Command, _ []string) {
			helpToStdout(rootHelp)
		},
	}
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpToStdout(rootHelp)
	})
	// Tag flag-parsing failures (unknown flag, missing flag value, etc.)
	// as KindUser so they exit 1 instead of falling through to ExitCodeFor's
	// default of 3. cobra propagates this hook to subcommands.
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return wtmuxerrors.Wrapf(wtmuxerrors.KindUser, err, "%s", err.Error())
	})

	cmd.PersistentFlags().StringVarP(&pf.configPath, "config", "c", "", "Override config discovery")
	cmd.PersistentFlags().StringVarP(&pf.group, "group", "g", "", "Override auto-detected group")
	cmd.PersistentFlags().BoolVarP(&pf.verbose, "verbose", "v", false, "Extra logging")

	cmd.AddCommand(newNewCmd())
	cmd.AddCommand(newLsCmd())
	cmd.AddCommand(newRmCmd())
	cmd.AddCommand(newConfigCmd())
	return cmd
}
