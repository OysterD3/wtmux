package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkConfig(groups ...Group) *Config {
	c := &Config{Groups: groups}
	c.applyDefaults()
	return c
}

func TestAddGroup_Appends(t *testing.T) {
	c := mkConfig()
	out, err := AddGroup(c, Group{Name: "x", Repos: []string{"/a", "/b"}})
	require.NoError(t, err)
	require.Len(t, out.Groups, 1)
	assert.Equal(t, "x", out.Groups[0].Name)
}

func TestAddGroup_RejectsDuplicate(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	_, err := AddGroup(c, Group{Name: "x", Repos: []string{"/c", "/d"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate group name: x")
}

func TestAddGroup_DoesNotMutateInput(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	_, err := AddGroup(c, Group{Name: "y", Repos: []string{"/c", "/d"}})
	require.NoError(t, err)
	assert.Len(t, c.Groups, 1, "input config must be untouched")
}

func TestRemoveGroup_Removes(t *testing.T) {
	c := mkConfig(
		Group{Name: "x", Repos: []string{"/a", "/b"}},
		Group{Name: "y", Repos: []string{"/c", "/d"}},
	)
	out, err := RemoveGroup(c, "x")
	require.NoError(t, err)
	require.Len(t, out.Groups, 1)
	assert.Equal(t, "y", out.Groups[0].Name)
}

func TestRemoveGroup_NotFound(t *testing.T) {
	c := mkConfig()
	_, err := RemoveGroup(c, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found: missing")
}

func TestRenameGroup_Renames(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	out, err := RenameGroup(c, "x", "y")
	require.NoError(t, err)
	require.Len(t, out.Groups, 1)
	assert.Equal(t, "y", out.Groups[0].Name)
}

func TestRenameGroup_SameName_NoOp(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	out, err := RenameGroup(c, "x", "x")
	require.NoError(t, err)
	assert.Equal(t, "x", out.Groups[0].Name)
}

func TestRenameGroup_NotFound(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	_, err := RenameGroup(c, "missing", "y")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found: missing")
}

func TestRenameGroup_DuplicateNewName(t *testing.T) {
	c := mkConfig(
		Group{Name: "x", Repos: []string{"/a", "/b"}},
		Group{Name: "y", Repos: []string{"/c", "/d"}},
	)
	_, err := RenameGroup(c, "x", "y")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate group name: y")
}

func TestUpdateGroup_ReplacesContents(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	replacement := Group{Name: "x", Repos: []string{"/aa", "/bb", "/cc"}}
	out, err := UpdateGroup(c, "x", replacement)
	require.NoError(t, err)
	assert.Equal(t, []string{"/aa", "/bb", "/cc"}, out.Groups[0].Repos)
}

func TestUpdateGroup_NotFound(t *testing.T) {
	c := mkConfig()
	_, err := UpdateGroup(c, "missing", Group{Name: "missing", Repos: []string{"/a", "/b"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found: missing")
}

func TestAddReposToGroup_Appends(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	out, err := AddReposToGroup(c, "x", []string{"/c", "/d"})
	require.NoError(t, err)
	assert.Equal(t, []string{"/a", "/b", "/c", "/d"}, out.Groups[0].Repos)
}

func TestAddReposToGroup_DeDuplicates(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	out, err := AddReposToGroup(c, "x", []string{"/b", "/c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"/a", "/b", "/c"}, out.Groups[0].Repos)
}

func TestAddReposToGroup_NotFound(t *testing.T) {
	c := mkConfig()
	_, err := AddReposToGroup(c, "missing", []string{"/a", "/b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found: missing")
}

func TestRemoveReposFromGroup_Removes(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b", "/c"}})
	out, err := RemoveReposFromGroup(c, "x", []string{"/b"})
	require.NoError(t, err)
	assert.Equal(t, []string{"/a", "/c"}, out.Groups[0].Repos)
}

func TestRemoveReposFromGroup_TooFewLeft(t *testing.T) {
	c := mkConfig(Group{Name: "x", Repos: []string{"/a", "/b"}})
	_, err := RemoveReposFromGroup(c, "x", []string{"/a"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least 2 repos after removal")
}

func TestRemoveReposFromGroup_NotFound(t *testing.T) {
	c := mkConfig()
	_, err := RemoveReposFromGroup(c, "missing", []string{"/a"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group not found: missing")
}
