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
