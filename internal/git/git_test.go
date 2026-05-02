package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("git"); err != nil {
		panic("git binary required for internal/git tests: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestSmoke(t *testing.T) {
	// Sentinel: confirms TestMain ran (suite would panic before reaching
	// any test if git was missing).
	t.Log("TestMain gate passed")
}

func TestCheckRefFormat(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"plain branch name", "main", true},
		{"slash in branch name", "feat/login", true},
		{"empty string", "", false},
		{"leading dash", "-bad", false},
		{"contains space", "bad name", false},
		{"double dot", "feat..bad", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckRefFormat(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// initRepo creates a fresh git repository in t.TempDir() with a single
// initial commit. The returned path is the repo root.
//
// Note on macOS: t.TempDir() lives under a symlinked path
// (/var/folders/... → /private/var/folders/...). git rev-parse resolves
// symlinks, so callers comparing against the returned path should resolve
// both sides via filepath.EvalSymlinks.
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

// resolveSymlinks is a test helper that calls filepath.EvalSymlinks and
// fails the test on error. Use it to compare paths returned by git
// (which resolves symlinks) against TempDir paths (which may contain
// unresolved symlinks on macOS).
func resolveSymlinks(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	require.NoError(t, err)
	return resolved
}

func TestGetToplevel_InsideRepo(t *testing.T) {
	repo := initRepo(t)
	sub := filepath.Join(repo, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	got, ok, err := GetToplevel(sub)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, resolveSymlinks(t, repo), resolveSymlinks(t, got))
}

func TestGetToplevel_OutsideRepo(t *testing.T) {
	dir := t.TempDir() // not initialized as a repo

	got, ok, err := GetToplevel(dir)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestIsWorktreeRoot_Main(t *testing.T) {
	repo := initRepo(t)

	got, err := IsWorktreeRoot(repo)
	require.NoError(t, err)
	assert.True(t, got)
}

func TestIsWorktreeRoot_Linked(t *testing.T) {
	repo := initRepo(t)
	wt := filepath.Join(t.TempDir(), "linked-wt")
	mustGit(t, repo, "worktree", "add", "-b", "feat", wt)

	got, err := IsWorktreeRoot(wt)
	require.NoError(t, err)
	assert.False(t, got)
}

func TestGetCurrentBranch_OnBranch(t *testing.T) {
	repo := initRepo(t)

	got, ok, err := GetCurrentBranch(repo)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "main", got)
}

func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	repo := initRepo(t)
	// Resolve HEAD to a SHA, then check it out detached.
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repo
	out, err := cmd.Output()
	require.NoError(t, err)
	sha := string(out[:len(out)-1]) // strip trailing newline
	mustGit(t, repo, "checkout", "--detach", sha)

	got, ok, err := GetCurrentBranch(repo)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, got)
}

func TestBranchExists_True(t *testing.T) {
	repo := initRepo(t)

	got, err := BranchExists(repo, "main")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestBranchExists_False(t *testing.T) {
	repo := initRepo(t)

	got, err := BranchExists(repo, "nonexistent")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestHasRef_True(t *testing.T) {
	repo := initRepo(t)

	got, err := HasRef(repo, "HEAD")
	require.NoError(t, err)
	assert.True(t, got)
}

func TestHasRef_False(t *testing.T) {
	repo := initRepo(t)

	got, err := HasRef(repo, "refs/heads/never-existed")
	require.NoError(t, err)
	assert.False(t, got)
}

func TestStatusPorcelain_Clean(t *testing.T) {
	repo := initRepo(t)

	got, err := StatusPorcelain(repo)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStatusPorcelain_Dirty(t *testing.T) {
	repo := initRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "new.txt"), []byte("hi"), 0o644))

	got, err := StatusPorcelain(repo)
	require.NoError(t, err)
	assert.Contains(t, got, "?? new.txt")
}

func TestStashList_Empty(t *testing.T) {
	repo := initRepo(t)

	got, err := StashList(repo)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestStashList_NonEmpty(t *testing.T) {
	repo := initRepo(t)
	// Create a tracked file, modify it, stash.
	tracked := filepath.Join(repo, "f.txt")
	require.NoError(t, os.WriteFile(tracked, []byte("v1"), 0o644))
	mustGit(t, repo, "add", "f.txt")
	mustGit(t, repo, "commit", "-m", "add f")
	require.NoError(t, os.WriteFile(tracked, []byte("v2"), 0o644))
	mustGit(t, repo, "stash")

	got, err := StashList(repo)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Contains(t, got[0], "stash@{0}")
}

func TestUnpushedCommits_NoUpstream(t *testing.T) {
	repo := initRepo(t)

	got, err := UnpushedCommits(repo)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUnpushedCommits_WithUpstream(t *testing.T) {
	// Create a bare upstream, add it as the remote, push initial commit,
	// then make a new commit locally that hasn't been pushed.
	upstream := filepath.Join(t.TempDir(), "upstream.git")
	cmd := exec.Command("git", "init", "--bare", upstream)
	require.NoError(t, cmd.Run())

	repo := initRepo(t)
	mustGit(t, repo, "remote", "add", "origin", upstream)
	mustGit(t, repo, "push", "-u", "origin", "main")

	// Local commit that isn't pushed yet.
	require.NoError(t, os.WriteFile(filepath.Join(repo, "x.txt"), []byte("x"), 0o644))
	mustGit(t, repo, "add", "x.txt")
	mustGit(t, repo, "commit", "-m", "local-only")

	got, err := UnpushedCommits(repo)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Contains(t, got[0], "local-only")
}
