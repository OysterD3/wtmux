package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_v0_4_2_BackwardsCompat(t *testing.T) {
	// Read the on-disk fixture, drop it into a temp dir, load via Discover.
	src, err := os.ReadFile("testdata/v0_4_2_config.json")
	require.NoError(t, err)

	dir := t.TempDir()
	target := filepath.Join(dir, ".wtmux.json")
	require.NoError(t, os.WriteFile(target, src, 0o644))

	loaded, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  dir,
		Home: t.TempDir(),
		Env:  envFunc(nil),
	}})
	require.NoError(t, err)
	assert.Equal(t, target, loaded.Path)
	assert.Equal(t, SourceWalk, loaded.Source)
	require.Len(t, loaded.Config.Groups, 1)
	assert.Equal(t, "myteam", loaded.Config.Groups[0].Name)
	assert.Equal(t, []string{"/repo/a", "/repo/b"}, loaded.Config.Groups[0].Repos)
}

func TestLoad_AppliesDefaultsBeforeValidate(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".wtmux.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o644))

	loaded, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  dir,
		Home: t.TempDir(),
		Env:  envFunc(nil),
	}})
	require.NoError(t, err)
	assert.Equal(t, []string{"node_modules", ".env"}, loaded.Config.SymlinkDirectories)
	assert.Equal(t, ".worktrees/{name}", loaded.Config.WorktreePathPattern)
	assert.Equal(t, []string{"claude"}, loaded.Config.LaunchCommand)
}

func TestLoad_TildeExpandedOnGroupRepos(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	target := filepath.Join(dir, ".wtmux.json")
	require.NoError(t, os.WriteFile(target,
		[]byte(`{"groups":[{"name":"x","repos":["~/a","/abs/b"]}]}`), 0o644))

	t.Setenv("HOME", home) // ExpandTilde uses os.UserHomeDir()
	loaded, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  dir,
		Home: home,
		Env:  envFunc(nil),
	}})
	require.NoError(t, err)
	require.Len(t, loaded.Config.Groups, 1)
	expanded := filepath.Join(home, "a")
	assert.Equal(t, []string{expanded, "/abs/b"}, loaded.Config.Groups[0].Repos)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".wtmux.json")
	require.NoError(t, os.WriteFile(target, []byte("not-json"), 0o644))

	_, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  dir,
		Home: t.TempDir(),
		Env:  envFunc(nil),
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestLoad_ValidationErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".wtmux.json")
	// repos has only 1 entry → min=2 violation
	require.NoError(t, os.WriteFile(target,
		[]byte(`{"groups":[{"name":"x","repos":["/a"]}]}`), 0o644))

	_, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  dir,
		Home: t.TempDir(),
		Env:  envFunc(nil),
	}})
	require.Error(t, err)
	var verrs ValidationErrors
	require.True(t, errors.As(err, &verrs))
}

func TestLoad_NotFoundPropagates(t *testing.T) {
	_, err := Load(LoadInputs{DiscoveryInputs: DiscoveryInputs{
		Cwd:  t.TempDir(),
		Home: t.TempDir(),
		Env:  envFunc(nil),
	}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}
