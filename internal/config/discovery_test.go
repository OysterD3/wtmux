package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDiscover_ExplicitFlagWins(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "custom.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o644))

	got, err := Discover(DiscoveryInputs{
		Explicit: target,
		Cwd:      dir,
		Home:     dir,
		Env:      envFunc(nil),
	})
	require.NoError(t, err)
	assert.Equal(t, target, got.Path)
	assert.Equal(t, SourceFlag, got.Source)
}

func TestDiscover_ExplicitFlagMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "no-such-file.json")

	_, err := Discover(DiscoveryInputs{
		Explicit: missing,
		Cwd:      dir,
		Home:     dir,
		Env:      envFunc(nil),
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotFound), "explicit-missing must NOT match ErrNotFound")
	assert.Contains(t, err.Error(), missing)
}

func TestDiscover_EnvVarWins(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "from-env.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o644))

	got, err := Discover(DiscoveryInputs{
		Cwd:  dir,
		Home: dir,
		Env:  envFunc(map[string]string{"WTMUX_CONFIG": target}),
	})
	require.NoError(t, err)
	assert.Equal(t, target, got.Path)
	assert.Equal(t, SourceEnv, got.Source)
}

func TestDiscover_EnvVarMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope.json")

	_, err := Discover(DiscoveryInputs{
		Cwd:  dir,
		Home: dir,
		Env:  envFunc(map[string]string{"WTMUX_CONFIG": missing}),
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotFound))
	assert.Contains(t, err.Error(), "WTMUX_CONFIG")
}

func TestDiscover_WalkUpward(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, ".wtmux.json")
	require.NoError(t, os.WriteFile(cfg, []byte("{}"), 0o644))
	deep := filepath.Join(root, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0o755))

	got, err := Discover(DiscoveryInputs{
		Cwd:  deep,
		Home: t.TempDir(), // separate, avoid XDG hit
		Env:  envFunc(nil),
	})
	require.NoError(t, err)
	assert.Equal(t, cfg, got.Path)
	assert.Equal(t, SourceWalk, got.Source)
}

func TestDiscover_XDGFallback(t *testing.T) {
	home := t.TempDir()
	xdgRoot := filepath.Join(home, ".config", "wtmux")
	require.NoError(t, os.MkdirAll(xdgRoot, 0o755))
	xdgCfg := filepath.Join(xdgRoot, "config.json")
	require.NoError(t, os.WriteFile(xdgCfg, []byte("{}"), 0o644))

	cwd := t.TempDir() // distinct from home, no .wtmux.json upward
	got, err := Discover(DiscoveryInputs{
		Cwd:  cwd,
		Home: home,
		Env:  envFunc(nil),
	})
	require.NoError(t, err)
	assert.Equal(t, xdgCfg, got.Path)
	assert.Equal(t, SourceXDG, got.Source)
}

func TestDiscover_XDGConfigHomeOverride(t *testing.T) {
	home := t.TempDir()
	xdgHome := filepath.Join(t.TempDir(), "xdgcfg")
	require.NoError(t, os.MkdirAll(filepath.Join(xdgHome, "wtmux"), 0o755))
	target := filepath.Join(xdgHome, "wtmux", "config.json")
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o644))

	cwd := t.TempDir()
	got, err := Discover(DiscoveryInputs{
		Cwd:  cwd,
		Home: home,
		Env:  envFunc(map[string]string{"XDG_CONFIG_HOME": xdgHome}),
	})
	require.NoError(t, err)
	assert.Equal(t, target, got.Path)
	assert.Equal(t, SourceXDG, got.Source)
}

func TestDiscover_NoneFoundReturnsErrNotFound(t *testing.T) {
	_, err := Discover(DiscoveryInputs{
		Cwd:  t.TempDir(),
		Home: t.TempDir(),
		Env:  envFunc(nil),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}
