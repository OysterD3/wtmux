package tui

import (
	"os"
	"path/filepath"
)

// isGitRepo reports whether dir contains a .git entry (file or dir).
func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

// collectGitRepos returns immediate child directories of dir that are git
// repos, skipping exclude (when non-empty).
func collectGitRepos(dir, exclude string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(dir, e.Name())
		if exclude != "" && full == exclude {
			continue
		}
		if isGitRepo(full) {
			out = append(out, full)
		}
	}
	return out
}

// DetectSiblings returns absolute paths of git repos that are siblings of
// cwd (or, when cwd itself isn't a git repo, child repos of cwd). Mirrors
// src/tui/siblings.ts.
func DetectSiblings(cwd string) []string {
	resolved, err := filepath.Abs(cwd)
	if err != nil {
		return nil
	}
	if isGitRepo(resolved) {
		parent := filepath.Dir(resolved)
		if parent == resolved {
			return nil
		}
		return collectGitRepos(parent, resolved)
	}
	return collectGitRepos(resolved, "")
}
