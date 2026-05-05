package preflight

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/OysterD3/wtmux/internal/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makePlan constructs a Plan with sensible defaults for tests. Callers
// override fields as needed.
func makePlan(repo, name, base string) *Plan {
	wt := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-wt-"+name)
	return &Plan{
		Name:          name,
		BaseBranch:    base,
		Kind:          group.KindSingle,
		Primary:       repo,
		Repos:         []RepoPlan{{Repo: repo, WtPath: wt}},
		SymlinkItems:  []string{".env"},
		LaunchCommand: []string{"claude"},
		Agent:         agents.AgentClaude,
	}
}

func TestValidate_HappyPath_Single(t *testing.T) {
	repo := initRepo(t)
	p := makePlan(repo, "feat-x", "main")
	require.NoError(t, p.Validate())
}

func TestValidate_InvalidName(t *testing.T) {
	repo := initRepo(t)
	p := makePlan(repo, "bad name", "main") // space rejected by check-ref-format
	err := p.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidName))
}

func TestValidate_NotWorktreeRoot(t *testing.T) {
	repo := initRepo(t)
	// Create a linked worktree and treat IT as the "repo" — that's not a
	// worktree root.
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, repo, "worktree", "add", "-b", "feat", linked)

	p := makePlan(linked, "feat-y", "main")
	err := p.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotWorktreeRoot))
	// Other per-repo checks short-circuited for this repo.
}

func TestValidate_WtPathExists(t *testing.T) {
	repo := initRepo(t)
	p := makePlan(repo, "feat-x", "main")
	require.NoError(t, os.MkdirAll(p.Repos[0].WtPath, 0o755))

	err := p.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrWtPathExists))
}

func TestValidate_BranchCheckedOutElsewhere(t *testing.T) {
	repo := initRepo(t)
	// Create branch + a worktree on it, then validate a Plan that targets
	// that same name → must fail.
	mustGit(t, repo, "branch", "feat")
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, repo, "worktree", "add", linked, "feat")

	p := makePlan(repo, "feat", "main")
	err := p.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBranchCheckedOut))
}

func TestValidate_BaseMissing(t *testing.T) {
	repo := initRepo(t)
	// Branch "feat" doesn't exist (so the new-branch path is taken), and
	// neither does base "develop".
	p := makePlan(repo, "feat", "develop")
	err := p.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBaseMissing))
}

func TestValidate_BranchExistsButNoWorktree_Passes(t *testing.T) {
	repo := initRepo(t)
	mustGit(t, repo, "branch", "feat") // exists but no worktree

	p := makePlan(repo, "feat", "main")
	err := p.Validate()
	require.NoError(t, err)
}

func TestValidate_AggregatesMultiple(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)

	p := &Plan{
		Name:       "feat-x",
		BaseBranch: "develop", // missing in both
		Kind:       group.KindGroup,
		Primary:    a,
		Repos: []RepoPlan{
			{Repo: a, WtPath: filepath.Join(a, ".worktrees", "feat-x")},
			{Repo: b, WtPath: filepath.Join(b, ".worktrees", "feat-x")},
		},
		SymlinkItems:  []string{".env"},
		LaunchCommand: []string{"claude"},
	}
	// Pre-create wtPath on b → triggers ErrWtPathExists alongside
	// the universal ErrBaseMissing on both repos.
	require.NoError(t, os.MkdirAll(p.Repos[1].WtPath, 0o755))

	err := p.Validate()
	require.Error(t, err)

	// Both sentinels match via errors.Is (errors.Join unwraps).
	assert.True(t, errors.Is(err, ErrBaseMissing), "expected ErrBaseMissing in joined error: %v", err)
	assert.True(t, errors.Is(err, ErrWtPathExists), "expected ErrWtPathExists in joined error: %v", err)
}
