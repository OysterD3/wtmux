package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveStrategy_AddDirArgsWins(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"claude"},
		AddDirArgs:    []string{"--include", "{path}"},
	})
	assert.Equal(t, StrategyFlag, got.Kind)
	assert.Equal(t, []string{"--include", "{path}"}, got.FlagArgs)
	assert.Equal(t, SourceAddDirArgs, got.Source)
}

func TestResolveStrategy_AddDirArgsWarnsWhenAgentAlsoSet(t *testing.T) {
	var warned string
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"claude"},
		Agent:         AgentClaude,
		AddDirArgs:    []string{"--x", "{path}"},
		Warn:          func(m string) { warned = m },
	})
	assert.Equal(t, StrategyFlag, got.Kind)
	assert.Equal(t, SourceAddDirArgs, got.Source)
	assert.Contains(t, warned, "addDirArgs overrides agent")
}

func TestResolveStrategy_AgentExplicitFlag(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"my-wrapper"},
		Agent:         AgentCodex,
	})
	assert.Equal(t, StrategyFlag, got.Kind)
	assert.Equal(t, []string{"--add-dir", "{path}"}, got.FlagArgs)
	assert.Equal(t, SourceAgent, got.Source)
}

func TestResolveStrategy_AgentExplicitNone(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"my-wrapper"},
		Agent:         AgentOpenCode,
	})
	assert.Equal(t, StrategyNone, got.Kind)
	assert.Equal(t, AgentOpenCode, got.AgentID)
	assert.Equal(t, SourceAgent, got.Source)
}

func TestResolveStrategy_BasenameMatch(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"/usr/local/bin/cursor"},
	})
	assert.Equal(t, StrategyFlag, got.Kind)
	assert.Equal(t, []string{"--add", "{path}"}, got.FlagArgs)
	assert.Equal(t, SourceBasename, got.Source)
}

func TestResolveStrategy_BasenameAliasQoderCLI(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"qodercli"},
	})
	assert.Equal(t, StrategyNone, got.Kind)
	assert.Equal(t, AgentQoder, got.AgentID)
	assert.Equal(t, SourceBasename, got.Source)
}

func TestResolveStrategy_UnknownBasenameFallsBack(t *testing.T) {
	got := ResolveStrategy(ResolveInput{
		LaunchCommand: []string{"my-mystery-agent"},
	})
	assert.Equal(t, StrategyPositional, got.Kind)
	assert.Equal(t, SourceFallback, got.Source)
}

func TestExpandAddDirArgs(t *testing.T) {
	t.Run("expands one sibling", func(t *testing.T) {
		got := ExpandAddDirArgs([]string{"--add-dir", "{path}"}, []string{"/abs/sib"})
		assert.Equal(t, []string{"--add-dir", "/abs/sib"}, got)
	})

	t.Run("expands multiple siblings, repeating the template per sibling", func(t *testing.T) {
		got := ExpandAddDirArgs(
			[]string{"--include", "{path}"},
			[]string{"/abs/a", "/abs/b"},
		)
		assert.Equal(t, []string{"--include", "/abs/a", "--include", "/abs/b"}, got)
	})

	t.Run("replaces every {path} occurrence within a single token", func(t *testing.T) {
		got := ExpandAddDirArgs([]string{"{path}={path}"}, []string{"/x"})
		assert.Equal(t, []string{"/x=/x"}, got)
	})

	t.Run("returns empty when no siblings", func(t *testing.T) {
		got := ExpandAddDirArgs([]string{"--add-dir", "{path}"}, nil)
		assert.Empty(t, got)
	})
}
