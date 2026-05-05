// Package config defines the wtmux configuration schema and the helpers used
// to load, validate, save, and mutate it. The schema mirrors v0.4.2 of the
// TypeScript implementation byte-for-byte so existing user configs keep
// working without edits.
package config

import "github.com/OysterD3/wtmux/internal/agents"

// Config is the top-level wtmux configuration. Fields use json:"omitempty"
// so saved files don't carry zero-valued keys.
type Config struct {
	Schema              string         `json:"$schema,omitempty"`
	SymlinkDirectories  []string       `json:"symlinkDirectories,omitempty"`
	WorktreePathPattern string         `json:"worktreePathPattern,omitempty" validate:"omitempty,containsName"`
	LaunchCommand       []string       `json:"launchCommand,omitempty"       validate:"omitempty,min=1"`
	Agent               agents.AgentID `json:"agent,omitempty"               validate:"omitempty,oneof=claude codex cursor code opencode qoder"`
	AddDirArgs          []string       `json:"addDirArgs,omitempty"          validate:"omitempty,min=1,containsPath"`
	Groups              []Group        `json:"groups,omitempty"              validate:"dive"`
}

// Group is a multi-repo coordination unit. Override fields fall back to the
// top-level Config equivalents when nil/empty. Override merging happens in
// internal/preflight.BuildPlan, not here.
type Group struct {
	Name                string         `json:"name"                          validate:"required"`
	Repos               []string       `json:"repos"                         validate:"min=2,dive,abspath"`
	SymlinkDirectories  []string       `json:"symlinkDirectories,omitempty"`
	WorktreePathPattern string         `json:"worktreePathPattern,omitempty" validate:"omitempty,containsName"`
	LaunchCommand       []string       `json:"launchCommand,omitempty"       validate:"omitempty,min=1"`
	Agent               agents.AgentID `json:"agent,omitempty"               validate:"omitempty,oneof=claude codex cursor code opencode qoder"`
	AddDirArgs          []string       `json:"addDirArgs,omitempty"          validate:"omitempty,min=1,containsPath"`
}
