package tui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/paths"
)

// ValidateGroupName returns an error suitable for huh.Input.Validate when
// the given name is empty, too long, or duplicates an existing name. Returns
// nil for valid input.
func ValidateGroupName(existing []string, name string) error {
	t := strings.TrimSpace(name)
	if t == "" {
		return errors.New("Name cannot be empty")
	}
	if len(t) > 64 {
		return errors.New("Name is too long (max 64 characters)")
	}
	for _, n := range existing {
		if n == t {
			return fmt.Errorf("A group named %q already exists", t)
		}
	}
	return nil
}

// ValidateWorktreePattern returns an error if pattern is non-empty but
// missing the {name} placeholder. Empty pattern is valid (means "unset
// override" in TUI flows).
func ValidateWorktreePattern(pattern string) error {
	t := strings.TrimSpace(pattern)
	if t == "" {
		return nil
	}
	if !strings.Contains(t, "{name}") {
		return errors.New("Pattern must contain {name}")
	}
	return nil
}

// RepoPathResult is the result of ResolveRepoPath. Resolved is set on
// success and is the toplevel of the repo (i.e. what `git rev-parse
// --show-toplevel` reports). Reason is populated on failure.
type RepoPathResult struct {
	OK       bool
	Resolved string
	Reason   string
}

// ResolveRepoPath validates a user-entered repo path: tilde-expands it,
// rejects relative paths, then asks git for the worktree top-level. The
// returned Resolved is git's answer, not the user's input — so the caller
// stores a canonical absolute path in the config.
func ResolveRepoPath(raw string) RepoPathResult {
	t := strings.TrimSpace(raw)
	if t == "" {
		return RepoPathResult{Reason: "Path cannot be empty"}
	}
	expanded := paths.ExpandTilde(t)
	if !filepath.IsAbs(expanded) {
		return RepoPathResult{Reason: "Path must be absolute or start with ~/"}
	}
	top, ok, err := git.GetToplevel(expanded)
	if err != nil {
		return RepoPathResult{Reason: fmt.Sprintf("git failed: %s", err)}
	}
	if !ok {
		return RepoPathResult{Reason: fmt.Sprintf("Not a git repository: %s", expanded)}
	}
	return RepoPathResult{OK: true, Resolved: top}
}
