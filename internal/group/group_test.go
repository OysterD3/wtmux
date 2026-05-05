package group

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/OysterD3/wtmux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initRepo creates a fresh git repo at t.TempDir() with one empty commit.
// Mirrors the Phase 3 helper to keep group/preflight tests self-contained.
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

// resolveSym is filepath.EvalSymlinks but t.Fatalf on error — a
// convenience for asserting against paths returned by git/os helpers
// that have already resolved symlinks.
func resolveSym(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	require.NoError(t, err)
	return r
}

func TestDeterminePrimary_CwdInGroup(t *testing.T) {
	repo := initRepo(t)
	other := initRepo(t)
	g := &config.Group{Name: "x", Repos: []string{repo, other}}

	got := determinePrimary(repo, g)
	assert.Equal(t, repo, got)
}

func TestDeterminePrimary_CwdNotInGroup_FallsBackToFirst(t *testing.T) {
	repo := initRepo(t) // not in the configured group
	a := initRepo(t)
	b := initRepo(t)
	g := &config.Group{Name: "x", Repos: []string{a, b}}

	got := determinePrimary(repo, g)
	assert.Equal(t, a, got)
}

func TestDeterminePrimary_CwdOutsideRepo_FallsBackToFirst(t *testing.T) {
	cwd := t.TempDir() // not git-init'd
	a := initRepo(t)
	b := initRepo(t)
	g := &config.Group{Name: "x", Repos: []string{a, b}}

	got := determinePrimary(cwd, g)
	assert.Equal(t, a, got)
}

func TestResolve_GroupFlag_Match(t *testing.T) {
	repo := initRepo(t)
	other := initRepo(t)
	cfg := &config.Config{
		Groups: []config.Group{{Name: "team", Repos: []string{repo, other}}},
	}

	got, err := Resolve(ResolveInput{Cwd: repo, Config: cfg, GroupFlag: "team"})
	require.NoError(t, err)
	assert.Equal(t, KindGroup, got.Kind)
	require.NotNil(t, got.Group)
	assert.Equal(t, "team", got.Group.Name)
	assert.Equal(t, repo, got.Primary)
}

func TestResolve_GroupFlag_Miss(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{{Name: "team", Repos: []string{"/a", "/b"}}},
	}
	_, err := Resolve(ResolveInput{Cwd: t.TempDir(), Config: cfg, GroupFlag: "nope"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGroupFlagNotFound))
}

func TestResolve_OutsideRepo(t *testing.T) {
	cfg := &config.Config{}
	got, err := Resolve(ResolveInput{Cwd: t.TempDir(), Config: cfg})
	require.NoError(t, err)
	assert.Equal(t, KindOutside, got.Kind)
}

func TestResolve_SingleRepo(t *testing.T) {
	repo := initRepo(t)
	cfg := &config.Config{} // no groups

	got, err := Resolve(ResolveInput{Cwd: repo, Config: cfg})
	require.NoError(t, err)
	assert.Equal(t, KindSingle, got.Kind)
	assert.Equal(t, resolveSym(t, repo), got.Single)
}

func TestResolve_GroupMatch_FromCwd(t *testing.T) {
	repo := initRepo(t)
	other := initRepo(t)
	cfg := &config.Config{
		Groups: []config.Group{{Name: "team", Repos: []string{repo, other}}},
	}
	got, err := Resolve(ResolveInput{Cwd: repo, Config: cfg})
	require.NoError(t, err)
	assert.Equal(t, KindGroup, got.Kind)
	assert.Equal(t, "team", got.Group.Name)
	assert.Equal(t, repo, got.Primary)
}

func TestResolve_GroupMatch_FromUnrelatedCwd_BecomesSingle(t *testing.T) {
	unrelated := initRepo(t)
	a := initRepo(t)
	b := initRepo(t)
	cfg := &config.Config{
		Groups: []config.Group{{Name: "team", Repos: []string{a, b}}},
	}
	got, err := Resolve(ResolveInput{Cwd: unrelated, Config: cfg})
	require.NoError(t, err)
	assert.Equal(t, KindSingle, got.Kind, "unrelated repo should fall through to single mode")
	assert.Equal(t, resolveSym(t, unrelated), got.Single)
}

func TestResolve_AmbiguousGroups(t *testing.T) {
	shared := initRepo(t)
	other1 := initRepo(t)
	other2 := initRepo(t)
	cfg := &config.Config{
		Groups: []config.Group{
			{Name: "alpha", Repos: []string{shared, other1}},
			{Name: "beta", Repos: []string{shared, other2}},
		},
	}
	_, err := Resolve(ResolveInput{Cwd: shared, Config: cfg})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAmbiguousGroups))
	assert.Contains(t, err.Error(), "alpha")
	assert.Contains(t, err.Error(), "beta")
}
