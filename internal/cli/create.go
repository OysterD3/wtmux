package cli

import (
	"fmt"
	"os"
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
		Args:          cobra.MinimumNArgs(1),
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
		return wtmuxerrors.New(wtmuxerrors.KindUser, "%s", err.Error())
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

	// --base ignored when branch already exists in any repo
	if flags.base != "" {
		for _, rp := range plan.Repos {
			exists, err := git.BranchExists(rp.Repo, name)
			if err != nil {
				return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
			}
			if exists {
				wtlog.Warnf(`--base %q ignored: branch %q already exists`, flags.base, name)
				break
			}
		}
	}

	type placed struct {
		repo   string
		wtPath string
		linked []string
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
			_ = symlinks.Remove(p.wtPath, p.linked)
			_ = git.WorktreeRemoveForce(p.repo, p.wtPath)
		}
	}

	for _, rp := range plan.Repos {
		exists, err := git.BranchExists(rp.Repo, name)
		if err != nil {
			rollback()
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		if exists {
			err = git.WorktreeAddExisting(rp.Repo, rp.WtPath, name)
		} else {
			err = git.WorktreeAddNew(rp.Repo, rp.WtPath, name, plan.BaseBranch)
		}
		if err != nil {
			rollback()
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}

		items := expandedByRepo[rp.Repo]
		results, err := symlinks.Replicate(symlinks.ReplicateInputs{
			Repo:  rp.Repo,
			WT:    rp.WtPath,
			Items: items,
		})
		if err != nil {
			rollback()
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
		}
		var linked []string
		for _, r := range results {
			if r.Action == symlinks.ActionLinked {
				linked = append(linked, r.Item)
			}
		}
		placedOK = append(placedOK, placed{repo: rp.Repo, wtPath: rp.WtPath, linked: linked})
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
