package config

import (
	"encoding/json"
	"testing"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_JSONRoundTrip(t *testing.T) {
	in := Config{
		Schema:              "https://example.com/wtmux.json",
		SymlinkDirectories:  []string{"node_modules", ".env"},
		WorktreePathPattern: ".worktrees/{name}",
		LaunchCommand:       []string{"claude"},
		Agent:               agents.AgentClaude,
		AddDirArgs:          []string{"--add-dir", "{path}"},
		Groups: []Group{
			{
				Name:  "myteam",
				Repos: []string{"/repo/a", "/repo/b"},
			},
		},
	}

	raw, err := json.Marshal(in)
	require.NoError(t, err)

	var out Config
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.Equal(t, in, out)
}

func TestConfig_OmitemptyDropsZeroFields(t *testing.T) {
	in := Config{} // every field zero
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(raw))
}

func TestGroup_JSONRoundTrip(t *testing.T) {
	in := Group{
		Name:                "x",
		Repos:               []string{"/a", "/b"},
		SymlinkDirectories:  []string{".env"},
		WorktreePathPattern: "../wt/{name}",
		LaunchCommand:       []string{"my-agent"},
		Agent:               agents.AgentCodex,
		AddDirArgs:          []string{"--include", "{path}"},
	}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	var out Group
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.Equal(t, in, out)
}
