package config

import (
	"errors"
	"testing"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_HappyPath(t *testing.T) {
	c := &Config{}
	c.applyDefaults()
	require.NoError(t, c.Validate())
}

func TestValidate_GoodGroup(t *testing.T) {
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"/a", "/b"}}},
	}
	c.applyDefaults()
	require.NoError(t, c.Validate())
}

func TestValidate_PatternMissingNamePlaceholder(t *testing.T) {
	c := &Config{WorktreePathPattern: ".worktrees/no-placeholder"}
	c.applyDefaults() // leaves WorktreePathPattern alone (already non-empty)
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{name}")
}

func TestValidate_AddDirArgsMissingPathPlaceholder(t *testing.T) {
	c := &Config{AddDirArgs: []string{"--add-dir"}} // no {path} anywhere
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "{path}")
}

func TestValidate_RelativeRepoPath(t *testing.T) {
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"./relative", "/abs"}}},
	}
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absolute path")
}

func TestValidate_GroupNeedsTwoRepos(t *testing.T) {
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"/only-one"}}},
	}
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least")
}

func TestValidate_UnknownAgentRejected(t *testing.T) {
	c := &Config{Agent: agents.AgentID("bogus")}
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestValidate_DuplicateGroupName(t *testing.T) {
	c := &Config{
		Groups: []Group{
			{Name: "team", Repos: []string{"/a", "/b"}},
			{Name: "team", Repos: []string{"/c", "/d"}},
		},
	}
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate value")
	assert.Contains(t, err.Error(), "groups[1]")
}

func TestValidate_DistinctGroupNamesPass(t *testing.T) {
	c := &Config{
		Groups: []Group{
			{Name: "alpha", Repos: []string{"/a", "/b"}},
			{Name: "beta", Repos: []string{"/c", "/d"}},
		},
	}
	c.applyDefaults()
	require.NoError(t, c.Validate())
}

func TestValidate_ReturnsValidationErrorsType(t *testing.T) {
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"./relative"}}},
	}
	c.applyDefaults()
	err := c.Validate()
	require.Error(t, err)

	var verrs ValidationErrors
	require.True(t, errors.As(err, &verrs), "expected ValidationErrors, got %T", err)
	require.NotEmpty(t, verrs)
}

func TestValidate_PathFormatsAsJSONLowerCase(t *testing.T) {
	c := &Config{
		Groups: []Group{{Name: "x", Repos: []string{"./relative", "/abs"}}},
	}
	c.applyDefaults()
	err := c.Validate()
	var verrs ValidationErrors
	require.True(t, errors.As(err, &verrs))

	// At least one entry must reference the JSON-style path.
	found := false
	for _, e := range verrs {
		if e.Path == "groups[0].repos[0]" {
			found = true
			assert.Equal(t, "abspath", e.Tag)
			break
		}
	}
	assert.True(t, found, "expected groups[0].repos[0] in errors: %v", verrs)
}

func TestValidate_ReportsMultipleIssues(t *testing.T) {
	c := &Config{
		WorktreePathPattern: "no-placeholder",
		Groups: []Group{
			{Name: "x", Repos: []string{"/only-one"}}, // min violation
		},
	}
	c.applyDefaults() // pattern stays "no-placeholder" (already non-empty)
	err := c.Validate()
	var verrs ValidationErrors
	require.True(t, errors.As(err, &verrs))
	assert.GreaterOrEqual(t, len(verrs), 2, "expected at least 2 errors, got %v", verrs)
}
