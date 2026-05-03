package preflight

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/OysterD3/wtmux/internal/config"
	"github.com/OysterD3/wtmux/internal/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initRepo creates a fresh git repo at t.TempDir() with one empty commit.
// Mirrors the helper in internal/group; per spec §7 we keep this local
// until a third caller appears.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-b", "main")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "commit", "--allow-empty", "-m", "init")
	return dir
}

func mustGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s failed: %v\n%s", args, cwd, err, out)
	}
}

func defaultedConfig() *config.Config {
	c := &config.Config{}
	// applyDefaults is unexported on Config; we let Validate-via-Save handle
	// real-world flows. For tests we set the three defaults explicitly.
	c.SymlinkDirectories = []string{"node_modules", ".env"}
	c.WorktreePathPattern = ".worktrees/{name}"
	c.LaunchCommand = []string{"claude"}
	return c
}

func TestBuildPlan_Single_DefaultBase(t *testing.T) {
	repo := initRepo(t)
	cfg := defaultedConfig()

	plan, err := BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindSingle, Single: repo},
		Name:       "feat-x",
	})
	require.NoError(t, err)
	assert.Equal(t, "feat-x", plan.Name)
	assert.Equal(t, "main", plan.BaseBranch)
	assert.Equal(t, group.KindSingle, plan.Kind)
	assert.Equal(t, repo, plan.Primary)
	require.Len(t, plan.Repos, 1)
	assert.Equal(t, repo, plan.Repos[0].Repo)
	assert.Equal(t, filepath.Join(repo, ".worktrees", "feat-x"), plan.Repos[0].WtPath)
	assert.Equal(t, []string{"node_modules", ".env"}, plan.SymlinkItems)
	assert.Equal(t, []string{"claude"}, plan.LaunchCommand)
	assert.Equal(t, agents.AgentID(""), plan.Agent)
	assert.Nil(t, plan.AddDirArgs)
}

func TestBuildPlan_Single_BaseOverride(t *testing.T) {
	repo := initRepo(t)
	cfg := defaultedConfig()

	plan, err := BuildPlan(BuildInput{
		Config:       cfg,
		Resolution:   &group.Resolution{Kind: group.KindSingle, Single: repo},
		Name:         "feat-x",
		BaseOverride: "develop",
	})
	require.NoError(t, err)
	assert.Equal(t, "develop", plan.BaseBranch, "BaseOverride wins even if branch absent — Validate's job")
}

func TestBuildPlan_Single_DetachedHead(t *testing.T) {
	repo := initRepo(t)
	// Get the commit SHA, then detach.
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	sha := string(out[:len(out)-1])
	mustGit(t, repo, "checkout", "--detach", sha)

	cfg := defaultedConfig()
	_, err = BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindSingle, Single: repo},
		Name:       "feat-x",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDetachedHead))
}

func TestBuildPlan_Outside(t *testing.T) {
	cfg := defaultedConfig()
	_, err := BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindOutside},
		Name:       "feat-x",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOutside))
}

func TestBuildPlan_Group_OverridesGroupFields(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)
	cfg := defaultedConfig()
	g := &config.Group{
		Name:                "team",
		Repos:               []string{a, b},
		SymlinkDirectories:  []string{"only-this"},
		WorktreePathPattern: "../wt/{name}",
		LaunchCommand:       []string{"my-agent"},
		Agent:               agents.AgentCodex,
		AddDirArgs:          []string{"--include", "{path}"},
	}

	plan, err := BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindGroup, Group: g, Primary: a},
		Name:       "feat-x",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"only-this"}, plan.SymlinkItems)
	assert.Equal(t, []string{"my-agent"}, plan.LaunchCommand)
	assert.Equal(t, agents.AgentCodex, plan.Agent)
	assert.Equal(t, []string{"--include", "{path}"}, plan.AddDirArgs)
	require.Len(t, plan.Repos, 2)
	// pattern is "../wt/{name}" — relative, so joined onto each repo.
	assert.Equal(t, filepath.Join(a, "..", "wt", "feat-x"), plan.Repos[0].WtPath)
	assert.Equal(t, filepath.Join(b, "..", "wt", "feat-x"), plan.Repos[1].WtPath)
}

func TestBuildPlan_Group_FallsBackToConfig(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)
	cfg := defaultedConfig()
	cfg.Agent = agents.AgentClaude
	cfg.AddDirArgs = []string{"--add-dir", "{path}"}

	g := &config.Group{Name: "team", Repos: []string{a, b}}

	plan, err := BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindGroup, Group: g, Primary: a},
		Name:       "feat-x",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"node_modules", ".env"}, plan.SymlinkItems)
	assert.Equal(t, []string{"claude"}, plan.LaunchCommand)
	assert.Equal(t, agents.AgentClaude, plan.Agent)
	assert.Equal(t, []string{"--add-dir", "{path}"}, plan.AddDirArgs)
	require.Len(t, plan.Repos, 2)
	assert.Equal(t, filepath.Join(a, ".worktrees", "feat-x"), plan.Repos[0].WtPath)
}

func TestBuildPlan_Group_WtPathPerRepo(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)
	cfg := defaultedConfig()
	g := &config.Group{Name: "team", Repos: []string{a, b}}

	plan, err := BuildPlan(BuildInput{
		Config:     cfg,
		Resolution: &group.Resolution{Kind: group.KindGroup, Group: g, Primary: a},
		Name:       "feat/login",
	})
	require.NoError(t, err)
	require.Len(t, plan.Repos, 2)
	// "/" in name flattens to "-".
	assert.Equal(t, filepath.Join(a, ".worktrees", "feat-login"), plan.Repos[0].WtPath)
	assert.Equal(t, filepath.Join(b, ".worktrees", "feat-login"), plan.Repos[1].WtPath)
}
