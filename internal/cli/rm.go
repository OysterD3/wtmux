package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/group"
	wtlog "github.com/OysterD3/wtmux/internal/log"
	"github.com/OysterD3/wtmux/internal/paths"
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
		Args:          userArgs(cobra.ExactArgs(1)),
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
		return wtmuxerrors.Wrapf(wtmuxerrors.KindUser, err, "%s", err.Error())
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

	type rmTarget struct {
		repo, wtPath string
	}

	// Pass 1: discover every linked worktree on `name` across the group.
	var targets []rmTarget
	for _, repo := range repos {
		wtPath, err := findWorktreeByBranch(repo, name)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		if wtPath != "" {
			targets = append(targets, rmTarget{repo: repo, wtPath: wtPath})
		}
	}

	if len(targets) == 0 {
		// Stay exit 0 so idempotent cleanup scripts (`wtmux rm $name` on a
		// branch already gone) don't trip, but surface a stderr notice so
		// typos are visible.
		wtlog.Infof("no coordinated worktrees named %q in this group", name)
		return nil
	}

	if flags.dryRun {
		for _, t := range targets {
			wtlog.Infof("dry-run: would remove %s in %s", t.wtPath, t.repo)
		}
		return nil
	}

	// Pass 2: precheck. If any target is dirty or unpushed and --force is
	// not set, refuse the entire operation. Removing only the clean repos
	// would break the coordinated-worktree invariant the README promises.
	if !flags.force {
		var blockers []string
		for _, t := range targets {
			status, err := git.StatusPorcelain(t.wtPath)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			if status != "" {
				blockers = append(blockers, fmt.Sprintf("  dirty:    %s", t.wtPath))
				continue
			}
			unpushed, err := git.UnpushedCommits(t.wtPath)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			if len(unpushed) > 0 {
				blockers = append(blockers, fmt.Sprintf("  unpushed: %s (%d commit(s))", t.wtPath, len(unpushed)))
			}
		}
		if len(blockers) > 0 {
			return wtmuxerrors.New(
				wtmuxerrors.KindPrecondition,
				"refusing to remove %q — coordinated worktrees have uncommitted work:\n%s\ncommit/stash/push, or pass --force to remove anyway",
				name, strings.Join(blockers, "\n"),
			)
		}
	}

	// Pass 3: every target passed (or --force). Remove them all. A git
	// failure here aborts mid-loop and may leave the group partially
	// removed, but that's a real-error condition rather than a user-misuse
	// one — there's no clean rollback for `git worktree remove`.
	for _, t := range targets {
		if flags.force {
			if err := git.WorktreeRemoveForce(t.repo, t.wtPath); err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
		} else {
			if err := git.WorktreeRemove(t.repo, t.wtPath); err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
		}
		fmt.Fprintf(os.Stdout, "removed %s\n", t.wtPath)

		var delErr error
		if flags.force {
			delErr = git.DeleteBranchForce(t.repo, name)
		} else {
			delErr = git.DeleteBranch(t.repo, name)
		}
		if delErr != nil {
			wtlog.Infof("could not delete branch %q in %s: %s", name, t.repo, delErr.Error())
		}

		if err := git.WorktreePrune(t.repo); err != nil {
			wtlog.Debugf("worktree prune in %s: %s", t.repo, err.Error())
		}
	}

	return nil
}

// findWorktreeByBranch returns the path of repo's linked worktree checked
// out on branch, or "" if no such worktree exists. The repo's main
// worktree is excluded so `rm <main-branch>` never tries to remove the
// repo root. We compare against the repo's symlink-resolved path because
// `git worktree list` always emits realpath'd entries while repo here may
// be a config-supplied path with symlinks.
func findWorktreeByBranch(repo, branch string) (string, error) {
	wts, err := git.ListWorktrees(repo)
	if err != nil {
		return "", err
	}
	repoReal := paths.RealpathSafe(repo)
	for _, w := range wts {
		if w.Branch == branch && w.Worktree != repoReal {
			return w.Worktree, nil
		}
	}
	return "", nil
}
