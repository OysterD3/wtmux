package symlinks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplicate_LinkAFile(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("X=1"), 0o644))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{".env"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{{Item: ".env", Action: ActionLinked}}, results)

	target := filepath.Join(wt, ".env")
	info, err := os.Lstat(target)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "target should be a symlink")
	dst, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(repo, ".env"), dst)
}

func TestReplicate_LinkADirectory(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o755))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{"node_modules"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{{Item: "node_modules", Action: ActionLinked}}, results)

	info, err := os.Lstat(filepath.Join(wt, "node_modules"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
}

func TestReplicate_SkipNoSource(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o755))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{"missing"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{{Item: "missing", Action: ActionSkippedNoSource}}, results)

	_, err = os.Lstat(filepath.Join(wt, "missing"))
	assert.True(t, os.IsNotExist(err), "no target should have been created")
}

func TestReplicate_SkipTargetExists(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("source"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(wt, ".env"), []byte("preexisting"), 0o644))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{".env"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{{Item: ".env", Action: ActionSkippedTargetExists}}, results)

	contents, err := os.ReadFile(filepath.Join(wt, ".env"))
	require.NoError(t, err)
	assert.Equal(t, "preexisting", string(contents), "preexisting file must be untouched")
}

func TestReplicate_CreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "config", "sub"), 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "config", "sub", "b.json"), []byte("{}"), 0o644))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{"config/sub/b.json"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{{Item: "config/sub/b.json", Action: ActionLinked}}, results)

	info, err := os.Lstat(filepath.Join(wt, "config", "sub", "b.json"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
}

func TestReplicate_MultipleItems(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("x"), 0o644))
	require.NoError(t, os.MkdirAll(wt, 0o755))

	results, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{"node_modules", ".env", "missing"}})
	require.NoError(t, err)
	assert.Equal(t, []SymlinkResult{
		{Item: "node_modules", Action: ActionLinked},
		{Item: ".env", Action: ActionLinked},
		{Item: "missing", Action: ActionSkippedNoSource},
	}, results)
}

func TestReplicate_PropagatesPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission tests don't apply to root")
	}
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	require.NoError(t, os.MkdirAll(wt, 0o000))
	t.Cleanup(func() { _ = os.Chmod(wt, 0o755) })
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("x"), 0o644))

	_, err := Replicate(ReplicateInputs{Repo: repo, WT: wt, Items: []string{".env"}})
	assert.Error(t, err, "permission-denied lstat on target should propagate")
}

func TestRemove_RemovesSymlink(t *testing.T) {
	wt := t.TempDir()
	target := filepath.Join(wt, ".env")
	require.NoError(t, os.Symlink("/nonexistent/source", target))

	err := Remove(wt, []string{".env"})
	require.NoError(t, err)

	_, err = os.Lstat(target)
	assert.True(t, os.IsNotExist(err), "symlink should be gone")
}

func TestRemove_NoOpWhenMissing(t *testing.T) {
	wt := t.TempDir()
	err := Remove(wt, []string{".env"})
	assert.NoError(t, err)
}

func TestRemove_DoesNotRemoveRegularFile(t *testing.T) {
	wt := t.TempDir()
	target := filepath.Join(wt, ".env")
	require.NoError(t, os.WriteFile(target, []byte("real"), 0o644))

	err := Remove(wt, []string{".env"})
	require.NoError(t, err)

	info, err := os.Lstat(target)
	require.NoError(t, err)
	assert.Zero(t, info.Mode()&os.ModeSymlink, "regular file must remain")
	contents, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "real", string(contents))
}

func TestRemove_DoesNotRemoveDirectory(t *testing.T) {
	wt := t.TempDir()
	target := filepath.Join(wt, "node_modules")
	require.NoError(t, os.MkdirAll(target, 0o755))

	err := Remove(wt, []string{"node_modules"})
	require.NoError(t, err)

	info, err := os.Lstat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "directory must remain")
}

func TestRemove_MultipleItems(t *testing.T) {
	wt := t.TempDir()
	require.NoError(t, os.Symlink("/nowhere/a", filepath.Join(wt, "a")))
	require.NoError(t, os.WriteFile(filepath.Join(wt, "b"), []byte("real"), 0o644))

	err := Remove(wt, []string{"a", "b", "missing"})
	require.NoError(t, err)

	_, err = os.Lstat(filepath.Join(wt, "a"))
	assert.True(t, os.IsNotExist(err), "symlink a should be gone")
	_, err = os.Lstat(filepath.Join(wt, "b"))
	assert.NoError(t, err, "regular file b must remain")
}

func TestRemove_PropagatesLstatError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission tests don't apply to root")
	}
	dir := t.TempDir()
	wt := filepath.Join(dir, "wt")
	require.NoError(t, os.MkdirAll(wt, 0o755))
	require.NoError(t, os.Symlink("/nowhere", filepath.Join(wt, ".env")))
	require.NoError(t, os.Chmod(wt, 0o000))
	t.Cleanup(func() { _ = os.Chmod(wt, 0o755) })

	err := Remove(wt, []string{".env"})
	assert.Error(t, err, "permission-denied lstat should propagate")
}
