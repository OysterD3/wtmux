// Package paths provides pure helpers for path manipulation: tilde expansion,
// absolute checks, worktree-name flattening, pattern interpolation, and a
// best-effort existence check.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandTilde expands a leading "~" or "~/" to the user's home directory.
// Mid-string tildes are not expanded. Returns the input unchanged if the home
// directory cannot be determined.
func ExpandTilde(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// FlattenWorktreeName replaces "/" with "-" so a branch name like "feat/login"
// becomes a single-segment worktree directory name "feat-login".
func FlattenWorktreeName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
}

// ExpandWorktreePath renders a worktree-path pattern by replacing "{name}"
// with the slash-flattened worktree name. If the resulting path is absolute,
// it is returned as-is; otherwise it is joined onto the repo root.
func ExpandWorktreePath(repo, pattern, name string) string {
	rendered := strings.ReplaceAll(pattern, "{name}", FlattenWorktreeName(name))
	if filepath.IsAbs(rendered) {
		return rendered
	}
	return filepath.Join(repo, rendered)
}

// Exists reports whether a filesystem entry exists at p, using lstat semantics
// (broken symlinks count as existing — matches the TS implementation's
// fs.lstat-based check).
func Exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// RealpathSafe returns filepath.EvalSymlinks(p), falling back to p itself on
// any error (most often ENOENT for an unmounted but configured path). Use
// this when comparing a config-supplied path against output from `git`,
// which always returns symlink-resolved paths.
func RealpathSafe(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}
