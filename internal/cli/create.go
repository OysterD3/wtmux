package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OysterD3/wtmux/internal/agents"
	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/glob"
	"github.com/OysterD3/wtmux/internal/group"
	"github.com/OysterD3/wtmux/internal/launch"
	wtlog "github.com/OysterD3/wtmux/internal/log"
	"github.com/OysterD3/wtmux/internal/preflight"
	"github.com/OysterD3/wtmux/internal/symlinks"
)

type createFlags struct {
	dryRun   bool
	base     string
	noLaunch bool
}

// newNewCmd returns the `wtmux new <name>` cobra command.
func newNewCmd() *cobra.Command {
	var flags createFlags
	cmd := &cobra.Command{
		Use:           "new <name> [-- agent-args...]",
		Short:         "Create coordinated worktrees and launch the agent",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          userArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyVerbose()
			extra := extractPassthrough(cmd, args)
			return runCreate(args[0], flags, extra)
		},
	}
	cmd.Flags().BoolVarP(&flags.dryRun, "dry-run", "n", false, "Plan without mutating")
	cmd.Flags().StringVarP(&flags.base, "base", "b", "", "Override base branch")
	cmd.Flags().BoolVar(&flags.noLaunch, "no-launch", false, "Skip launching the agent at the end")
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpToStdout(newHelp)
	})
	return cmd
}

// extractPassthrough returns argv tokens after a literal "--" terminator.
// We re-read os.Args because cobra strips "--" and everything after at
// parse time when ArgsLenAtDash() is consulted from a parent command with
// subcommands; reading os.Args directly is the reliable path.
func extractPassthrough(_ *cobra.Command, _ []string) []string {
	for i, a := range os.Args[1:] {
		if a == "--" {
			return append([]string{}, os.Args[i+2:]...)
		}
	}
	return nil
}

