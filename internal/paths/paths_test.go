package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	t.Run("expands a leading ~ to the home directory", func(t *testing.T) {
		assert.Equal(t, filepath.Join(home, "foo"), ExpandTilde("~/foo"))
	})

	t.Run("returns ~ alone as the home directory", func(t *testing.T) {
		assert.Equal(t, home, ExpandTilde("~"))
	})

	t.Run("returns non-tilde paths unchanged", func(t *testing.T) {
		assert.Equal(t, "/abs/path", ExpandTilde("/abs/path"))
		assert.Equal(t, "relative/path", ExpandTilde("relative/path"))
	})

	t.Run("does not expand a mid-string tilde", func(t *testing.T) {
		assert.Equal(t, "/foo/~/bar", ExpandTilde("/foo/~/bar"))
	})
}

func TestFlattenWorktreeName(t *testing.T) {
	t.Run("flattens forward slashes to dashes", func(t *testing.T) {
		assert.Equal(t, "feat-login", FlattenWorktreeName("feat/login"))
		assert.Equal(t, "feat-auth-magic-link", FlattenWorktreeName("feat/auth/magic-link"))
	})

	t.Run("returns names without slashes unchanged", func(t *testing.T) {
		assert.Equal(t, "main", FlattenWorktreeName("main"))
		assert.Equal(t, "feat-login", FlattenWorktreeName("feat-login"))
	})
}

func TestExpandWorktreePath(t *testing.T) {
	t.Run("interpolates {name} relative to repo for a relative pattern", func(t *testing.T) {
		got := ExpandWorktreePath("/repos/api", ".worktrees/{name}", "feat/login")
		assert.Equal(t, "/repos/api/.worktrees/feat-login", got)
	})

	t.Run("interpolates {name} as-is for an absolute pattern", func(t *testing.T) {
		got := ExpandWorktreePath("/repos/api", "/scratch/{name}", "feat/login")
		assert.Equal(t, "/scratch/feat-login", got)
	})

	t.Run("flattens slashes in the name before interpolation", func(t *testing.T) {
		got := ExpandWorktreePath("/repos/api", ".worktrees/{name}", "feat/auth/magic")
		assert.Equal(t, "/repos/api/.worktrees/feat-auth-magic", got)
	})
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	t.Run("returns true for an existing directory", func(t *testing.T) {
		assert.True(t, Exists(dir))
	})

	t.Run("returns true for an existing file", func(t *testing.T) {
		f := filepath.Join(dir, "file.txt")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		assert.True(t, Exists(f))
	})

	t.Run("returns true for a broken symlink (lstat semantics)", func(t *testing.T) {
		link := filepath.Join(dir, "broken")
		if err := os.Symlink("/nonexistent/target", link); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		assert.True(t, Exists(link))
	})

	t.Run("returns false for a missing path", func(t *testing.T) {
		assert.False(t, Exists(filepath.Join(dir, "does-not-exist")))
	})
}

func TestRealpathSafe(t *testing.T) {
	t.Run("returns the input unchanged when EvalSymlinks fails", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-such-thing")
		assert.Equal(t, missing, RealpathSafe(missing))
	})

	t.Run("resolves an existing path via EvalSymlinks", func(t *testing.T) {
		dir := t.TempDir()
		want, err := filepath.EvalSymlinks(dir)
		assert.NoError(t, err)
		assert.Equal(t, want, RealpathSafe(dir))
	})
}
