package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/group"
	wtlog "github.com/OysterD3/wtmux/internal/log"
)

type rmFlags struct {
	dryRun bool
	force  bool
}

func newRmCmd() *cobra.Command {
	var flags rmFlags
	cmd := &cobra.Command{
		Use:           "rm <name>",
		Short:         "Remove coordinated worktrees",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyVerbose()
			return runRm(args[0], flags)
		},
	}
	cmd.Flags().BoolVarP(&flags.dryRun, "dry-run", "n", false, "Plan without mutating")
	cmd.Flags().BoolVarP(&flags.force, "force", "f", false, "Skip dirty/unpushed guards")
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpToStdout(rmHelp)
	})
	return cmd
}

func runRm(name string, flags rmFlags) error {
	cwd, err := mustGetwd()
	if err != nil {
		return err
	}
	cfg, err := loadConfigOrDefault(cwd)
	if err != nil {
		return err
	}

	resolved, err := group.Resolve(group.ResolveInput{
		Cwd:       cwd,
		Config:    cfg,
		GroupFlag: pf.group,
	})
	if err != nil {
		return wtmuxerrors.New(wtmuxerrors.KindUser, "%s", err.Error())
	}
	if resolved.Kind == group.KindOutside {
		return wtmuxerrors.New(wtmuxerrors.KindPrecondition, "cwd is not inside any git repository")
	}

	var repos []string
	if resolved.Kind == group.KindGroup {
		repos = append(repos, resolved.Group.Repos...)
	} else {
		repos = append(repos, resolved.Single)
	}

	for _, repo := range repos {
		wtPath, err := findWorktreeByBranch(repo, name)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		if wtPath == "" {
			continue
		}

		if flags.dryRun {
			wtlog.Infof("dry-run: would remove %s in %s", wtPath, repo)
			continue
		}

		if !flags.force {
			status, err := git.StatusPorcelain(wtPath)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			if status != "" {
				wtlog.Warnf("skipping %s — uncommitted changes", wtPath)
				fmt.Fprintf(os.Stderr, "skipped %s (dirty)\n", wtPath)
				continue
			}
			unpushed, err := git.UnpushedCommits(wtPath)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			if len(unpushed) > 0 {
				wtlog.Warnf("skipping %s — %d unpushed commit(s)", wtPath, len(unpushed))
				fmt.Fprintf(os.Stderr, "skipped %s (unpushed)\n", wtPath)
				continue
			}
		}

		if flags.force {
			if err := git.WorktreeRemoveForce(repo, wtPath); err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
		} else {
			if err := git.WorktreeRemove(repo, wtPath); err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
		}
		fmt.Fprintf(os.Stdout, "removed %s\n", wtPath)

		var delErr error
		if flags.force {
			delErr = git.DeleteBranchForce(repo, name)
		} else {
			delErr = git.DeleteBranch(repo, name)
		}
		if delErr != nil {
			wtlog.Infof("could not delete branch %q in %s: %s", name, repo, delErr.Error())
		}

		if err := git.WorktreePrune(repo); err != nil {
			wtlog.Debugf("worktree prune in %s: %s", repo, err.Error())
		}
	}

	return nil
}

// findWorktreeByBranch returns the path of repo's linked worktree checked
// out on branch, or "" if no such worktree exists. The repo's main
// worktree is excluded so `rm <main-branch>` never tries to remove the
// repo root.
func findWorktreeByBranch(repo, branch string) (string, error) {
	wts, err := git.ListWorktrees(repo)
	if err != nil {
		return "", err
	}
	for _, w := range wts {
		if w.Branch == branch && w.Worktree != repo {
			return w.Worktree, nil
		}
	}
	return "", nil
}
