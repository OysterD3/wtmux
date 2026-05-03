// Package preflight builds and validates the per-invocation Plan that
// wtmux's create flow consumes. BuildPlan resolves per-repo target paths,
// the base branch, and the five group-vs-config override merges; Validate
// runs the live filesystem and git-state checks. The split lets dry-run
// inspect a Plan without touching the filesystem.
package preflight

import (
	"errors"
	"fmt"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/OysterD3/wtmux/internal/config"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/group"
	"github.com/OysterD3/wtmux/internal/paths"
)

// RepoPlan is the per-repo unit of work in a Plan: the original repo path
// (un-realpathed, so symlink replication and git calls see the path the
// user wrote) and the expanded worktree target.
type RepoPlan struct {
	Repo   string
	WtPath string
}

// Plan is the fully-resolved create-time intent. Built once, validated
// once, then handed to flows for execution. Plan never has Kind ==
// KindOutside; BuildPlan rejects that case before constructing one.
type Plan struct {
	// Identity
	Name       string
	BaseBranch string
	Kind       group.Kind

	// Repos
	Primary string     // for group: Resolution.Primary; for single: Repos[0].Repo
	Repos   []RepoPlan // exactly one entry in single mode

	// Resolved overrides (group field ?? config field)
	SymlinkItems  []string // unexpanded; flows runs glob right before symlink replication
	LaunchCommand []string
	Agent         agents.AgentID
	AddDirArgs    []string
}

// BuildInput is the BuildPlan parameter bundle.
type BuildInput struct {
	Config       *config.Config
	Resolution   *group.Resolution
	Name         string
	BaseOverride string // "" if --base not set
}

// Sentinel errors returned directly by BuildPlan (no aggregation).
var (
	// ErrOutside is returned when Resolution.Kind == KindOutside.
	ErrOutside = errors.New("preflight: cwd is not inside any git repository")
	// ErrDetachedHead is returned when no --base override is given and the
	// primary repo is on a detached HEAD.
	ErrDetachedHead = errors.New("preflight: detached HEAD on primary repo (pass --base or check out a branch)")
)

// BuildPlan resolves a fully-formed Plan from the given input. Pure-ish:
// the only I/O is git.GetCurrentBranch on Primary, called only when no
// BaseOverride is provided.
func BuildPlan(in BuildInput) (*Plan, error) {
	if in.Resolution.Kind == group.KindOutside {
		return nil, ErrOutside
	}

	var primary string
	switch in.Resolution.Kind {
	case group.KindGroup:
		primary = in.Resolution.Primary
	case group.KindSingle:
		primary = in.Resolution.Single
	default:
		return nil, fmt.Errorf("preflight: unknown resolution kind: %s", in.Resolution.Kind)
	}

	base := in.BaseOverride
	if base == "" {
		b, ok, err := git.GetCurrentBranch(primary)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDetachedHead
		}
		base = b
	}

	plan := &Plan{
		Name:       in.Name,
		BaseBranch: base,
		Kind:       in.Resolution.Kind,
		Primary:    primary,
	}

	if in.Resolution.Kind == group.KindGroup {
		g := in.Resolution.Group
		pattern := pickString(g.WorktreePathPattern, in.Config.WorktreePathPattern)
		for _, r := range g.Repos {
			plan.Repos = append(plan.Repos, RepoPlan{
				Repo:   r,
				WtPath: paths.ExpandWorktreePath(r, pattern, in.Name),
			})
		}
		plan.SymlinkItems = pickSlice(g.SymlinkDirectories, in.Config.SymlinkDirectories)
		plan.LaunchCommand = pickSlice(g.LaunchCommand, in.Config.LaunchCommand)
		plan.Agent = pickAgent(g.Agent, in.Config.Agent)
		plan.AddDirArgs = pickSlice(g.AddDirArgs, in.Config.AddDirArgs)
	} else {
		plan.Repos = []RepoPlan{{
			Repo:   primary,
			WtPath: paths.ExpandWorktreePath(primary, in.Config.WorktreePathPattern, in.Name),
		}}
		plan.SymlinkItems = in.Config.SymlinkDirectories
		plan.LaunchCommand = in.Config.LaunchCommand
		plan.Agent = in.Config.Agent
		plan.AddDirArgs = in.Config.AddDirArgs
	}

	return plan, nil
}

// pickString returns override if non-empty, else fallback.
func pickString(override, fallback string) string {
	if override != "" {
		return override
	}
	return fallback
}

// pickSlice returns override if non-empty, else fallback. Distinct from
// pickString because we need to discriminate nil/empty from set on slices
// in Go (no zero-string equivalent for a list).
func pickSlice(override, fallback []string) []string {
	if len(override) > 0 {
		return override
	}
	return fallback
}

// pickAgent returns override if set, else fallback.
func pickAgent(override, fallback agents.AgentID) agents.AgentID {
	if override != "" {
		return override
	}
	return fallback
}
