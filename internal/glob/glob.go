// Package glob expands symlink-directory items, mixing literal pass-through
// with stdlib-based glob matching. Items without glob metacharacters are
// returned unchanged; items containing "*", "?", "[", "]", "{", "}" are
// matched against the repo tree using a hand-rolled recursive matcher
// supporting the doublestar (**) pattern. Forward slashes only in output.
package glob

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// globChars is the set of metacharacters that route an item to the glob
// branch in ExpandSymlinkItems. Mirrors the TS GLOB_CHARS regex
// /[*?[\]{}]/ exactly. Note that {} routes to the glob branch even though
// the matcher does not perform brace expansion (see Phase 2 design §3
// "Known parity gap").
const globChars = "*?[]{}"

// hasGlobChars reports whether s contains any glob metacharacter.
func hasGlobChars(s string) bool {
	return strings.ContainsAny(s, globChars)
}

// matchSegs reports whether the file-path segments fSegs match the pattern
// segments pSegs. A pattern segment of "**" matches zero or more file
// segments greedily but the algorithm tries every position, so
// {"a", "**", "b"} matches {"a", "b"} (zero), {"a", "x", "b"} (one), and
// {"a", "x", "y", "b"} (two). Non-** segments delegate to stdlib path.Match,
// which supports "*", "?", "[...]", and "\" escapes. Returns the path.Match
// error verbatim on malformed patterns (e.g. unclosed "[").
func matchSegs(pSegs, fSegs []string) (bool, error) {
	if len(pSegs) == 0 {
		return len(fSegs) == 0, nil
	}
	if pSegs[0] == "**" {
		for i := 0; i <= len(fSegs); i++ {
			ok, err := matchSegs(pSegs[1:], fSegs[i:])
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if len(fSegs) == 0 {
		return false, nil
	}
	ok, err := path.Match(pSegs[0], fSegs[0])
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return matchSegs(pSegs[1:], fSegs[1:])
}

// matchPattern is a single-string convenience over matchSegs. Both arguments
// are split on "/" — relPath should already be in forward-slash form (the
// caller normalizes via filepath.ToSlash in walkAndMatch).
func matchPattern(pattern, relPath string) (bool, error) {
	return matchSegs(strings.Split(pattern, "/"), strings.Split(relPath, "/"))
}

// walkAndMatch returns every relative path under root (forward-slash form)
// that matches the given pattern. Walks the tree with filepath.WalkDir,
// which uses os.Lstat (does not follow symlinks — safe against loops).
// Hidden files and directories are included; matching is the caller's call.
func walkAndMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		ok, err := matchPattern(pattern, rel)
		if err != nil {
			return err
		}
		if ok {
			matches = append(matches, rel)
		}
		return nil
	})
	return matches, err
}

// ExpandSymlinkItems resolves a list of symlink-directory items relative to repo.
// Items without glob metacharacters pass through unchanged. Items containing
// "*", "?", "[", "]", "{", "}" are matched against the repo tree, with results
// rendered as forward-slash relative paths. Duplicates (literal + overlapping
// glob) are dropped, preserving first-seen order. The empty match is an empty
// (non-nil) []string.
func ExpandSymlinkItems(repo string, items []string) ([]string, error) {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		if !hasGlobChars(item) {
			if !seen[item] {
				seen[item] = true
				out = append(out, item)
			}
			continue
		}
		matches, err := walkAndMatch(repo, item)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	return out, nil
}
