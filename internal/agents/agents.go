// Package agents holds the agent registry and the logic for deciding how to
// inject sibling-repo paths into the launched agent's argv.
package agents

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AgentID is one of the agents wtmux knows how to launch out of the box.
type AgentID string

const (
	AgentClaude   AgentID = "claude"
	AgentCodex    AgentID = "codex"
	AgentCursor   AgentID = "cursor"
	AgentVSCode   AgentID = "code"
	AgentOpenCode AgentID = "opencode"
	AgentQoder    AgentID = "qoder"
)

// StrategyKind describes how (or whether) wtmux injects sibling paths into the
// launched agent's argv.
type StrategyKind int

const (
	// StrategyFlag prepends a flag-template per sibling, e.g. "--add-dir <path>".
	StrategyFlag StrategyKind = iota
	// StrategyNone means the agent has no multi-root support; wtmux skips the launch.
	StrategyNone
	// StrategyPositional is the fallback when wtmux can't identify the agent —
	// caller decides what (if anything) to do.
	StrategyPositional
)

// Strategy is the per-agent injection rule.
type Strategy struct {
	Kind StrategyKind
	// Args is the flag template; "{path}" is replaced with each sibling's
	// absolute path. Populated only when Kind == StrategyFlag.
	Args []string
}

// Registry maps every known AgentID to its injection Strategy. Read-only at
// runtime — do not mutate after package init.
var Registry = map[AgentID]Strategy{
	AgentClaude:   {Kind: StrategyFlag, Args: []string{"--add-dir", "{path}"}},
	AgentCodex:    {Kind: StrategyFlag, Args: []string{"--add-dir", "{path}"}},
	AgentCursor:   {Kind: StrategyFlag, Args: []string{"--add", "{path}"}},
	AgentVSCode:   {Kind: StrategyFlag, Args: []string{"--add", "{path}"}},
	AgentOpenCode: {Kind: StrategyNone},
	AgentQoder:    {Kind: StrategyNone},
}

// basenameAlias maps a launch command's basename to a canonical AgentID when
// the names don't match (e.g., qoder's CLI is named "qodercli").
var basenameAlias = map[string]AgentID{
	"qodercli": AgentQoder,
}

// ResolvedStrategySource indicates which input drove the resolution decision.
// Used in warnings and verbose logging.
type ResolvedStrategySource string

const (
	SourceAddDirArgs ResolvedStrategySource = "addDirArgs"
	SourceAgent      ResolvedStrategySource = "agent"
	SourceBasename   ResolvedStrategySource = "basename"
	SourceFallback   ResolvedStrategySource = "fallback"
)

// ResolvedStrategy is what ResolveStrategy returns. Per-Kind field map:
//
//	StrategyFlag       → FlagArgs populated; AgentID empty
//	StrategyNone       → AgentID populated; FlagArgs nil
//	StrategyPositional → both empty; caller decides
//
// Source is always populated.
type ResolvedStrategy struct {
	Kind     StrategyKind
	FlagArgs []string // populated when Kind == StrategyFlag
	Source   ResolvedStrategySource
	// AgentID is set when Kind == StrategyNone (so the caller can name the
	// agent in its "skipping launch" message).
	AgentID AgentID
}

// ResolveInput collects the inputs ResolveStrategy needs.
type ResolveInput struct {
	LaunchCommand []string
	Agent         AgentID  // optional, "" if unset
	AddDirArgs    []string // optional, nil if unset
	// Warn, if non-nil, receives advisory messages (e.g. addDirArgs overriding agent).
	Warn func(msg string)
}

// ResolveStrategy decides how to inject sibling-repo paths into the launched
// agent's argv. Precedence:
//
//  1. Explicit addDirArgs (with optional warn if an agent is also set)
//  2. Explicit agent
//  3. Basename of LaunchCommand[0], possibly via basenameAlias
//  4. Positional fallback — caller chooses behavior
func ResolveStrategy(in ResolveInput) ResolvedStrategy {
	if len(in.AddDirArgs) > 0 {
		if in.Agent != "" && in.Warn != nil {
			in.Warn(fmt.Sprintf("addDirArgs overrides agent %q", in.Agent))
		}
		return ResolvedStrategy{
			Kind:     StrategyFlag,
			FlagArgs: in.AddDirArgs,
			Source:   SourceAddDirArgs,
		}
	}

	if in.Agent != "" {
		if s, ok := Registry[in.Agent]; ok {
			return resolvedFrom(in.Agent, s, SourceAgent)
		}
		// Unknown agent: bare Registry[in.Agent] would return a zero-value
		// Strategy whose Kind == StrategyFlag (iota 0) with FlagArgs nil,
		// which silently drops every sibling at expansion time. Warn and
		// fall through to basename detection instead.
		if in.Warn != nil {
			in.Warn(fmt.Sprintf("agent %q is not registered; falling back to basename detection", in.Agent))
		}
	}

	if len(in.LaunchCommand) > 0 {
		base := filepath.Base(in.LaunchCommand[0])
		resolved := AgentID(base)
		if alias, ok := basenameAlias[base]; ok {
			resolved = alias
		}
		if s, ok := Registry[resolved]; ok {
			return resolvedFrom(resolved, s, SourceBasename)
		}
	}

	return ResolvedStrategy{Kind: StrategyPositional, Source: SourceFallback}
}

func resolvedFrom(id AgentID, s Strategy, src ResolvedStrategySource) ResolvedStrategy {
	if s.Kind == StrategyFlag {
		return ResolvedStrategy{Kind: StrategyFlag, FlagArgs: s.Args, Source: src}
	}
	return ResolvedStrategy{Kind: StrategyNone, Source: src, AgentID: id}
}

// ExpandAddDirArgs expands a flag-template across siblings. For each sibling,
// every token in the template is rendered (each occurrence of "{path}" is
// replaced with the sibling's absolute path), and the rendered tokens are
// appended in template order. Returns an empty slice when siblings is empty.
func ExpandAddDirArgs(template, siblings []string) []string {
	if len(siblings) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(template)*len(siblings))
	for _, sib := range siblings {
		for _, tok := range template {
			out = append(out, strings.ReplaceAll(tok, "{path}", sib))
		}
	}
	return out
}