func runCreate(name string, flags createFlags, extra []string) error {
	cwd, err := mustGetwd()
	if err != nil {
		return err
	}
	if name == "" {
		return wtmuxerrors.New(wtmuxerrors.KindUser, "missing name argument: usage: wtmux new <name> [-- agent-args...]")
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
		return wtmuxerrors.New(
			wtmuxerrors.KindPrecondition,
			"cwd is not inside any git repository — run wtmux from within a repo, or pass --group",
		)
	}

	plan, err := preflight.BuildPlan(preflight.BuildInput{
		Config:       cfg,
		Resolution:   resolved,
		Name:         name,
		BaseOverride: flags.base,
	})
	if err != nil {
		return wtmuxerrors.New(wtmuxerrors.KindPrecondition, "%s", err.Error())
	}

	if flags.dryRun {
		printDryRun(plan)
		return nil
	}

	if err := plan.Validate(); err != nil {
		return wtmuxerrors.New(wtmuxerrors.KindUser, "%s", err.Error())
	}

	// Probe each repo once: BranchExists drives both the --base warning
	// below and the WorktreeAddExisting/AddNew dispatch in the placement
	// loop. Computing it twice (the original layout) was a wasted git call
	// and made it harder to keep the two decisions in sync.
	branchExists := make(map[string]bool, len(plan.Repos))
	for _, rp := range plan.Repos {
		e, err := git.BranchExists(rp.Repo, name)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		branchExists[rp.Repo] = e
	}

	// --base only applies to repos where the branch is being newly created.
	// In a mixed-state group some repos reuse the existing branch (HEAD)
	// while others branch off --base, so the warning has to name names —
	// the previous "ignored" blanket was misleading.
	if flags.base != "" {
		var reused, fresh []string
		for _, rp := range plan.Repos {
			label := filepath.Base(rp.Repo)
			if branchExists[rp.Repo] {
				reused = append(reused, label)
			} else {
				fresh = append(fresh, label)
			}
		}
		switch {
		case len(reused) == 0:
			// branch is being created in every repo — --base applies cleanly.
		case len(fresh) == 0:
			wtlog.Warnf(`--base %q ignored: branch %q already exists in every repo`, flags.base, name)
		default:
			wtlog.Warnf(
				`--base %q used in %s; ignored in %s where branch %q already exists`,
				flags.base, strings.Join(fresh, ", "), strings.Join(reused, ", "), name,
			)
		}
	}

	type placed struct {
		repo   string
		wtPath string
		items  []string // every item passed to Replicate; symlinks.Remove skips non-symlinks safely
	}

	expandedByRepo := make(map[string][]string, len(plan.Repos))
	for _, rp := range plan.Repos {
		exp, err := glob.ExpandSymlinkItems(rp.Repo, plan.SymlinkItems)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		expandedByRepo[rp.Repo] = exp
	}

	var placedOK []placed
	rollback := func() {
		for i := len(placedOK) - 1; i >= 0; i-- {
			p := placedOK[i]
			_ = symlinks.Remove(p.wtPath, p.items)
			_ = git.WorktreeRemoveForce(p.repo, p.wtPath)
		}
	}

	for _, rp := range plan.Repos {
		var err error
		if branchExists[rp.Repo] {
			err = git.WorktreeAddExisting(rp.Repo, rp.WtPath, name)
		} else {
			err = git.WorktreeAddNew(rp.Repo, rp.WtPath, name, plan.BaseBranch)
		}
		if err != nil {
			rollback()
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}

		// Record the freshly-created worktree before any further mutation
		// can fail. Without this, a Replicate failure below would leave
		// this repo's worktree orphaned on disk because rollback only
		// walks placedOK. symlinks.Replicate's contract on error is "the
		// returned slice is nil; pass the full Items slice to Remove",
		// and Remove safely skips non-symlinks — so capturing items up
		// front works for both partial-failure and clean-success cases.
		items := expandedByRepo[rp.Repo]
		placedOK = append(placedOK, placed{repo: rp.Repo, wtPath: rp.WtPath, items: items})

		if _, err := symlinks.Replicate(symlinks.ReplicateInputs{
			Repo:  rp.Repo,
			WT:    rp.WtPath,
			Items: items,
		}); err != nil {
			rollback()
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
	}

	if flags.noLaunch {
		return nil
	}

	strategy := agents.ResolveStrategy(agents.ResolveInput{
		LaunchCommand: plan.LaunchCommand,
		Agent:         plan.Agent,
		AddDirArgs:    plan.AddDirArgs,
		Warn:          func(m string) { wtlog.Warn(m) },
	})

	primaryWt := primaryWorktreePath(plan)
	if plan.Kind == group.KindSingle {
		if strategy.Kind == agents.StrategyNone {
			argv := append([]string{}, plan.LaunchCommand...)
			argv = append(argv, extra...)
			launch.Exec(argv, primaryWt)
			return nil
		}
		argv, err := launch.BuildArgv(plan.LaunchCommand, nil, strategy, extra)
		if err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		launch.Exec(argv, primaryWt)
		return nil
	}

	siblings := make([]string, 0, len(plan.Repos)-1)
	for _, rp := range plan.Repos {
		if rp.Repo == plan.Primary {
			continue
		}
		siblings = append(siblings, rp.WtPath)
	}

	if strategy.Kind == agents.StrategyNone {
		wtlog.Infof("%s has no multi-root support. Worktrees ready:", strategy.AgentID)
		fmt.Fprintf(os.Stderr, "  - %s\n", primaryWt)
		for _, s := range siblings {
			fmt.Fprintf(os.Stderr, "  - %s\n", s)
		}
		return nil
	}

	argv, err := launch.BuildArgv(plan.LaunchCommand, siblings, strategy, extra)
	if err != nil {
		return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
	}
	launch.Exec(argv, primaryWt)
	return nil
}

func primaryWorktreePath(plan *preflight.Plan) string {
	for _, rp := range plan.Repos {
		if rp.Repo == plan.Primary {
			return rp.WtPath
		}
	}
	// Single-repo plan: only one entry, and Primary == that entry.
	return plan.Repos[0].WtPath
}

func printDryRun(plan *preflight.Plan) {
	if plan.Kind == group.KindSingle {
		rp := plan.Repos[0]
		wtlog.Infof("dry-run: would create %s from %s (%s)", rp.WtPath, plan.BaseBranch, plan.Name)
		return
	}
	wtlog.Infof(`dry-run: would create worktrees for %q from %q:`, plan.Name, plan.BaseBranch)
	for _, rp := range plan.Repos {
		wtlog.Infof("  %s -> %s", rp.Repo, rp.WtPath)
	}
	wtlog.Infof("dry-run: symlink items: %s", strings.Join(plan.SymlinkItems, ", "))
}
