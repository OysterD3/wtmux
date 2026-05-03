package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_WritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")

	c := &Config{}
	c.applyDefaults()
	c.Groups = []Group{{Name: "x", Repos: []string{"/a", "/b"}}}

	require.NoError(t, Save(target, c))

	raw, err := os.ReadFile(target)
	require.NoError(t, err)

	var round Config
	require.NoError(t, json.Unmarshal(raw, &round))
	assert.Equal(t, *c, round)
}

func TestSave_TwoSpaceIndentTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"/a", "/b"}}},
	}
	c.applyDefaults()

	require.NoError(t, Save(target, c))

	raw, err := os.ReadFile(target)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "  \"groups\":", "expected 2-space indent")
	assert.True(t, len(s) > 0 && s[len(s)-1] == '\n', "expected trailing newline")
}

func TestSave_CreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "deep", "nested", "out.json")

	c := &Config{}
	c.applyDefaults()
	require.NoError(t, Save(target, c))
	_, err := os.Stat(target)
	require.NoError(t, err)
}

func TestSave_RefusesInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"/only-one"}}}, // min violation
	}
	c.applyDefaults()
	err := Save(target, c)
	require.Error(t, err)
	_, statErr := os.Stat(target)
	assert.True(t, os.IsNotExist(statErr), "no file should be written on validation failure")
}

func TestSave_NoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.json")
	c := &Config{}
	c.applyDefaults()
	require.NoError(t, Save(target, c))

	// .tmp file must be gone after rename.
	_, err := os.Stat(target + ".tmp")
	assert.True(t, os.IsNotExist(err), "%s.tmp should not exist", target)
}
