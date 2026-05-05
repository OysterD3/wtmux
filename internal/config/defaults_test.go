package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyDefaults_EmptyConfigGetsAllDefaults(t *testing.T) {
	c := &Config{}
	c.applyDefaults()
	assert.Equal(t, []string{"node_modules", ".env"}, c.SymlinkDirectories)
	assert.Equal(t, ".worktrees/{name}", c.WorktreePathPattern)
	assert.Equal(t, []string{"claude"}, c.LaunchCommand)
	assert.Nil(t, c.Groups)
}

func TestApplyDefaults_PreservesUserValues(t *testing.T) {
	c := &Config{
		SymlinkDirectories:  []string{"only-this"},
		WorktreePathPattern: "../wt/{name}",
		LaunchCommand:       []string{"custom-agent"},
	}
	c.applyDefaults()
	assert.Equal(t, []string{"only-this"}, c.SymlinkDirectories)
	assert.Equal(t, "../wt/{name}", c.WorktreePathPattern)
	assert.Equal(t, []string{"custom-agent"}, c.LaunchCommand)
}

func TestApplyDefaults_EmptySliceGetsDefault(t *testing.T) {
	// Treat zero-length slice the same as nil — TS .default() does this.
	c := &Config{
		SymlinkDirectories: []string{},
		LaunchCommand:      []string{},
	}
	c.applyDefaults()
	assert.Equal(t, []string{"node_modules", ".env"}, c.SymlinkDirectories)
	assert.Equal(t, []string{"claude"}, c.LaunchCommand)
}

func TestApplyDefaults_DoesNotTouchGroups(t *testing.T) {
	c := &Config{Groups: []Group{{Name: "x", Repos: []string{"/a", "/b"}}}}
	c.applyDefaults()
	// Groups left as-is; group field defaults are NOT applied here.
	assert.Equal(t, []Group{{Name: "x", Repos: []string{"/a", "/b"}}}, c.Groups)
}
